package passes

import (
	"main/internal/domain/models"
)

type PassActivation struct {
	Pass            models.Pass
	UpdatedBookings []models.Booking
}
