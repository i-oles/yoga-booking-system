package bookings

import (
	"context"

	"github.com/google/uuid"
)

type IService interface {
	CreateBooking(ctx context.Context, token string) (BookingCreation, error)
	CancelBooking(ctx context.Context, id uuid.UUID, token string) error
	GetBookingCancellationForm(
		ctx context.Context,
		id uuid.UUID,
		token string,
	) (BookingCancellationForm, error)
	DeleteBooking(ctx context.Context, id uuid.UUID) error
}
