package sqlite

import (
	"context"
	"fmt"

	"main/internal/domain/repositories"

	"gorm.io/gorm"
)

type unitOfWork struct {
	db *gorm.DB
}

func NewUnitOfWork(db *gorm.DB) *unitOfWork {
	return &unitOfWork{db: db}
}

func (u *unitOfWork) WithTransaction(
	ctx context.Context,
	fn func(repos repositories.Repositories) error, //nolint
) error {
	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repos := repositories.Repositories{
			PendingBookings: NewPendingBookingsRepo(tx),
			Bookings:        NewBookingsRepo(tx),
			Classes:         NewClassesRepo(tx),
			Passes:          NewPassesRepo(tx),
			Contacts:        NewContactsRepo(tx),
		}

		return fn(repos)
	})
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}
