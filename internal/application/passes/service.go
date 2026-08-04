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

	"github.com/google/uuid"
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
		pass                     models.Pass
		bookingsToAssignToPass   = make([]models.Booking, 0, initialAssignedSlots)
		bookingIDsAssignedToPass = make([]uuid.UUID, 0, initialAssignedSlots)
	)

	err := s.unitOfWork.WithTransaction(ctx, func(repos repositories.Repositories) error {
		var err error

		pass, err = repos.Passes.Insert(ctx, email, totalPassSlots)
		if err != nil {
			return fmt.Errorf("could not insert pass for %s: %w", email, err)
		}

		// user may want to add one or more existing bookings - system needs to assign those to Pass
		if initialAssignedSlots > 0 {
			bookingsToAssignToPass, err = repos.Bookings.ListWithoutPassByEmail(
				ctx, email, initialAssignedSlots,
			)
			if err != nil {
				return fmt.Errorf("could not ListWithoutPass: %w", err)
			}

			if initialAssignedSlots != len(bookingsToAssignToPass) {
				return api.ErrValidation(
					fmt.Errorf("initialUsedSlots should be equal to bookingsToAssignToPass: %d != %d",
						initialAssignedSlots,
						len(bookingsToAssignToPass),
					),
				)
			}

			for _, booking := range bookingsToAssignToPass {
				err = repos.Bookings.Update(ctx, booking.ID, map[string]any{
					"pass_id": pass.ID,
				})
				if err != nil {
					return fmt.Errorf("could not update booking %s with pass_id %d: %w", booking.ID, pass.ID, err)
				}

				bookingIDsAssignedToPass = append(bookingIDsAssignedToPass, booking.ID)
			}
		}

		return nil
	})
	if err != nil {
		return PassActivation{}, fmt.Errorf("pass activation transaction failed: %w", err)
	}

	passSlots := passes.BuildPassSlots(bookingsToAssignToPass, totalPassSlots, time.Now())

	err = s.notifier.NotifyPassActivation(email, passSlots)
	if err != nil {
		return PassActivation{}, fmt.Errorf("could notify pass activation with %v: %w", pass, err)
	}

	return PassActivation{
		Pass:               pass,
		BookingIDsAssigned: bookingIDsAssignedToPass,
	}, nil
}
