package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"main/internal/application/bookings"
	"main/internal/application/classes"
	"main/internal/application/location"
	"main/internal/application/passes"
	"main/internal/application/pendingbookings"
	"main/internal/application/reminder"
	"main/internal/domain/repositories"
	"main/internal/domain/services"
	"main/internal/infrastructure/configuration"
	"main/internal/infrastructure/generator/token"
	dbModels "main/internal/infrastructure/models/db"
	"main/internal/infrastructure/notifier/gmail"
	sqliteRepo "main/internal/infrastructure/repository/sqlite"
	"main/internal/infrastructure/sender"
	gmailSender "main/internal/infrastructure/sender/gmail"
	"main/internal/infrastructure/sender/memory"
	apiErrs "main/internal/interfaces/http/api/errs"
	apiErrHandler "main/internal/interfaces/http/api/errs/handler"
	"main/internal/interfaces/http/api/errs/logging"
	"main/internal/interfaces/http/api/handlers/activatepass"
	"main/internal/interfaces/http/api/handlers/createclasses"
	"main/internal/interfaces/http/api/handlers/createcontacts"
	"main/internal/interfaces/http/api/handlers/deletebooking"
	"main/internal/interfaces/http/api/handlers/deleteclass"
	"main/internal/interfaces/http/api/handlers/listbookings"
	"main/internal/interfaces/http/api/handlers/listbookingsbyclass"
	"main/internal/interfaces/http/api/handlers/listclasses"
	"main/internal/interfaces/http/api/handlers/listcontacts"
	"main/internal/interfaces/http/api/handlers/listpendingbookings"
	"main/internal/interfaces/http/api/handlers/updateclass"
	viewErrs "main/internal/interfaces/http/html/errs"
	viewErrHandler "main/internal/interfaces/http/html/errs/handler"
	logWrapper "main/internal/interfaces/http/html/errs/wrapper"
	"main/internal/interfaces/http/html/handlers/cancelbooking"
	"main/internal/interfaces/http/html/handlers/cancelbookingform"
	"main/internal/interfaces/http/html/handlers/createbooking"
	"main/internal/interfaces/http/html/handlers/emails"
	"main/internal/interfaces/http/html/handlers/errorpage"
	"main/internal/interfaces/http/html/handlers/home"
	creatependingbooking "main/internal/interfaces/http/html/handlers/pendingbooking"
	"main/internal/interfaces/http/html/handlers/pendingbookingform"
	"main/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"golang.org/x/time/rate"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Components struct {
	unitOfWork             repositories.IUnitOfWork
	classesService         services.IClassesService
	bookingsService        services.IBookingsService
	pendingBookingsService services.IPendingBookingsService
	passesService          services.IPassesService
	bookingsRepo           repositories.IBookings
	pendingBookingsRepo    repositories.IPendingBookings
	contactsRepo           repositories.IContacts
	reminder               reminder.IReminderService
	database               *gorm.DB
	storage                *memory.Storage
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("failed to loading configuration", slog.String("err", err.Error()))
		os.Exit(1)
	}

	components, err := buildComponents(cfg)
	if err != nil {
		slog.Error("failed to build components", slog.String("err", err.Error()))
		os.Exit(1)
	}

	router := setupRouter(
		components.bookingsService,
		components.classesService,
		components.pendingBookingsService,
		components.passesService,
		components.bookingsRepo,
		components.pendingBookingsRepo,
		components.contactsRepo,
		components.storage,
		cfg,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func(ctxTimeout time.Duration) {
		cleanupCtx, cancel := context.WithTimeout(ctx, ctxTimeout)
		defer cancel()

		cleanUpPendingBookings(cleanupCtx, components.database)
	}(cfg.ContextTimeout.Duration)

	go func(ctxTimeout time.Duration) {
		time.Sleep(2 * time.Second)

		reminderCtx, cancel := context.WithTimeout(ctx, ctxTimeout)
		defer cancel()

		remindBookings(reminderCtx, components.reminder)
	}(cfg.ContextTimeout.Duration)

	srv := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           router,
		ReadHeaderTimeout: cfg.ReadTimeout.Duration,
		ReadTimeout:       cfg.ReadTimeout.Duration,
		WriteTimeout:      cfg.WriteTimeout.Duration,
	}

	runServer(srv, cfg)
}

func loadConfig() (*configuration.Configuration, error) {
	cfg, err := configuration.GetConfig("./config")
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %w", err)
	}

	if cfg.LogConfig {
		slog.Info(cfg.Pretty())
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	return cfg, nil
}

func buildComponents(cfg *configuration.Configuration) (Components, error) {
	database, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return Components{}, fmt.Errorf("failed to connect to database: %w", err)
	}

	slog.Info("Successfully connected to database")

	err = database.AutoMigrate(
		&dbModels.SQLClass{},
		&dbModels.SQLPendingBooking{},
		&dbModels.SQLBooking{},
		&dbModels.SQLPass{},
		&dbModels.SQLContact{},
	)
	if err != nil {
		return Components{}, fmt.Errorf("failed to migrate database: %w", err)
	}

	classesRepo := sqliteRepo.NewClassesRepo(database)
	bookingsRepo := sqliteRepo.NewBookingsRepo(database)
	pendingBookingsRepo := sqliteRepo.NewPendingBookingsRepo(database)
	passesRepo := sqliteRepo.NewPassesRepo(database)
	contactsRepo := sqliteRepo.NewContactsRepo(database)

	tokenGenerator := token.NewGenerator()

	var sender sender.IEmailSender

	sender = gmailSender.NewSender(
		cfg.Notifier.Host,
		cfg.Notifier.Port,
		cfg.Notifier.Login,
		cfg.Notifier.Password,
	)

	memoryStorage := memory.Storage{
		Views: make([]string, 0),
	}

	if cfg.MockEmailSender {
		sender = memory.NewSender(&memoryStorage)
	}

	emailNotifier := gmail.NewNotifier(
		sender,
		cfg.Notifier.Login,
		cfg.Notifier.Signature,
		cfg.BaseNotifierTmplPath,
	)

	unitOfWork := sqliteRepo.NewUnitOfWork(database)
	passManager := services.PassManager{}

	locationResolver := location.NewMemoryResolver()

	classesService := classes.NewService(
		classesRepo,
		bookingsRepo,
		unitOfWork,
		&passManager,
		emailNotifier,
		locationResolver,
		cfg.DomainAddr,
	)
	bookingsService := bookings.NewService(
		unitOfWork,
		bookingsRepo,
		&passManager,
		emailNotifier,
		locationResolver,
		cfg.DomainAddr,
	)

	pendingBookingsService := pendingbookings.NewService(
		unitOfWork,
		tokenGenerator,
		emailNotifier,
		cfg.DomainAddr,
	)

	passesService := passes.NewService(passesRepo, bookingsRepo, emailNotifier, &passManager)

	reminder := reminder.New(
		unitOfWork,
		classesRepo,
		bookingsRepo,
		emailNotifier,
		&passManager,
		locationResolver,
		cfg.DomainAddr,
	)

	return Components{
		unitOfWork:             unitOfWork,
		classesService:         classesService,
		bookingsService:        bookingsService,
		pendingBookingsService: pendingBookingsService,
		passesService:          passesService,
		bookingsRepo:           bookingsRepo,
		pendingBookingsRepo:    pendingBookingsRepo,
		contactsRepo:           contactsRepo,
		reminder:               reminder,
		database:               database,
		storage:                &memoryStorage,
	}, nil
}

func setupRouter(
	bookingsService services.IBookingsService,
	classesService services.IClassesService,
	pendingBookingsService services.IPendingBookingsService,
	passesService services.IPassesService,
	bookingsRepo repositories.IBookings,
	pendingBookingsRepo repositories.IPendingBookings,
	contactsRepo repositories.IContacts,
	storage *memory.Storage,
	cfg *configuration.Configuration,
) *gin.Engine {
	router := gin.Default()

	router.Static("web/static", "./web/static")
	router.LoadHTMLGlob("web/templates/*")
	router.Use(middleware.RequestID())
	api := router.Group("/")

	var viewErrorHandler viewErrs.IErrorHandler

	viewErrorHandler = viewErrHandler.NewErrorHandler()
	viewErrorHandler = logWrapper.NewErrorHandler(viewErrorHandler, cfg.LogBusinessErrors)

	// HTML
	rateLimiterMiddleware := middleware.GlobalRateLimit

	homeHandler := home.NewHandler(classesService, viewErrorHandler, cfg.IsVacation)
	createBookingHandler := createbooking.NewHandler(bookingsService, viewErrorHandler)
	cancelBookingHandler := cancelbooking.NewHandler(bookingsService, viewErrorHandler)
	createPendingBookingHandler := creatependingbooking.NewHandler(pendingBookingsService, viewErrorHandler)
	pendingBookingFormHandler := pendingbookingform.NewHandler()
	cancelBookingFormHandler := cancelbookingform.NewHandler(bookingsService, viewErrorHandler)
	errorPageHandler := errorpage.NewHandler()
	emailsHander := emails.NewHandler(storage)

	{
		// home
		api.GET("/", homeHandler.Handle)

		// error page
		api.GET("/error", errorPageHandler.Handle)

		// bookings
		// this endpoint should be POST according to REST, it is GET - confirmation link sent via email
		api.GET("/bookings", createBookingHandler.Handle)
		api.DELETE("/bookings/:id", cancelBookingHandler.Handle)
		api.GET("/bookings/:id/cancel_form", cancelBookingFormHandler.Handle)

		// pending_bookings
		api.GET("/classes/:class_id/pending_bookings/form", pendingBookingFormHandler.Handle)

		requestLimiter := rate.NewLimiter(rate.Limit(1), 2)
		api.POST("/pending_bookings", rateLimiterMiddleware(requestLimiter), createPendingBookingHandler.Handle)

		// testing
		if cfg.MockEmailSender {
			api.GET("/emails", emailsHander.Handle)
		}
	}

	var apiErrorHandler apiErrs.IErrorHandler

	apiErrorHandler = apiErrHandler.NewErrorHandler()
	apiErrorHandler = logging.NewErrorHandler(apiErrorHandler)

	// API
	authMiddleware := middleware.Auth(cfg.AuthSecret)

	createClassHandler := createclasses.NewHandler(classesService, apiErrorHandler)
	getClassesHandler := listclasses.NewHandler(classesService, apiErrorHandler)
	updateClassHandler := updateclass.NewHandler(classesService, apiErrorHandler)
	deleteClassHandler := deleteclass.NewHandler(classesService, apiErrorHandler)
	listBookingsHandler := listbookings.NewHandler(bookingsRepo, apiErrorHandler)
	listBookingsByClassHandler := listbookingsbyclass.NewHandler(bookingsRepo, apiErrorHandler)
	deleteBookingHandler := deletebooking.NewHandler(bookingsService, apiErrorHandler)
	listPendingBookingsHandler := listpendingbookings.NewHandler(pendingBookingsRepo, apiErrorHandler)
	activatePassHandler := activatepass.NewHandler(passesService, apiErrorHandler)
	listContactsHandler := listcontacts.NewHandler(contactsRepo, apiErrorHandler)
	createContactsHandler := createcontacts.NewHandler(contactsRepo, apiErrorHandler)

	{
		api.GET("/api/v1/bookings", authMiddleware, listBookingsHandler.Handle)
		api.DELETE("/api/v1/bookings/:booking_id", authMiddleware, deleteBookingHandler.Handle)
		api.GET("api/v1/pending_bookings", authMiddleware, listPendingBookingsHandler.Handle)
		api.POST("/api/v1/classes", authMiddleware, createClassHandler.Handle)
		api.GET("/api/v1/classes", authMiddleware, getClassesHandler.Handle)
		api.PATCH("/api/v1/classes/:class_id", authMiddleware, updateClassHandler.Handle)
		api.DELETE("/api/v1/classes/:class_id", authMiddleware, deleteClassHandler.Handle)
		api.GET("/api/v1/classes/:class_id/bookings", authMiddleware, listBookingsByClassHandler.Handle)
		api.PUT("/api/v1/passes", authMiddleware, activatePassHandler.Handle)
		api.GET("/api/v1/contacts", authMiddleware, listContactsHandler.Handle)
		api.POST("/api/v1/contacts", authMiddleware, createContactsHandler.Handle)
	}

	return router
}

func runServer(srv *http.Server, cfg *configuration.Configuration) {
	go func() {
		slog.Info("Starting server...", slog.String("address", cfg.ListenAddress))

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", slog.String("err", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ContextTimeout.Duration)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", slog.String("err", err.Error()))
	}

	slog.Info("Server stopped")
}

func cleanUpPendingBookings(ctx context.Context, database *gorm.DB) {
	oneHourAgo := time.Now().UTC().Add(-1 * time.Hour)

	var pendingBooking dbModels.SQLPendingBooking

	result := database.WithContext(ctx).
		Where("created_at < ?", oneHourAgo).
		Delete(&pendingBooking)

	if result.Error != nil {
		if errors.Is(result.Error, context.DeadlineExceeded) {
			slog.Warn("cleanup timeout exceeded")
		} else {
			slog.Error("failed to cleanup pending bookings async",
				slog.String("err", result.Error.Error()))
		}

		return
	}

	slog.Info("PendingBookingCleaner: cleaned up pending bookings", slog.Int64("deleted", result.RowsAffected))
}

func remindBookings(ctx context.Context, reminder reminder.IReminderService) {
	err := reminder.RemindBookings(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Warn("remind classes timeout exceeded")
		} else {
			slog.Error("failed to remind classes async",
				slog.String("err", err.Error()))
		}
	}
}
