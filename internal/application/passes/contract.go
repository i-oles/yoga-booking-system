package passes

import (
	"context"

	"main/internal/domain/models"
)

type IService interface {
	ActivatePass(
		ctx context.Context,
		params models.PassActivationParams,
	) (models.PassActivation, error)
}
