package sqlite

import (
	"context"
	"errors"
	"fmt"

	"main/internal/domain/models"
	"main/internal/infrastructure/errs"
	"main/internal/infrastructure/models/db"

	"gorm.io/gorm"
)

type pendingBookingsRepo struct {
	db *gorm.DB
}

func NewPendingBookingsRepo(db *gorm.DB) *pendingBookingsRepo {
	return &pendingBookingsRepo{
		db: db,
	}
}

func (r *pendingBookingsRepo) Insert(
	ctx context.Context,
	pendingBooking models.PendingBooking,
) error {
	sqlPendingBooking := db.SQLPendingBookingFromDomain(pendingBooking)

	if err := r.db.WithContext(ctx).Create(&sqlPendingBooking).Error; err != nil {
		return fmt.Errorf("could not insert pending booking: %w", err)
	}

	return nil
}

func (r *pendingBookingsRepo) GetByConfirmationToken(
	ctx context.Context,
	token string,
) (models.PendingBooking, error) {
	var sqlPendingBooking db.SQLPendingBooking

	if err := r.db.WithContext(ctx).
		Where("confirmation_token = ?", token).
		Preload("Class").
		First(&sqlPendingBooking).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.PendingBooking{}, errs.ErrNotFound
		}

		return models.PendingBooking{}, fmt.Errorf("could not get pending booking: %w", err)
	}

	return sqlPendingBooking.ToDomain(), nil
}

func (r *pendingBookingsRepo) List(ctx context.Context) ([]models.PendingBooking, error) {
	var SQLPendingBookings []db.SQLPendingBooking

	if err := r.db.WithContext(ctx).
		Preload("Class").
		Find(&SQLPendingBookings).Error; err != nil {
		return nil, fmt.Errorf("could not list all pending bookings: %w", err)
	}

	result := make([]models.PendingBooking, len(SQLPendingBookings))

	for i, SQLPendingBooking := range SQLPendingBookings {
		result[i] = SQLPendingBooking.ToDomain()
	}

	return result, nil
}
