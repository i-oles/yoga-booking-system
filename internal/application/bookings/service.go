package bookings

import (
	"context"
	"errors"
	"fmt"
	"time"

	"main/internal/application/classes"
	"main/internal/application/location"
	viewErrors "main/internal/domain/errs/view"
	"main/internal/domain/models"
	"main/internal/domain/notifier"
	"main/internal/domain/repositories"
	"main/internal/domain/services/passes"
	"main/internal/infrastructure/errs"
	"main/pkg/optional"

	"github.com/google/uuid"
)

const (
	threeLastPasses = 3
)

type service struct {
	unitOfWork           repositories.IUnitOfWork
	bookingsRepo         repositories.IBookings
	notifier             notifier.INotifier
	locationLinkProvider location.ILinkProvider
	domainAddr           string
}

func NewService(
	unitOfWork repositories.IUnitOfWork,
	bookingsRepo repositories.IBookings,
	notifier notifier.INotifier,
	locationLinkProvider location.ILinkProvider,
	domainAddr string,
) *service {
	return &service{
		unitOfWork:           unitOfWork,
		bookingsRepo:         bookingsRepo,
		notifier:             notifier,
		locationLinkProvider: locationLinkProvider,
		domainAddr:           domainAddr,
	}
}

func (s *service) CreateBooking(ctx context.Context, token string) (BookingCreation, error) {
	var (
		booking   models.Booking
		passSlots []models.PassSlot
	)

	err := s.unitOfWork.WithTransaction(ctx, func(repos repositories.Repositories) error {
		var err error

		pendingBooking, err := s.validatePendingBooking(ctx, token, repos)
		if err != nil {
			return fmt.Errorf("could not validate pendingBooking: %w", err)
		}

		err = s.saveContact(ctx, repos, pendingBooking)
		if err != nil {
			return fmt.Errorf("could not insert contact: %w", err)
		}

		booking, err = s.createBooking(ctx, pendingBooking, repos)
		if err != nil {
			return fmt.Errorf("could not insert booking: %w", err)
		}

		if booking.Pass.Exists() {
			passSlots, err = s.buildPassSlots(ctx, booking.Pass.Get(), repos)
			if err != nil {
				return fmt.Errorf("could not build passSlots: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return BookingCreation{}, fmt.Errorf("create booking transaction failed: %w", err)
	}

	locationLink, err := s.locationLinkProvider.GetLink(booking.Class.Location)
	if err != nil {
		return BookingCreation{},
			fmt.Errorf("could not resolve location link for: %s", booking.Class.Location)
	}

	notifierParams, err := s.buildNotifierParams(booking, locationLink, passSlots)
	if err != nil {
		return BookingCreation{}, fmt.Errorf("could not build notifierParams: %w", err)
	}

	err = s.sendConfirmation(booking, notifierParams, token)
	if err != nil {
		return BookingCreation{},
			fmt.Errorf("could not send confirmation email %s: %w", booking.Email, err)
	}

	return buildBookingCreation(booking, locationLink), nil
}

func (s *service) validatePendingBooking(
	ctx context.Context,
	token string,
	repos repositories.Repositories,
) (models.PendingBooking, error) {
	pendingBooking, err := repos.PendingBookings.GetByConfirmationToken(ctx, token)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return models.PendingBooking{}, viewErrors.ErrPendingBookingNotFound(
				fmt.Errorf("pending booking for token: %s not found", token),
			)
		}

		return models.PendingBooking{}, fmt.Errorf("could not get pending booking: %w", err)
	}

	_, err = repos.Bookings.GetByEmailAndClassID(ctx, pendingBooking.ClassID, pendingBooking.Email)
	if err == nil {
		return models.PendingBooking{}, viewErrors.ErrBookingAlreadyExists(
			pendingBooking.ClassID,
			pendingBooking.Email,
			fmt.Errorf("booking already exists for email %s and classID %s",
				pendingBooking.Email,
				pendingBooking.ClassID,
			),
		)
	}

	if !errors.Is(err, errs.ErrNotFound) {
		return models.PendingBooking{},
			fmt.Errorf("could not get booking for email %s and classID %s: %w",
				pendingBooking.Email,
				pendingBooking.ClassID,
				err,
			)
	}

	err = s.checkClassAvailability(ctx, repos, pendingBooking.Class)
	if err != nil {
		return models.PendingBooking{}, fmt.Errorf("class unavailable: %w", err)
	}

	return pendingBooking, nil
}

func (s *service) saveContact(
	ctx context.Context,
	repos repositories.Repositories,
	pendingBooking models.PendingBooking,
) error {
	_, err := repos.Contacts.Insert(
		ctx, pendingBooking.Email, pendingBooking.FirstName, pendingBooking.LastName,
	)
	if err != nil {
		if !errors.Is(err, errs.ErrAlreadyExist) {
			return fmt.Errorf("could not insert contact: %w", err)
		}
	}

	return nil
}

func (s *service) checkClassAvailability(
	ctx context.Context,
	repos repositories.Repositories,
	class models.Class,
) error {
	if class.StartTime.Before(time.Now()) {
		return viewErrors.ErrClassExpired(
			class.ID, fmt.Errorf("class %s has expired at %v", class.ID, class.StartTime),
		)
	}

	bookingCount, err := repos.Bookings.CountForClassID(ctx, class.ID)
	if err != nil {
		return fmt.Errorf("could not count bookings for class %v: %w ", class.ID, err)
	}

	if bookingCount == class.MaxCapacity {
		return viewErrors.ErrSomeoneBookedClassFaster(
			fmt.Errorf("max capacity of class %d exceeded", class.MaxCapacity),
		)
	}

	return nil
}

func (s *service) createBooking(
	ctx context.Context,
	pendingBooking models.PendingBooking,
	repos repositories.Repositories,
) (models.Booking, error) {
	// I need to check if previous passes don't have some empty slots. Three is enough.
	passes, err := repos.Passes.ListByEmail(ctx, pendingBooking.Email, threeLastPasses)
	if err != nil {
		return models.Booking{}, fmt.Errorf("could not get pass: %w", err)
	}

	booking := models.Booking{
		ID:                uuid.New(),
		ClassID:           pendingBooking.ClassID,
		FirstName:         pendingBooking.FirstName,
		LastName:          pendingBooking.LastName,
		Email:             pendingBooking.Email,
		CreatedAt:         time.Now().UTC(),
		ConfirmationToken: pendingBooking.ConfirmationToken,
		Class:             pendingBooking.Class,
	}

	for _, pass := range passes {
		bookingsWithPassCount, err := repos.Bookings.CountForPassID(ctx, pass.ID)
		if err != nil {
			return models.Booking{}, fmt.Errorf("could not count bookings for passID %d: %w", pass.ID, err)
		}

		if bookingsWithPassCount < pass.TotalSlots {
			booking.PassID = optional.Of(pass.ID)
			booking.Pass = optional.Of(pass)

			break
		}
	}

	_, err = repos.Bookings.Insert(ctx, booking)
	if err != nil {
		return models.Booking{}, fmt.Errorf("could not insert booking: %w", err)
	}

	return booking, nil
}

func (s *service) buildPassSlots(
	ctx context.Context,
	pass models.Pass,
	repos repositories.Repositories,
) ([]models.PassSlot, error) {
	bookings, err := repos.Bookings.ListByPassID(ctx, pass.ID)
	if err != nil {
		return nil, fmt.Errorf("could not list bookings by passID %d: %w", pass.ID, err)
	}

	return passes.BuildPassSlots(bookings, pass.TotalSlots, time.Now()), nil
}

func (s *service) sendConfirmation(
	booking models.Booking,
	notifierParams models.NotifierParams,
	token string,
) error {
	cancellationLink := fmt.Sprintf(
		"%s/bookings/%s/cancel_form?token=%s", s.domainAddr, booking.ID, token,
	)

	err := s.notifier.NotifyBookingConfirmation(notifierParams, cancellationLink)
	if err != nil {
		return fmt.Errorf("could not notify booking confirmation: %w", err)
	}

	return nil
}

func buildBookingCreation(
	booking models.Booking, locationLink string,
) BookingCreation {
	return BookingCreation{
		Class: classes.ClassPresentation{
			ID:           booking.Class.ID,
			StartTime:    booking.Class.StartTime,
			ClassLevel:   booking.Class.ClassLevel,
			ClassName:    booking.Class.ClassName,
			MaxCapacity:  booking.Class.MaxCapacity,
			Location:     booking.Class.Location,
			LocationLink: locationLink,
		},
	}
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
					fmt.Errorf("delete booking failure, booking with email %s for class %s not found",
						booking.Email,
						booking.ClassID,
					),
				)
			}

			return fmt.Errorf("could not delete booking: %w", err)
		}

		if booking.Pass.Exists() {
			passSlots, err = s.buildPassSlots(ctx, booking.Pass.Get(), repos)
			if err != nil {
				return fmt.Errorf("could not build passSlots: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("cancel booking transaction failed: %w", err)
	}

	locationLink, err := s.locationLinkProvider.GetLink(booking.Class.Location)
	if err != nil {
		return fmt.Errorf("could not resolve location link for: %s", booking.Class.Location)
	}

	notifierParams, err := s.buildNotifierParams(booking, locationLink, passSlots)
	if err != nil {
		return fmt.Errorf("could not build notifierParams: %w", err)
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

func (s *service) buildNotifierParams(
	booking models.Booking, locationLink string, passSlots []models.PassSlot,
) (models.NotifierParams, error) {
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

	return notifierParams, nil
}

func (s *service) GetBookingCancellationForm(
	ctx context.Context, bookingID uuid.UUID, token string,
) (BookingCancellationForm, error) {
	booking, err := s.bookingsRepo.GetByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return BookingCancellationForm{}, viewErrors.ErrBookingNotFound(
				fmt.Errorf("booking with id %s not found: %w", bookingID, err),
			)
		}

		return BookingCancellationForm{},
			fmt.Errorf("could not get booking for id %s: %w", bookingID, err)
	}

	if booking.ConfirmationToken != token {
		return BookingCancellationForm{}, viewErrors.ErrInvalidCancellationLink(err)
	}

	class := classes.BookingCancellationClass{
		ID:         booking.Class.ID,
		StartTime:  booking.Class.StartTime,
		ClassLevel: booking.Class.ClassLevel,
		ClassName:  booking.Class.ClassName,
		Location:   booking.Class.Location,
	}

	cancellationForm := BookingCancellationForm{
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
			passSlots, err = s.buildPassSlots(ctx, booking.Pass.Get(), repos)
			if err != nil {
				return fmt.Errorf("could not build passSlots: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("delete booking transaction failed: %w", err)
	}

	locationLink, err := s.locationLinkProvider.GetLink(booking.Class.Location)
	if err != nil {
		return fmt.Errorf("could not resolve location link for: %s", booking.Class.Location)
	}

	notifierParams, err := s.buildNotifierParams(booking, locationLink, passSlots)
	if err != nil {
		return fmt.Errorf("could not build notifierParams: %w", err)
	}

	err = s.notifier.NotifyBookingCancellation(notifierParams)
	if err != nil {
		return fmt.Errorf("could not nofify booking cancellation with %+v: %w", notifierParams, err)
	}

	return nil
}
