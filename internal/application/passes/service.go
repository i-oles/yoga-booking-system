package passes

import (
	"context"
	"fmt"
	"time"

	"main/internal/domain/errs/api"
	"main/internal/domain/models"
	"main/internal/domain/notifier"
	"main/internal/domain/repositories"
	"main/internal/domain/services/passes"
)

type service struct {
	unitOfWork repositories.IUnitOfWork
	notifier   notifier.INotifier
}

func NewService(
	unitOfWork repositories.IUnitOfWork,
	notifier notifier.INotifier,
) *service {
	return &service{
		unitOfWork: unitOfWork,
		notifier:   notifier,
	}
}

func (s *service) ActivatePass(
	ctx context.Context,
	email string,
	initialAssignedSlots, totalPassSlots int,
) (PassActivation, error) {
	if initialAssignedSlots > totalPassSlots {
		return PassActivation{},
			api.ErrValidation(
				fmt.Errorf("initialAssignedSlots: %d is grater than totalSlots: %d",
					initialAssignedSlots,
					totalPassSlots),
			)
	}

	var (
		pass            models.Pass
		updatedBookings = make([]models.Booking, 0, initialAssignedSlots)
	)

	err := s.unitOfWork.WithTransaction(ctx, func(repos repositories.Repositories) error {
		var err error

		pass, err = repos.Passes.Insert(ctx, email, totalPassSlots)
		if err != nil {
			return fmt.Errorf("could not insert pass for %s: %w", email, err)
		}

		// user may add one or more existing bookings to pass - system needs to update those with pass
		if initialAssignedSlots > 0 {
			updatedBookings, err = s.updateBookingsWithPass(ctx, repos, pass, email, initialAssignedSlots)
			if err != nil {
				return fmt.Errorf("could not update bookings with pass: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return PassActivation{}, fmt.Errorf("pass activation transaction failed: %w", err)
	}

	passSlots := passes.BuildPassSlots(updatedBookings, totalPassSlots, time.Now())

	err = s.notifier.NotifyPassActivation(email, passSlots)
	if err != nil {
		return PassActivation{},
			fmt.Errorf("could notify pass activation for %s with %v: %w", email, passSlots, err)
	}

	return PassActivation{
		Pass:            pass,
		UpdatedBookings: updatedBookings,
	}, nil
}

func (s *service) updateBookingsWithPass(
	ctx context.Context,
	repos repositories.Repositories,
	pass models.Pass,
	email string,
	initialAssignedSlots int,
) ([]models.Booking, error) {
	bookings, err := repos.Bookings.ListWithoutPassByEmail(
		ctx, email, initialAssignedSlots,
	)
	if err != nil {
		return nil, fmt.Errorf("could not ListWithoutPass: %w", err)
	}

	if initialAssignedSlots != len(bookings) {
		return nil, api.ErrValidation(
			fmt.Errorf("initialUsedSlots should be equal to len bookingsToAssign: %d != %d",
				initialAssignedSlots,
				len(bookings),
			),
		)
	}

	change := map[string]any{"pass_id": pass.ID}

	updatedBookings := make([]models.Booking, 0, len(bookings))

	for _, booking := range bookings {
		updatedBooking, err := repos.Bookings.Update(ctx, booking.ID, change)
		if err != nil {
			return nil, fmt.Errorf("could not update booking %s with pass %d: %w", booking.ID, pass.ID, err)
		}

		updatedBookings = append(updatedBookings, updatedBooking)
	}

	return updatedBookings, nil
}
