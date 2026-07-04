package bookings

import (
	"context"
	"errors"
	"fmt"
	"time"

	"main/internal/application/location"
	appModels "main/internal/application/models"
	viewErrors "main/internal/domain/errs/view"
	"main/internal/domain/models"
	"main/internal/domain/notifier"
	"main/internal/domain/repositories"
	"main/internal/domain/services"
	"main/internal/infrastructure/errs"
	"main/pkg/optional"

	"github.com/google/uuid"
)

const (
	threeLastPasses = 3
)

type service struct {
	unitOfWork       repositories.IUnitOfWork
	bookingsRepo     repositories.IBookings
	passManager      services.IPassManager
	notifier         notifier.INotifier
	loactionResolver location.IResolver
	domainAddr       string
}

func NewService(
	unitOfWork repositories.IUnitOfWork,
	bookingsRepo repositories.IBookings,
	passManager services.IPassManager,
	notifier notifier.INotifier,
	locationResolver location.IResolver,
	domainAddr string,
) *service {
	return &service{
		unitOfWork:       unitOfWork,
		bookingsRepo:     bookingsRepo,
		passManager:      passManager,
		notifier:         notifier,
		loactionResolver: locationResolver,
		domainAddr:       domainAddr,
	}
}

func (s *service) CreateBooking(ctx context.Context, token string) (appModels.BookingCreation, error) {
	var (
		pendingBooking models.PendingBooking
		bookingID      uuid.UUID
		passSlots      []models.PassSlot
	)

	err := s.unitOfWork.WithTransaction(ctx, func(repos repositories.Repositories) error {
		var err error

		pendingBooking, err = repos.PendingBookings.GetByConfirmationToken(ctx, token)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				return viewErrors.ErrPendingBookingNotFound(
					fmt.Errorf("pending booking for token: %s not found", token),
				)
			}

			return fmt.Errorf("could not get pending booking: %w", err)
		}

		_, err = repos.Bookings.GetByEmailAndClassID(ctx, pendingBooking.ClassID, pendingBooking.Email)
		if err == nil {
			return viewErrors.ErrBookingAlreadyExists(
				pendingBooking.ClassID,
				pendingBooking.Email,
				fmt.Errorf("booking already exists for email %s and classID %s", pendingBooking.Email, pendingBooking.ClassID),
			)
		}

		if !errors.Is(err, errs.ErrNotFound) {
			return fmt.Errorf("could not get booking for email %s and classID %s: %w",
				pendingBooking.Email,
				pendingBooking.ClassID,
				err,
			)
		}

		err = s.checkClassAvailability(ctx, repos, pendingBooking.Class)
		if err != nil {
			return fmt.Errorf("class unavailable: %w", err)
		}

		// I need to make sure that I will check if previous pass will not have some empty slots. Three is enough.
		passes, err := repos.Passes.ListByEmail(ctx, pendingBooking.Email, threeLastPasses)
		if err != nil {
			return fmt.Errorf("could not get pass: %w", err)
		}

		booking := models.Booking{
			ID:                uuid.New(),
			ClassID:           pendingBooking.ClassID,
			FirstName:         pendingBooking.FirstName,
			LastName:          pendingBooking.LastName,
			Email:             pendingBooking.Email,
			CreatedAt:         time.Now().UTC(),
			ConfirmationToken: pendingBooking.ConfirmationToken,
		}

		_, err = repos.Contacts.Insert(ctx, booking.Email, booking.FirstName, booking.LastName)
		if err != nil {
			if !errors.Is(err, errs.ErrAlreadyExist) {
				return fmt.Errorf("could not insert contact: %w", err)
			}
		}

		for _, pass := range passes {
			bookingsWithPassCount, err := s.bookingsRepo.CountForPassID(ctx, pass.ID)
			if err != nil {
				return fmt.Errorf("could not count bookings for passID %d: %w", pass.ID, err)
			}

			if bookingsWithPassCount < pass.TotalSlots {
				booking.PassID = optional.Of(pass.ID)
				booking.Pass = optional.Of(pass)

				bookingID, err = repos.Bookings.Insert(ctx, booking)
				if err != nil {
					return fmt.Errorf("could not insert booking: %w", err)
				}

				bookings, err := repos.Bookings.ListByPassID(ctx, pass.ID)
				if err != nil {
					return fmt.Errorf("could not list bookings by passID %d: %w", pass.ID, err)
				}

				passSlots = s.passManager.BuildPassSlots(bookings, pass.TotalSlots)

				return nil
			}
		}

		bookingID, err = repos.Bookings.Insert(ctx, booking)
		if err != nil {
			return fmt.Errorf("could not insert booking: %w", err)
		}

		return nil
	})
	if err != nil {
		return appModels.BookingCreation{}, fmt.Errorf("create booking transaction failed: %w", err)
	}

	locationLink, err := s.loactionResolver.GetLink(pendingBooking.Class.Location)
	if err != nil {
		return appModels.BookingCreation{}, fmt.Errorf("could not resolve location link for location: %s", pendingBooking.Class.Location)
	}

	err = s.sendConfirmation(
		pendingBooking, passSlots, locationLink, token, bookingID,
	)
	if err != nil {
		return appModels.BookingCreation{},
			fmt.Errorf("could not send confirmation email %s: %w", pendingBooking.Email, err)
	}

	return appModels.BookingCreation{
		Class: appModels.ClassPresentation{
			ID:           pendingBooking.Class.ID,
			StartTime:    pendingBooking.Class.StartTime,
			ClassLevel:   pendingBooking.Class.ClassLevel,
			ClassName:    pendingBooking.Class.ClassName,
			MaxCapacity:  pendingBooking.Class.MaxCapacity,
			Location:     pendingBooking.Class.Location,
			LocationLink: locationLink,
		},
	}, nil
}

func (s *service) checkClassAvailability(
	ctx context.Context,
	repos repositories.Repositories,
	class models.Class,
) error {
	if class.StartTime.Before(time.Now()) {
		return viewErrors.ErrClassExpired(class.ID, fmt.Errorf("class %s has expired at %v", class.ID, class.StartTime))
	}

	bookingCount, err := repos.Bookings.CountForClassID(ctx, class.ID)
	if err != nil {
		return fmt.Errorf("could not count bookings for class %v: %w ", class.ID, err)
	}

	if bookingCount == class.MaxCapacity {
		return viewErrors.ErrSomeoneBookedClassFaster(fmt.Errorf("max capacity of class %d exceeded", class.MaxCapacity))
	}

	return nil
}

func (s *service) sendConfirmation(
	pendingBooking models.PendingBooking,
	passSlots []models.PassSlot,
	locationLink string,
	token string,
	bookingID uuid.UUID,
) error {
	notifierParams := models.NotifierParams{
		RecipientEmail:     pendingBooking.Email,
		RecipientFirstName: pendingBooking.FirstName,
		RecipientLastName:  pendingBooking.LastName,
		ClassName:          pendingBooking.Class.ClassName,
		ClassLevel:         pendingBooking.Class.ClassLevel,
		StartTime:          pendingBooking.Class.StartTime,
		Location:           pendingBooking.Class.Location,
		LocationLink:       locationLink,
		PassSlots:          passSlots,
	}

	cancellationLink := fmt.Sprintf(
		"%s/bookings/%s/cancel_form?token=%s", s.domainAddr, bookingID, token,
	)

	err := s.notifier.NotifyBookingConfirmation(notifierParams, cancellationLink)
	if err != nil {
		return fmt.Errorf("could not notify booking confirmation: %w", err)
	}

	return nil
}

func (s *service) CancelBooking(ctx context.Context, bookingID uuid.UUID, token string) error {
	var (
		booking   models.Booking
		passSlots []models.PassSlot
	)

	err := s.unitOfWork.WithTransaction(ctx, func(repos repositories.Repositories) error {
		var err error

		booking, err = s.ensureBookingCancellationAllowed(ctx, repos, bookingID, token)
		if err != nil {
			return fmt.Errorf("booking cancellation not allowed for bookingID %s: %w", bookingID, err)
		}

		err = repos.Bookings.Delete(ctx, bookingID)
		if err != nil {
			if errors.Is(err, errs.ErrNoRowsAffected) {
				return viewErrors.ErrBookingNotFound(
					fmt.Errorf("delete booking failure, booking with email %s for class %s not found", booking.Email, booking.ClassID),
				)
			}

			return fmt.Errorf("could not delete booking: %w", err)
		}

		if booking.Pass.Exists() {
			pass := booking.Pass.Get()

			usedBookings, err := repos.Bookings.ListByPassID(ctx, pass.ID)
			if err != nil {
				return fmt.Errorf("could not list bookings by pass id %d: %w", pass.ID, err)
			}

			passSlots = s.passManager.BuildPassSlots(usedBookings, pass.TotalSlots)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("cancel booking transaction failed: %w", err)
	}

	locationLink, err := s.loactionResolver.GetLink(booking.Class.Location)
	if err != nil {
		return fmt.Errorf("could not resolve location link for: %s", booking.Class.Location)
	}

	notifierParams := models.NotifierParams{
		RecipientFirstName: booking.FirstName,
		RecipientLastName:  booking.LastName,
		RecipientEmail:     booking.Email,
		ClassName:          booking.Class.ClassName,
		ClassLevel:         booking.Class.ClassLevel,
		StartTime:          booking.Class.StartTime,
		Location:           booking.Class.Location,
		LocationLink:       locationLink,
		PassSlots:          passSlots,
	}

	err = s.notifier.NotifyBookingCancellation(notifierParams)
	if err != nil {
		return fmt.Errorf("could not notify booking cancellation with %+v: %w", notifierParams, err)
	}

	return nil
}

func (s *service) ensureBookingCancellationAllowed(
	ctx context.Context, r repositories.Repositories, bookingID uuid.UUID, token string,
) (models.Booking, error) {
	booking, err := r.Bookings.GetByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return models.Booking{}, viewErrors.ErrBookingNotFound(
				fmt.Errorf("booking with id %s not found: %w", bookingID, err),
			)
		}

		return models.Booking{}, fmt.Errorf("could not get booking for id %s: %w", bookingID, err)
	}

	if booking.ConfirmationToken != token {
		return models.Booking{}, viewErrors.ErrInvalidCancellationLink(
			fmt.Errorf("cancel booking failed due to invalid token: %s for email: %s", booking.Email, token),
		)
	}

	if booking.Class.StartTime.Before(time.Now()) {
		return models.Booking{}, viewErrors.ErrClassExpired(
			booking.Class.ID,
			fmt.Errorf("class %s has expired at %v", booking.ClassID, booking.Class.StartTime),
		)
	}

	return booking, nil
}

func (s *service) GetBookingCancellationForm(
	ctx context.Context, bookingID uuid.UUID, token string,
) (appModels.BookingCancellationForm, error) {
	booking, err := s.bookingsRepo.GetByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return appModels.BookingCancellationForm{}, viewErrors.ErrBookingNotFound(
				fmt.Errorf("booking with id %s not found: %w", bookingID, err),
			)
		}

		return appModels.BookingCancellationForm{},
			fmt.Errorf("could not get booking for id %s: %w", bookingID, err)
	}

	if booking.ConfirmationToken != token {
		return appModels.BookingCancellationForm{}, viewErrors.ErrInvalidCancellationLink(err)
	}

	class := appModels.BookingCancellationClass{
		ID:         booking.Class.ID,
		StartTime:  booking.Class.StartTime,
		ClassLevel: booking.Class.ClassLevel,
		ClassName:  booking.Class.ClassName,
		Location:   booking.Class.Location,
	}

	cancellationForm := appModels.BookingCancellationForm{
		Class:             class,
		BookingID:         booking.ID,
		ConfirmationToken: booking.ConfirmationToken,
	}

	return cancellationForm, nil
}

func (s *service) DeleteBooking(ctx context.Context, bookingID uuid.UUID) error {
	var (
		booking   models.Booking
		passSlots []models.PassSlot
	)

	err := s.unitOfWork.WithTransaction(ctx, func(repos repositories.Repositories) error {
		var err error

		booking, err = repos.Bookings.GetByID(ctx, bookingID)
		if err != nil {
			return fmt.Errorf("could get booking for id %s: %w", bookingID, err)
		}

		err = repos.Bookings.Delete(ctx, bookingID)
		if err != nil {
			return fmt.Errorf("could not delete booking for id %s: %w", bookingID, err)
		}

		if booking.Pass.Exists() {
			pass := booking.Pass.Get()
			usedBookings, err := repos.Bookings.ListByPassID(ctx, pass.ID)
			if err != nil {
				return fmt.Errorf("could not list bookings by pass id %d: %w", pass.ID, err)
			}

			passSlots = s.passManager.BuildPassSlots(usedBookings, pass.TotalSlots)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("delete booking transaction failed: %w", err)
	}

	locationLink, err := s.loactionResolver.GetLink(booking.Class.Location)
	if err != nil {
		return fmt.Errorf("could not resolve location link for location: %s", booking.Class.Location)
	}

	notifierParams := models.NotifierParams{
		RecipientFirstName: booking.FirstName,
		RecipientLastName:  booking.LastName,
		RecipientEmail:     booking.Email,
		ClassName:          booking.Class.ClassName,
		ClassLevel:         booking.Class.ClassLevel,
		StartTime:          booking.Class.StartTime,
		Location:           booking.Class.Location,
		LocationLink:       locationLink,
		PassSlots:          passSlots,
	}

	err = s.notifier.NotifyBookingCancellation(notifierParams)
	if err != nil {
		return fmt.Errorf("could not nofify booking cancellation with %+v: %w", notifierParams, err)
	}

	return nil
}
