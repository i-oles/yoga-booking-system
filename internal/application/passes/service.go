package passes

import (
	"context"
	"fmt"

	"main/internal/domain/errs/api"
	"main/internal/domain/models"
	"main/internal/domain/notifier"
	"main/internal/domain/repositories"
	"main/internal/domain/services"

	"github.com/google/uuid"
)

type service struct {
	passesRepo   repositories.IPasses
	bookingsRepo repositories.IBookings
	notifier     notifier.INotifier
	passManager  services.IPassManager
}

func NewService(
	passesRepo repositories.IPasses,
	bookingsRepo repositories.IBookings,
	notifier notifier.INotifier,
	passManager services.IPassManager,
) *service {
	return &service{
		passesRepo:   passesRepo,
		bookingsRepo: bookingsRepo,
		notifier:     notifier,
		passManager:  passManager,
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
				fmt.Errorf("number of initialUsedSlots should be exactly equal to number of bookingsToAssignToPass: %d != %d",
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

	passSlots := s.passManager.BuildPassSlots(bookingsToAssignToPass, params.TotalSlots)

	err = s.notifier.NotifyPassActivation(params.Email, passSlots)
	if err != nil {
		return models.PassActivation{}, fmt.Errorf("could notify pass activation with %v: %w", pass, err)
	}

	return models.PassActivation{
		Pass:               pass,
		BookingIDsAssigned: bookingIDsAssignedToPass,
	}, nil
}
