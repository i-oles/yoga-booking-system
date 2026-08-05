package dto

import (
	"fmt"
	"time"

	"main/internal/application/passes"
	"main/internal/domain/models"
	"main/pkg/converter"
)

type ActivatePassRequest struct {
	Email                string `binding:"required,min=3,max=40" json:"email"`
	InitialAssignedSlots int    `binding:"min=0" json:"initial_assigned_slots"`
	TotalSlots           int    `binding:"min=1" json:"total_slots"`
}

type ActivatePassResponse struct {
	Pass            PassDTO           `json:"pass"`
	UpdatedBookings []BookingResponse `json:"updated_bookings"`
}

type PassDTO struct {
	ID         int       `json:"id"`
	Email      string    `json:"email"`
	TotalSlots int       `json:"total_slots"`
	UpdatedAt  time.Time `json:"updated_at"`
	CreatedAt  time.Time `json:"created_at"`
}

func ToPassDTO(pass models.Pass) (PassDTO, error) {
	cratedAtWarsawTime, err := converter.ConvertToWarsawTime(pass.CreatedAt)
	if err != nil {
		return PassDTO{}, fmt.Errorf("error while converting createdAt to warsaw time: %w", err)
	}

	updatedAtWarsawTime, err := converter.ConvertToWarsawTime(pass.UpdatedAt)
	if err != nil {
		return PassDTO{}, fmt.Errorf("error while converting createdAt to warsaw time: %w", err)
	}

	return PassDTO{
		ID:         pass.ID,
		Email:      pass.Email,
		TotalSlots: pass.TotalSlots,
		UpdatedAt:  updatedAtWarsawTime,
		CreatedAt:  cratedAtWarsawTime,
	}, nil
}

func ToPassActivationResp(
	passActivation passes.PassActivation,
) (ActivatePassResponse, error) {
	passDTO, err := ToPassDTO(passActivation.Pass)
	if err != nil {
		return ActivatePassResponse{}, fmt.Errorf("could not convert pass to dto: %w", err)
	}

	updatedBookings, err := ToBookingsListResponse(passActivation.UpdatedBookings)
	if err != nil {
		return ActivatePassResponse{}, fmt.Errorf("could not convert updated bookings to dto: %w", err)
	}

	return ActivatePassResponse{
		Pass:            passDTO,
		UpdatedBookings: updatedBookings,
	}, nil
}
