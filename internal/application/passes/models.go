package passes

import (
	"time"

	"main/internal/domain/models"
)

type PassActivation struct {
	Pass            models.Pass
	UpdatedBookings []models.Booking
}

type PassPresentation struct {
	ID         int
	Email      string
	TotalSlots int
	UsedSlots  int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
