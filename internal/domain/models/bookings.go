package models

import (
	"time"

	"main/pkg/optional"

	"github.com/google/uuid"
)

type Booking struct {
	ID                uuid.UUID
	ClassID           uuid.UUID
	Class             Class
	PassID            optional.Optional[int]
	Pass              optional.Optional[Pass]
	FirstName         string
	LastName          string
	Email             string
	CreatedAt         time.Time
	ConfirmationToken string
	RemindedAt        *time.Time
}
