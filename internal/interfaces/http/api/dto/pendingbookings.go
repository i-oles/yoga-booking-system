package dto

import (
	"fmt"
	"time"

	"main/internal/domain/models"
	"main/pkg/converter"

	"github.com/google/uuid"
)

type PendingBookingResponse struct {
	ID        uuid.UUID    `json:"id"`
	ClassID   uuid.UUID    `json:"class_id"`
	Class     models.Class `json:"class"`
	FirstName string       `json:"first_name"`
	LastName  string       `json:"last_name"`
	Email     string       `json:"email"`
	CreatedAt time.Time    `json:"created_at"`
}

func ToPendingBookingResponse(
	pendingBooking models.PendingBooking,
) (PendingBookingResponse, error) {
	createdAtWarsawTime, err := converter.ConvertToWarsawTime(pendingBooking.CreatedAt)
	if err != nil {
		return PendingBookingResponse{}, fmt.Errorf("could not convert createdAt to warsaw time: %w", err)
	}

	return PendingBookingResponse{
		ID:        pendingBooking.ID,
		ClassID:   pendingBooking.ClassID,
		Class:     pendingBooking.Class,
		FirstName: pendingBooking.FirstName,
		LastName:  pendingBooking.LastName,
		Email:     pendingBooking.Email,
		CreatedAt: createdAtWarsawTime,
	}, nil
}

func ToPendingBookingsListResponse(
	pendingBookings []models.PendingBooking,
) ([]PendingBookingResponse, error) {
	pendingBookingsListResponse := make([]PendingBookingResponse, len(pendingBookings))

	for idx, pendingBooking := range pendingBookings {
		resp, err := ToPendingBookingResponse(pendingBooking)
		if err != nil {
			return nil, fmt.Errorf("could not convert booking to bookingResponse: %w", err)
		}

		pendingBookingsListResponse[idx] = resp
	}

	return pendingBookingsListResponse, nil
}
