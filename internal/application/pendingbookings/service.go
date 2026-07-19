package pendingbookings

import (
	"context"
	"errors"
	"fmt"
	"time"

	viewErrors "main/internal/domain/errs/view"
	"main/internal/domain/models"
	"main/internal/domain/notifier"
	"main/internal/domain/repositories"
	"main/internal/domain/services"
	"main/internal/infrastructure/errs"

	"github.com/google/uuid"
)

const (
	allowedTotalPendingBookingsLimit = 200
	tokenLength                      = 32
	deadlineBeforeClassStart         = 3 * time.Hour
)

type service struct {
	unitOfWork     repositories.IUnitOfWork
	tokenGenerator services.ITokenGenerator
	notifier       notifier.INotifier
	domainAddr     string
}

func NewService(
	unitOfWork repositories.IUnitOfWork,
	tokenGenerator services.ITokenGenerator,
	notifier notifier.INotifier,
	domainAddr string,
) *service {
	return &service{
		unitOfWork:     unitOfWork,
		tokenGenerator: tokenGenerator,
		notifier:       notifier,
		domainAddr:     domainAddr,
	}
}

func (s *service) CreatePendingBooking(
	ctx context.Context,
	pendingBookingParams models.PendingBookingParams,
) error {
	var (
		confirmationToken string
		class             models.Class
	)

	err := s.unitOfWork.WithTransaction(ctx, func(repos repositories.Repositories) error {
		err := s.ensurePendingBookingCreationAllowed(
			ctx,
			repos,
			pendingBookingParams.ClassID,
			pendingBookingParams.Email,
		)
		if err != nil {
			return fmt.Errorf("pending booking creation not allowed: %w", err)
		}

		class, err = s.checkClassAvailability(ctx, repos, pendingBookingParams.ClassID)
		if err != nil {
			return fmt.Errorf("class not available: %w", err)
		}

		confirmationToken, err = s.tokenGenerator.Generate(tokenLength)
		if err != nil {
			return fmt.Errorf("could not generate confirmation token: %w", err)
		}

		pendingBooking := models.PendingBooking{
			ID:                uuid.New(),
			ClassID:           pendingBookingParams.ClassID,
			Email:             pendingBookingParams.Email,
			FirstName:         pendingBookingParams.FirstName,
			LastName:          pendingBookingParams.LastName,
			ConfirmationToken: confirmationToken,
			CreatedAt:         time.Now().UTC(),
		}

		err = repos.PendingBookings.Insert(ctx, pendingBooking)
		if err != nil {
			return fmt.Errorf("could not insert pending booking: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("create pending booking transaction failed: %w", err)
	}

	err = s.notifier.NotifyConfirmationLink(
		pendingBookingParams.Email,
		pendingBookingParams.FirstName,
		fmt.Sprintf("%s/bookings?token=%s", s.domainAddr, confirmationToken),
		class.StartTime,
	)
	if err != nil {
		return fmt.Errorf("could not notify confirmation link: %w", err)
	}

	return nil
}

func (s *service) ensurePendingBookingCreationAllowed(
	ctx context.Context,
	repos repositories.Repositories,
	classID uuid.UUID,
	email string,
) error {
	_, err := repos.Bookings.GetByEmailAndClassID(ctx, classID, email)
	if err == nil {
		return viewErrors.ErrBookingAlreadyExists(
			classID,
			email,
			fmt.Errorf("booking for class %v and email %s already exists", classID, email),
		)
	}

	if !errors.Is(err, errs.ErrNotFound) {
		return fmt.Errorf("could not get booking: %w", err)
	}

	pendingBookings, err := repos.PendingBookings.List(ctx)
	if err != nil {
		return fmt.Errorf("could not list pending bookings: %w", err)
	}

	if len(pendingBookings) >= allowedTotalPendingBookingsLimit {
		return fmt.Errorf("limit: %d of pending bookings exceeded", allowedTotalPendingBookingsLimit)
	}

	var count int

	for _, pendingBooking := range pendingBookings {
		if pendingBooking.Email == email && pendingBooking.ClassID == classID {
			count++
		}
	}

	if count >= 2 {
		return viewErrors.ErrTooManyPendingBookings(
			classID,
			email,
			fmt.Errorf("found %d pending operations per user", count),
		)
	}

	return nil
}

func (s *service) checkClassAvailability(
	ctx context.Context,
	repos repositories.Repositories,
	classID uuid.UUID,
) (models.Class, error) {
	bookingCount, err := repos.Bookings.CountForClassID(ctx, classID)
	if err != nil {
		return models.Class{}, fmt.Errorf("could not count bookings for class %v: %w ", classID, err)
	}

	class, err := repos.Classes.Get(ctx, classID)
	if err != nil {
		return models.Class{}, fmt.Errorf("could not get class: %w", err)
	}

	if bookingCount == class.MaxCapacity {
		return models.Class{}, viewErrors.ErrClassFullyBooked(classID, fmt.Errorf("no spots left in class with id: %d", classID))
	}

	if class.StartTime.Before(time.Now()) {
		return models.Class{}, viewErrors.ErrClassExpired(classID, fmt.Errorf("class %s has expired at %v", classID, class.StartTime))
	}

	if bookingCount == 0 && time.Until(class.StartTime) < deadlineBeforeClassStart {
		return models.Class{}, viewErrors.ErrTooLateToBook(classID, fmt.Errorf("class %s is empty and it is to late to book", class.ID))
	}

	return class, nil
}
