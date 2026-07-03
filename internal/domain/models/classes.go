package models

import (
	"time"

	"github.com/google/uuid"
)

type Class struct {
	ID          uuid.UUID
	StartTime   time.Time
	ClassLevel  string
	ClassName   string
	MaxCapacity int
	Location    string
}
