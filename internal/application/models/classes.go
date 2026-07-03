package models

import (
	"time"

	"github.com/google/uuid"
)

type UpdateClassCommand struct {
	ID          uuid.UUID
	StartTime   *time.Time
	ClassLevel  *string
	ClassName   *string
	MaxCapacity *int
	Location    *string
}

type BookingCancellationClass struct {
	ID         uuid.UUID
	StartTime  time.Time
	ClassLevel string
	ClassName  string
	Location   string
}

type ClassData struct {
	ID              uuid.UUID
	StartTime       time.Time
	ClassLevel      string
	ClassName       string
	CurrentCapacity int
	MaxCapacity     int
	Location        string
}

type ClassPresentation struct {
	ID              uuid.UUID
	StartTime       time.Time
	ClassLevel      string
	ClassName       string
	CurrentCapacity int
	MaxCapacity     int
	Location        string
	LocationLink    string
}
