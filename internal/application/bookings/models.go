package bookings

import (
	"main/internal/application/classes"

	"github.com/google/uuid"
)

type BookingCreation struct {
	Class classes.ClassPresentation
}

type BookingCancellationForm struct {
	Class             classes.BookingCancellationClass
	BookingID         uuid.UUID
	ConfirmationToken string
}
