package sqlite

import (
	"context"
	"fmt"

	"main/internal/domain/models"
	"main/internal/infrastructure/models/db"

	"gorm.io/gorm"
)

type passesRepo struct {
	db *gorm.DB
}

func NewPassesRepo(db *gorm.DB) *passesRepo {
	return &passesRepo{
		db: db,
	}
}

func (r *passesRepo) ListByEmail(
	ctx context.Context, email string, limit int,
) ([]models.Pass, error) {
	var SQLPasses []db.SQLPass

	if err := r.db.WithContext(ctx).
		Where("email = ?", email).
		Order("created_at DESC").
		Limit(limit).
		Find(&SQLPasses).Error; err != nil {
		return nil, fmt.Errorf("could not get passes for email %s: %w", email, err)
	}

	result := make([]models.Pass, len(SQLPasses))

	for i, SQLPass := range SQLPasses {
		result[i] = SQLPass.ToDomain()
	}

	return result, nil
}

func (r *passesRepo) Insert(
	ctx context.Context,
	email string,
	totalSlots int,
) (models.Pass, error) {
	pass := db.SQLPass{
		Email:      email,
		TotalSlots: totalSlots,
	}

	if err := r.db.WithContext(ctx).Create(&pass).Error; err != nil {
		return models.Pass{}, fmt.Errorf("could not insert pass: %w", err)
	}

	return pass.ToDomain(), nil
}
