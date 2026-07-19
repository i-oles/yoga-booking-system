package repositories

import (
	"context"

	"main/internal/domain/models"

	"github.com/google/uuid"
)

type Repositories struct {
	PendingBookings IPendingBookings
	Bookings        IBookings
	Classes         IClasses
	Passes          IPasses
	Contacts        IContacts
}

type IClasses interface {
	Get(ctx context.Context, id uuid.UUID) (models.Class, error)
	List(ctx context.Context) ([]models.Class, error)
	Insert(ctx context.Context, classes []models.Class) ([]models.Class, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Update(ctx context.Context, id uuid.UUID, update map[string]any) (models.Class, error)
}

type IBookings interface {
	GetByID(ctx context.Context, id uuid.UUID) (models.Booking, error)
	GetByEmailAndClassID(ctx context.Context, classID uuid.UUID, email string) (models.Booking, error)
	List(ctx context.Context) ([]models.Booking, error)
	ListWithoutPassByEmail(ctx context.Context, email string, limit int) ([]models.Booking, error)
	ListByClassID(ctx context.Context, classID uuid.UUID) ([]models.Booking, error)
	ListByPassID(ctx context.Context, passID int) ([]models.Booking, error)
	CountForPassID(ctx context.Context, passID int) (int, error)
	CountForClassID(ctx context.Context, classID uuid.UUID) (int, error)
	Insert(ctx context.Context, booking models.Booking) (uuid.UUID, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Update(ctx context.Context, id uuid.UUID, update map[string]any) error
}

type IPendingBookings interface {
	GetByConfirmationToken(ctx context.Context, token string) (models.PendingBooking, error)
	Insert(ctx context.Context, booking models.PendingBooking) error
	List(ctx context.Context) ([]models.PendingBooking, error)
}

type IPasses interface {
	Insert(ctx context.Context, email string, totalSlots int) (models.Pass, error)
	ListByEmail(ctx context.Context, email string, limit int) ([]models.Pass, error)
}

type IContacts interface {
	Insert(ctx context.Context, email, firstName, lastName string) (models.Contact, error)
	List(ctx context.Context) ([]models.Contact, error)
}
