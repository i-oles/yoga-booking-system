package sqlite

import (
	"context"
	"errors"
	"fmt"
	"time"

	"main/internal/domain/models"
	"main/internal/infrastructure/errs"
	"main/internal/infrastructure/models/db"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type classesRepo struct {
	db *gorm.DB
}

func NewClassesRepo(db *gorm.DB) *classesRepo {
	return &classesRepo{
		db: db,
	}
}

func (r *classesRepo) List(ctx context.Context) ([]models.Class, error) {
	var sqlClasses []db.SQLClass

	if err := r.db.WithContext(ctx).Order("start_time ASC").Find(&sqlClasses).Error; err != nil {
		return nil, fmt.Errorf("could not get all classes: %w", err)
	}

	classes := make([]models.Class, len(sqlClasses))

	for i, sqlClass := range sqlClasses {
		classes[i] = sqlClass.ToDomain()
	}

	return classes, nil
}

func (r *classesRepo) Get(ctx context.Context, id uuid.UUID) (models.Class, error) {
	var sqlClass db.SQLClass

	if err := r.db.WithContext(ctx).First(&sqlClass, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Class{}, errs.ErrNotFound
		}

		return models.Class{}, fmt.Errorf("could not get class: %w", err)
	}

	return sqlClass.ToDomain(), nil
}

func (r *classesRepo) Insert(ctx context.Context, classes []models.Class) ([]models.Class, error) {
	sqlClass := make([]db.SQLClass, len(classes))
	for i, class := range classes {
		sqlClass[i] = db.SQLClassFromDomain(class)
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&sqlClass).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not insert classes: %w", err)
	}

	insertedClasses := make([]models.Class, len(sqlClass))
	for i, SQLClass := range sqlClass {
		insertedClasses[i] = SQLClass.ToDomain()
	}

	return insertedClasses, nil
}

func (r *classesRepo) Delete(ctx context.Context, id uuid.UUID) error {
	var sqlClass db.SQLClass

	result := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&sqlClass)
	if result.Error != nil {
		return fmt.Errorf("could not delete class: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errs.ErrNoRowsAffected
	}

	return nil
}

func (r *classesRepo) Update(
	ctx context.Context,
	classID uuid.UUID,
	update map[string]any,
) (models.Class, error) {
	var sqlClass db.SQLClass

	if err := r.db.WithContext(ctx).
		Model(&sqlClass).
		Clauses(clause.Returning{}).
		Where("id = ?", classID).
		Updates(update).Error; err != nil {
		return models.Class{},
			fmt.Errorf("could not update class: %v with data: %v, %w", classID, update, err)
	}

	return sqlClass.ToDomain(), nil
}

func (r *classesRepo) CountUpcomingClasses(ctx context.Context) (int, error) {
	var count int64

	if err := r.db.WithContext(ctx).
		Model(&db.SQLClass{}).
		Where("start_time > ?", time.Now()).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("could not count upcoming classes: %w", err)
	}

	return int(count), nil
}
