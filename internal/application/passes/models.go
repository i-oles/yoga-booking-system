package passes

import (
	"main/internal/domain/models"

	"github.com/google/uuid"
)

type PassActivation struct {
	Pass               models.Pass
	BookingIDsAssigned []uuid.UUID
}
