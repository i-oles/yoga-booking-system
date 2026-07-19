package db

import (
	"time"

	"main/internal/domain/models"
)

type SQLPass struct {
	ID         int       `gorm:"primaryKey"`
	Email      string    `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	TotalSlots int       `gorm:"not null"`
}

func (SQLPass) TableName() string {
	return "passes"
}

func (s SQLPass) ToDomain() models.Pass {
	pass := models.Pass{
		ID:         s.ID,
		Email:      s.Email,
		UpdatedAt:  s.UpdatedAt,
		CreatedAt:  s.CreatedAt,
		TotalSlots: s.TotalSlots,
	}

	return pass
}

func SQLPassFromDomain(domain models.Pass) SQLPass {
	return SQLPass{
		ID:         domain.ID,
		Email:      domain.Email,
		UpdatedAt:  domain.UpdatedAt,
		CreatedAt:  domain.CreatedAt,
		TotalSlots: domain.TotalSlots,
	}
}
