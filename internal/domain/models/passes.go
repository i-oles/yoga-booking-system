package models

import (
	"time"

	"github.com/google/uuid"
)

type Pass struct {
	ID         int
	Email      string
	TotalSlots int
	UpdatedAt  time.Time
	CreatedAt  time.Time
}

type PassSlot struct {
	Status         PassSlotStatus
	ClassStartTime *time.Time
}

type PassSlotStatus int

const (
	Blank PassSlotStatus = iota
	Past
	Future
)

type PassActivationParams struct {
	Email                string
	InitialAssignedSlots int
	TotalSlots           int
}

type PassActivation struct {
	Pass               Pass
	BookingIDsAssigned []uuid.UUID
}
