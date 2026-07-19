package models

import (
	"time"

	"github.com/google/uuid"
)

type PendingBooking struct {
	ID                uuid.UUID
	ClassID           uuid.UUID
	Class             Class
	Email             string
	FirstName         string
	LastName          string
	ConfirmationToken string
	CreatedAt         time.Time
}

type PendingBookingParams struct {
	ClassID   uuid.UUID
	FirstName string
	LastName  string
	Email     string
}
