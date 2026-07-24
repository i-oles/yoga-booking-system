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
	passesRepo   repositories.IPasses
	bookingsRepo repositories.IBookings
	notifier     notifier.INotifier
}

func NewService(
	passesRepo repositories.IPasses,
	bookingsRepo repositories.IBookings,
	notifier notifier.INotifier,
) *service {
	return &service{
		passesRepo:   passesRepo,
		bookingsRepo: bookingsRepo,
		notifier:     notifier,
	}
}

func (s *service) ActivatePass(
	ctx context.Context, params models.PassActivationParams,
) (models.PassActivation, error) {
	if params.InitialAssignedSlots > params.TotalSlots {
		return models.PassActivation{},
			api.ErrValidation(
				fmt.Errorf("initialAssignedSlots: %d is grater than totalSlots: %d",
					params.InitialAssignedSlots,
					params.TotalSlots),
			)
	}

	pass, err := s.passesRepo.Insert(
		ctx,
		params.Email,
		params.TotalSlots,
	)
	if err != nil {
		return models.PassActivation{}, fmt.Errorf("could not insert pass for %s: %w", params.Email, err)
	}

	bookingsToAssignToPass := make([]models.Booking, 0, params.InitialAssignedSlots)
	bookingIDsAssignedToPass := make([]uuid.UUID, 0, params.InitialAssignedSlots)

	// user may want to add one or more existing future bookings - system needs to assign those to Pass
	if params.InitialAssignedSlots > 0 {
		bookingsToAssignToPass, err = s.bookingsRepo.ListWithoutPassByEmail(
			ctx, params.Email, params.InitialAssignedSlots,
		)
		if err != nil {
			return models.PassActivation{},
				fmt.Errorf("could not list bookings for email %s: %w", params.Email, err)
		}

		if params.InitialAssignedSlots != len(bookingsToAssignToPass) {
			return models.PassActivation{}, api.ErrValidation(
				fmt.Errorf("initialUsedSlots should be equal to bookingsToAssignToPass: %d != %d",
					params.InitialAssignedSlots,
					len(bookingsToAssignToPass),
				),
			)
		}

		for _, booking := range bookingsToAssignToPass {
			err = s.bookingsRepo.Update(ctx, booking.ID, map[string]any{
				"pass_id": pass.ID,
			})
			if err != nil {
				return models.PassActivation{},
					fmt.Errorf("could not update booking %s with pass_id %d: %w", booking.ID, pass.ID, err)
			}

			bookingIDsAssignedToPass = append(bookingIDsAssignedToPass, booking.ID)
		}
	}

	passSlots := passes.BuildPassSlots(bookingsToAssignToPass, params.TotalSlots, time.Now())

	err = s.notifier.NotifyPassActivation(params.Email, passSlots)
	if err != nil {
		return models.PassActivation{}, fmt.Errorf("could notify pass activation with %v: %w", pass, err)
	}

	return models.PassActivation{
		Pass:               pass,
		BookingIDsAssigned: bookingIDsAssignedToPass,
	}, nil
}
