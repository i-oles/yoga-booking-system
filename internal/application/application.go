package application

import "main/internal/domain/models"

type IPassManager interface {
	BuildPassSlots(bookings []models.Booking, totalSlots int) []models.PassSlot
}

type ILocationResolver interface {
	GetLink(location string) (string, error)
}
