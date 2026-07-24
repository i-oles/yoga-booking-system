package classes

import (
	"context"

	"main/internal/domain/models"

	"github.com/google/uuid"
)

type IService interface {
	ListClasses(
		ctx context.Context,
		onlyUpcomingClasses bool,
		classesLimit *int,
	) ([]ClassPresentation, error)
	CreateClasses(ctx context.Context, classes []models.Class) ([]models.Class, error)
	UpdateClass(ctx context.Context, id uuid.UUID, update UpdateClassCommand) (ClassData, error)
	DeleteClass(ctx context.Context, classID uuid.UUID, msg *string) error
}
