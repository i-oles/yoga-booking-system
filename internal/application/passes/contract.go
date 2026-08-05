package passes

import (
	"context"
)

type IService interface {
	ActivatePass(
		ctx context.Context,
		email string,
		initialAssignedSlots, totalPassSlots int,
	) (PassActivation, error)
}
