package reminder

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"main/internal/application"
	"main/internal/domain/models"
	"main/internal/domain/notifier"
	"main/internal/domain/repositories"

	"github.com/google/uuid"
)

type IReminderService interface {
	RemindBookings(ctx context.Context) error
}

type service struct {
	unitOfWork       repositories.IUnitOfWork
	classesRepo      repositories.IClasses
	bookingsRepo     repositories.IBookings
	notifier         notifier.INotifier
	passManager      application.IPassManager
	locationResolver application.ILocationResolver
	domainAddr       string
}

func New(
	unitOfWork repositories.IUnitOfWork,
	classesRepo repositories.IClasses,
	bookingsRepo repositories.IBookings,
	notifier notifier.INotifier,
	passManager application.IPassManager,
	locationResolver application.ILocationResolver,
	domainAddr string,
) *service {
	return &service{
		unitOfWork:       unitOfWork,
		classesRepo:      classesRepo,
		bookingsRepo:     bookingsRepo,
		notifier:         notifier,
		passManager:      passManager,
		locationResolver: locationResolver,
		domainAddr:       domainAddr,
	}
}

func (s *service) RemindBookings(ctx context.Context) error {
	slog.Info("Reminder: searching bookings...")

	classes, err := s.classesRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("could not list classes: %w", err)
	}

	now := time.Now()

	futureClasses := make([]models.Class, 0)

	for _, class := range classes {
		if class.StartTime.After(now) {
			futureClasses = append(futureClasses, class)
		}
	}

	for _, class := range futureClasses {
		if isTimeToRemind(class.StartTime, now) {
			err := s.sendReminders(ctx, class.ID, class.StartTime)
			if err != nil {
				return fmt.Errorf("could not send reminders for class %v: %w", class.ID, err)
			}

			continue
		}

		slog.Info(
			"Reminder: to early to remind class", "class_id", class.ID, "start_time", class.StartTime,
		)
	}

	slog.Info("Reminder: searching bookings done!")

	return nil
}

func isTimeToRemind(classStartTime, now time.Time) bool {
	diff := classStartTime.Sub(now)

	return diff > 0 && diff < 24*time.Hour
}

func (s *service) sendReminders(
	ctx context.Context,
	classID uuid.UUID,
	classStartTime time.Time,
) error {
	bookings, err := s.bookingsRepo.ListByClassID(ctx, classID)
	if err != nil {
		return fmt.Errorf("could not list bookings for %v: %w", classID, err)
	}

	if len(bookings) == 0 {
		slog.Info("Reminder: no bookings for class", "class_id", classID)

		return nil
	}

	slog.Info(fmt.Sprintf("Reminder: found bookings: %d", len(bookings)), "class_id", classID)

	for _, booking := range bookings {
		if !shouldRemindBooking(booking, classStartTime) {
			continue
		}

		err := s.remindBooking(ctx, booking)
		if err != nil {
			return fmt.Errorf("could not remind about booking %v: %w", booking.ID, err)
		}
	}

	return nil
}

func (s *service) remindBooking(ctx context.Context, booking models.Booking) error {
	err := s.unitOfWork.WithTransaction(ctx, func(repos repositories.Repositories) error {
		update := map[string]any{"reminded_at": time.Now()}

		err := repos.Bookings.Update(ctx, booking.ID, update)
		if err != nil {
			return fmt.Errorf("could not update booking %v with %v: %w", booking.ID, update, err)
		}

		locationLink, err := s.locationResolver.GetLink(booking.Class.Location)
		if err != nil {
			return fmt.Errorf("could not get location link for %v: %w", booking.Class.Location, err)
		}

		notifierParams := models.NotifierParams{
			RecipientEmail:     booking.Email,
			RecipientFirstName: booking.FirstName,
			RecipientLastName:  booking.LastName,
			ClassName:          booking.Class.ClassName,
			ClassLevel:         booking.Class.ClassLevel,
			StartTime:          booking.Class.StartTime,
			Location:           booking.Class.Location,
			LocationLink:       locationLink,
		}

		if booking.Pass.Exists() {
			pass := booking.Pass.Get()

			usedBookings, err := repos.Bookings.ListByPassID(ctx, pass.ID)
			if err != nil {
				return fmt.Errorf("could not list bookings for pass %v: %w", pass.ID, err)
			}

			passSlots := s.passManager.BuildPassSlots(usedBookings, pass.TotalSlots)

			notifierParams.PassSlots = passSlots
		}

		cancellationLink := fmt.Sprintf(
			"%s/bookings/%s/cancel_form?token=%s", s.domainAddr, booking.ID, booking.ConfirmationToken,
		)

		err = s.notifier.NotifyBookingReminder(notifierParams, cancellationLink)
		if err != nil {
			return fmt.Errorf("could not nofify booking with %v: %w", notifierParams, err)
		}

		slog.Info("Reminder: booking reminded",
			"email", booking.Email, "reminded_at", update["reminded_at"],
		)

		return nil
	})
	if err != nil {
		return fmt.Errorf("remind booking transaction failed: %w", err)
	}

	return nil
}

func shouldRemindBooking(booking models.Booking, classStartTime time.Time) bool {
	if booking.RemindedAt != nil {
		slog.Info(
			"Reminder: skipping already reminded booking",
			"booking_id", booking.ID, "email", booking.Email,
		)

		return false
	}

	if isBookedSameOrPreviousDayAsClassDay(booking.CreatedAt, classStartTime) {
		slog.Info(
			"Reminder: skipping booking created at the same or previous day as class day",
			"email", booking.Email, "created_at", booking.CreatedAt, "class_start_time", classStartTime,
		)

		return false
	}

	return true
}

func isBookedSameOrPreviousDayAsClassDay(bookingCreatedAt, classStartTime time.Time) bool {
	if bookingCreatedAt.IsZero() || classStartTime.IsZero() {
		return false
	}

	aDate := time.Date(
		bookingCreatedAt.Year(), bookingCreatedAt.Month(), bookingCreatedAt.Day(), 0, 0, 0, 0, time.UTC,
	)
	bDate := time.Date(
		classStartTime.Year(), classStartTime.Month(), classStartTime.Day(), 0, 0, 0, 0, time.UTC,
	)

	prevDay := bDate.Add(-24 * time.Hour)

	return aDate.Equal(bDate) || aDate.Equal(prevDay)
}
