package db

import (
	"time"

	"main/internal/domain/models"
	"main/pkg/optional"

	"github.com/google/uuid"
)

type SQLBooking struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey"`
	ClassID           uuid.UUID `gorm:"type:uuid;not null"`
	Class             SQLClass  `gorm:"foreignKey:class_id"`
	PassID            *int
	Pass              *SQLPass `gorm:"foreignKey:pass_id"`
	Email             string   `gorm:"not null"`
	FirstName         string   `gorm:"not null"`
	LastName          string   `gorm:"not null"`
	ConfirmationToken string   `gorm:"unique;not null"`
	RemindedAt        *time.Time
	CreatedAt         time.Time `gorm:"autoCreateTime"`
}

func (SQLBooking) TableName() string {
	return "bookings"
}

func (s SQLBooking) ToDomain() models.Booking {
	booking := models.Booking{
		ID:                s.ID,
		ClassID:           s.ClassID,
		Class:             s.Class.ToDomain(),
		FirstName:         s.FirstName,
		LastName:          s.LastName,
		Email:             s.Email,
		CreatedAt:         s.CreatedAt,
		RemindedAt:        s.RemindedAt,
		ConfirmationToken: s.ConfirmationToken,
	}

	if s.Pass != nil {
		pass := s.Pass.ToDomain()
		booking.PassID = optional.Of(pass.ID)
		booking.Pass = optional.Of(pass)
	}

	return booking
}

func SQLBookingFromDomain(domain models.Booking) SQLBooking {
	booking := SQLBooking{
		ID:                domain.ID,
		ClassID:           domain.ClassID,
		Class:             SQLClassFromDomain(domain.Class),
		FirstName:         domain.FirstName,
		LastName:          domain.LastName,
		Email:             domain.Email,
		CreatedAt:         domain.CreatedAt,
		RemindedAt:        domain.RemindedAt,
		ConfirmationToken: domain.ConfirmationToken,
	}

	if domain.Pass.Exists() {
		pass := SQLPassFromDomain(domain.Pass.Get())
		booking.PassID = &pass.ID
		booking.Pass = &pass
	}

	return booking
}
