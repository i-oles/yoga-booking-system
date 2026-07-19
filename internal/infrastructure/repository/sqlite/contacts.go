package sqlite

import (
	"context"
	"errors"
	"fmt"

	"main/internal/domain/models"
	"main/internal/infrastructure/errs"
	"main/internal/infrastructure/models/db"

	"github.com/mattn/go-sqlite3"
	"gorm.io/gorm"
)

type contactsRepo struct {
	db *gorm.DB
}

func NewContactsRepo(db *gorm.DB) *contactsRepo {
	return &contactsRepo{
		db: db,
	}
}

func (r *contactsRepo) Insert(
	ctx context.Context,
	email, firstName, lastName string,
) (models.Contact, error) {
	contact := db.SQLContact{
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
	}

	var sqliteErr sqlite3.Error

	if err := r.db.WithContext(ctx).
		Create(&contact).Error; err != nil {
		if errors.As(err, &sqliteErr) &&
			sqliteErr.Code == sqlite3.ErrConstraint &&
			sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return models.Contact{}, errs.ErrAlreadyExist
		}

		return models.Contact{}, fmt.Errorf("could not insert contact: %w", err)
	}

	return contact.ToDomain(), nil
}

func (r *contactsRepo) List(ctx context.Context) ([]models.Contact, error) {
	var SQLContacts []db.SQLContact

	if err := r.db.WithContext(ctx).
		Order("last_name ASC").
		Find(&SQLContacts).Error; err != nil {
		return nil, fmt.Errorf("could not list contacts: %w", err)
	}

	result := make([]models.Contact, len(SQLContacts))

	for i, SQLContact := range SQLContacts {
		result[i] = SQLContact.ToDomain()
	}

	return result, nil
}
