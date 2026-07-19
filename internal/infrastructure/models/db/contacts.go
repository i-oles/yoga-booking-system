package db

import (
	"main/internal/domain/models"
)

type SQLContact struct {
	ID        int    `gorm:"primaryKey"`
	Email     string `gorm:"uniqueIndex;not null"`
	FirstName string `gorm:"not null"`
	LastName  string `gorm:"not null"`
}

func (SQLContact) TableName() string {
	return "contacts"
}

func (s SQLContact) ToDomain() models.Contact {
	return models.Contact{
		ID:        s.ID,
		Email:     s.Email,
		FirstName: s.FirstName,
		LastName:  s.LastName,
	}
}
