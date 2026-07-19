package db

import (
	"time"

	"main/internal/domain/models"

	"github.com/google/uuid"
)

type SQLPendingBooking struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey"`
	ClassID           uuid.UUID `gorm:"type:uuid;not null"`
	Class             SQLClass  `gorm:"foreignKey:class_id"`
	Email             string    `gorm:"not null"`
	FirstName         string    `gorm:"not null"`
	LastName          string    `gorm:"not null"`
	ConfirmationToken string    `gorm:"unique;not null"`
	CreatedAt         time.Time `gorm:"autoCreateTime"`
}

func (SQLPendingBooking) TableName() string {
	return "pending_bookings"
}

func (s SQLPendingBooking) ToDomain() models.PendingBooking {
	return models.PendingBooking{
		ID:                s.ID,
		ClassID:           s.ClassID,
		Class:             s.Class.ToDomain(),
		Email:             s.Email,
		FirstName:         s.FirstName,
		LastName:          s.LastName,
		ConfirmationToken: s.ConfirmationToken,
		CreatedAt:         s.CreatedAt,
	}
}

func SQLPendingBookingFromDomain(domain models.PendingBooking) SQLPendingBooking {
	return SQLPendingBooking{
		ID:                domain.ID,
		ClassID:           domain.ClassID,
		Class:             SQLClassFromDomain(domain.Class),
		Email:             domain.Email,
		FirstName:         domain.FirstName,
		LastName:          domain.LastName,
		ConfirmationToken: domain.ConfirmationToken,
		CreatedAt:         domain.CreatedAt,
	}
}
