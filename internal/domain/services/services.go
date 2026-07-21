package services

import (
	"context"

	appModels "main/internal/application/models"
	"main/internal/domain/models"

	"github.com/google/uuid"
)

type IClassesService interface {
	ListClasses(
		ctx context.Context,
		onlyUpcomingClasses bool,
		classesLimit *int,
	) ([]appModels.ClassPresentation, error)
	CreateClasses(ctx context.Context, classes []models.Class) ([]models.Class, error)
	UpdateClass(ctx context.Context, id uuid.UUID, update appModels.UpdateClassCommand) (appModels.ClassData, error)
	DeleteClass(ctx context.Context, classID uuid.UUID, msg *string) error
}

type IBookingsService interface {
	CreateBooking(ctx context.Context, token string) (appModels.BookingCreation, error)
	CancelBooking(ctx context.Context, id uuid.UUID, token string) error
	GetBookingCancellationForm(
		ctx context.Context,
		id uuid.UUID,
		token string,
	) (appModels.BookingCancellationForm, error)
	DeleteBooking(ctx context.Context, id uuid.UUID) error
}

type IPendingBookingsService interface {
	CreatePendingBooking(ctx context.Context, params models.PendingBookingParams) error
}

type IPassesService interface {
	ActivatePass(
		ctx context.Context,
		params models.PassActivationParams,
	) (models.PassActivation, error)
}

type ITokenGenerator interface {
	Generate(length int) (string, error)
}
