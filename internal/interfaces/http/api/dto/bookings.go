package dto

import (
	"fmt"
	"time"

	domainModels "main/internal/domain/models"
	"main/pkg/converter"

	"github.com/google/uuid"
)

type BookingResponse struct {
	ID        uuid.UUID     `json:"id"`
	ClassID   uuid.UUID     `json:"class_id"`
	PassID    *int          `json:"pass_id,omitempty"`
	Pass      *PassDTO      `json:"pass,omitempty"`
	FirstName string        `json:"first_name"`
	LastName  string        `json:"last_name"`
	Email     string        `json:"email"`
	CreatedAt time.Time     `json:"created_at"`
	Class     ClassResponse `json:"class"`
}

func ToBookingResponse(booking domainModels.Booking) (BookingResponse, error) {
	createdAtWarsawTime, err := converter.ConvertToWarsawTime(booking.CreatedAt)
	if err != nil {
		return BookingResponse{}, fmt.Errorf("could not convert createdAt to warsaw time: %w", err)
	}

	class, err := FromClass(booking.Class)
	if err != nil {
		return BookingResponse{}, fmt.Errorf("could not cast class to dto class: %w", err)
	}

	resp := BookingResponse{
		ID:        booking.ID,
		ClassID:   booking.ClassID,
		FirstName: booking.FirstName,
		LastName:  booking.LastName,
		Email:     booking.Email,
		CreatedAt: createdAtWarsawTime,
		Class:     class,
	}

	if booking.Pass.Exists() {
		pass := booking.Pass.Get()
		resp.PassID = &pass.ID

		passDTO, err := ToPassDTO(pass)
		if err != nil {
			return BookingResponse{}, fmt.Errorf("could not convert pass to passDTO: %w", err)
		}

		resp.Pass = &passDTO
	}

	return resp, nil
}

func ToBookingsListResponse(bookings []domainModels.Booking) ([]BookingResponse, error) {
	bookingsListResponse := make([]BookingResponse, len(bookings))

	for idx, booking := range bookings {
		bookingResponse, err := ToBookingResponse(booking)
		if err != nil {
			return nil, fmt.Errorf("could not convert booking to bookingResponse: %w", err)
		}

		bookingsListResponse[idx] = bookingResponse
	}

	return bookingsListResponse, nil
}
