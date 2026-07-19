package sqlite

import (
	"context"
	"errors"
	"fmt"

	"main/internal/domain/models"
	"main/internal/infrastructure/errs"
	"main/internal/infrastructure/models/db"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type bookingsRepo struct {
	db *gorm.DB
}

func NewBookingsRepo(db *gorm.DB) *bookingsRepo {
	return &bookingsRepo{
		db: db,
	}
}

func (r *bookingsRepo) GetByID(
	ctx context.Context, bookingID uuid.UUID,
) (models.Booking, error) {
	var SQLBooking db.SQLBooking

	result := r.db.WithContext(ctx).Where("id = ?", bookingID).Preload("Class").Preload("Pass").First(&SQLBooking)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return models.Booking{}, errs.ErrNotFound
		}

		return models.Booking{},
			fmt.Errorf("could not get booking for id %s: %w", bookingID, result.Error)
	}

	return SQLBooking.ToDomain(), nil
}

func (r *bookingsRepo) GetByEmailAndClassID(
	ctx context.Context,
	classID uuid.UUID,
	email string,
) (models.Booking, error) {
	var SQLBooking db.SQLBooking

	result := r.db.WithContext(ctx).
		Where("class_id = ? AND email = ?", classID, email).
		First(&SQLBooking)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return models.Booking{}, errs.ErrNotFound
		}

		return models.Booking{},
			fmt.Errorf("could not get booking by email %s, classID %s: %w", email, classID, result.Error)
	}

	return SQLBooking.ToDomain(), nil
}

func (r *bookingsRepo) ListWithoutPassByEmail(
	ctx context.Context, email string, limit int) ([]models.Booking, error,
) {
	var SQLBookings []db.SQLBooking

	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive: %d", limit)
	}

	if err := r.db.WithContext(ctx).
		Where("email = ? AND pass_id IS NULL", email).
		Order("created_at DESC").
		Limit(limit).
		Preload("Class").
		Preload("Pass").
		Find(&SQLBookings).Error; err != nil {
		return nil, fmt.Errorf("could not get bookings for %s without pass_id: %w", email, err)
	}

	result := make([]models.Booking, len(SQLBookings))

	for i, SQLBooking := range SQLBookings {
		result[i] = SQLBooking.ToDomain()
	}

	return result, nil
}

func (r *bookingsRepo) List(ctx context.Context) ([]models.Booking, error) {
	var SQLBookings []db.SQLBooking

	if err := r.db.WithContext(ctx).Preload("Class").Preload("Pass").Find(&SQLBookings).Error; err != nil {
		return nil, fmt.Errorf("could not list bookings: %w", err)
	}

	result := make([]models.Booking, len(SQLBookings))

	for i, SQLBooking := range SQLBookings {
		result[i] = SQLBooking.ToDomain()
	}

	return result, nil
}

func (r *bookingsRepo) CountForClassID(ctx context.Context, classID uuid.UUID) (int, error) {
	var count int64

	var SQLBooking db.SQLBooking

	if err := r.db.WithContext(ctx).
		Model(&SQLBooking).
		Where("class_id = ?", classID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("could count bookings for classID %s: %w", classID, err)
	}

	return int(count), nil
}

func (r *bookingsRepo) CountForPassID(ctx context.Context, passID int) (int, error) {
	var count int64

	var SQLBooking db.SQLBooking

	if err := r.db.WithContext(ctx).
		Model(&SQLBooking).
		Where("pass_id = ?", passID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("could count bookings for passID %d: %w", passID, err)
	}

	return int(count), nil
}

func (r *bookingsRepo) ListByClassID(
	ctx context.Context,
	classID uuid.UUID,
) ([]models.Booking, error) {
	var SQLBookings []db.SQLBooking

	if err := r.db.WithContext(ctx).
		Where("class_id = ?", classID).
		Preload("Class").
		Preload("Pass").
		Find(&SQLBookings).Error; err != nil {
		return nil, fmt.Errorf("could not get bookings for classID %s: %w", classID, err)
	}

	result := make([]models.Booking, len(SQLBookings))

	for i, SQLBooking := range SQLBookings {
		result[i] = SQLBooking.ToDomain()
	}

	return result, nil
}

func (r *bookingsRepo) ListByPassID(
	ctx context.Context,
	passID int,
) ([]models.Booking, error) {
	var SQLBookings []db.SQLBooking

	if err := r.db.WithContext(ctx).
		Preload("Class").Preload("Pass").
		Where("pass_id = ?", passID).
		Order("created_at ASC").
		Find(&SQLBookings).Error; err != nil {
		return nil, fmt.Errorf("could not list bookings: %w", err)
	}

	result := make([]models.Booking, len(SQLBookings))

	for i, SQLBooking := range SQLBookings {
		result[i] = SQLBooking.ToDomain()
	}

	return result, nil
}

func (r *bookingsRepo) Insert(
	ctx context.Context,
	booking models.Booking,
) (uuid.UUID, error) {
	SQLBooking := db.SQLBookingFromDomain(booking)

	if err := r.db.WithContext(ctx).Create(&SQLBooking).Error; err != nil {
		return uuid.Nil, fmt.Errorf("could not insert booking: %w", err)
	}

	return booking.ID, nil
}

func (r *bookingsRepo) Delete(ctx context.Context, id uuid.UUID) error {
	var SQLBooking db.SQLBooking

	result := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&SQLBooking)
	if result.Error != nil {
		return fmt.Errorf("could not delete booking: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errs.ErrNoRowsAffected
	}

	return nil
}

func (r *bookingsRepo) Update(
	ctx context.Context,
	bookingID uuid.UUID,
	update map[string]any,
) error {
	var SQLBooking db.SQLBooking

	result := r.db.WithContext(ctx).
		Model(&SQLBooking).
		Clauses(clause.Returning{}).
		Where("id = ?", bookingID).
		Updates(update)

	if result.Error != nil {
		return fmt.Errorf("could not update booking: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no booking found for id: %s", bookingID)
	}

	return nil
}
