package pendingbookings

import (
	"context"

	"main/internal/domain/models"
)

type IService interface {
	CreatePendingBooking(ctx context.Context, params models.PendingBookingParams) error
}

type ITokenGenerator interface {
	Generate(length int) (string, error)
}
