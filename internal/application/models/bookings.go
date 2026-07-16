package models

import (
	"github.com/google/uuid"
)

type BookingCreation struct {
	Class ClassPresentation
}

type BookingCancellationForm struct {
	Class             BookingCancellationClass
	BookingID         uuid.UUID
	ConfirmationToken string
}
