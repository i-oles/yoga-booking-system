package passes

import (
	"context"
	"testing"
	"time"

	"main/internal/domain/models"
	"main/internal/domain/repositories"
	"main/mock"
	"main/pkg/optional"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type testData struct {
	pass    models.Pass
	booking models.Booking
}

func newTestData() testData {
	return testData{
		pass:    newPass(),
		booking: newBooking(newClass()),
	}
}

func newClass() models.Class {
	return models.Class{
		ID:          uuid.New(),
		StartTime:   time.Now().Add(24 * time.Hour),
		ClassLevel:  "for everyone",
		ClassName:   "Morning Yoga",
		MaxCapacity: 10,
		Location:    "Warsaw",
	}
}

func newBooking(class models.Class) models.Booking {
	return models.Booking{
		ID:                uuid.New(),
		ClassID:           class.ID,
		Class:             class,
		ConfirmationToken: "token",
		FirstName:         "John",
		LastName:          "Doe",
		Email:             "john@example.com",
		CreatedAt:         time.Now(),
	}
}

func newPass() models.Pass {
	return models.Pass{
		ID:         1,
		Email:      "john@example.com",
		TotalSlots: 8,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func mockActivatePassTransaction(
	uow *mock.MockIUnitOfWork,
	passesRepo *mock.MockIPasses,
	bookingsRepo *mock.MockIBookings,
) {
	uow.EXPECT().
		WithTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			ctx context.Context,
			fn func(repositories.Repositories) error,
		) error {
			return fn(repositories.Repositories{
				Passes:   passesRepo,
				Bookings: bookingsRepo,
			})
		})
}

func TestService_ActivatePass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		email                string
		initialAssignedSlots int
		totalPassSlots       int

		data func() testData

		mocks func(
			data testData,
			unitOfWork *mock.MockIUnitOfWork,
			passesRepo *mock.MockIPasses,
			bookingsRepo *mock.MockIBookings,
			notifier *mock.MockINotifier,
		)

		assert func(
			t *testing.T,
			data testData,
			got PassActivation,
		)

		wantError     bool
		errorContains string
	}{
		{
			name:                 "Failure pass activation - initial slots greater than total slots",
			email:                "john@example.com",
			initialAssignedSlots: 5,
			totalPassSlots:       4,

			wantError:     true,
			errorContains: "initialAssignedSlots: 5 is grater than totalSlots: 4",
		},
		{
			name:                 "Failure pass activation - insert pass error",
			email:                "john@example.com",
			initialAssignedSlots: 0,
			totalPassSlots:       8,
			data:                 newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				passesRepo *mock.MockIPasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
			) {
				mockActivatePassTransaction(
					unitOfWork,
					passesRepo,
					bookingsRepo,
				)

				passesRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pass.Email,
						data.pass.TotalSlots,
					).
					Return(models.Pass{}, assert.AnError)
			},

			wantError:     true,
			errorContains: "could not insert pass",
		},

		{
			name:                 "Failure pass activation - list bookings without pass error",
			email:                "john@example.com",
			initialAssignedSlots: 2,
			totalPassSlots:       8,
			data:                 newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				passesRepo *mock.MockIPasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
			) {
				mockActivatePassTransaction(
					unitOfWork,
					passesRepo,
					bookingsRepo,
				)

				passesRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pass.Email,
						data.pass.TotalSlots,
					).
					Return(data.pass, nil)

				bookingsRepo.EXPECT().
					ListWithoutPassByEmail(
						gomock.Any(),
						data.pass.Email,
						2,
					).
					Return(nil, assert.AnError)
			},

			wantError:     true,
			errorContains: "could not ListWithoutPass",
		},

		{
			name:                 "Failure pass activation - bookings count mismatch",
			email:                "john@example.com",
			initialAssignedSlots: 2,
			totalPassSlots:       8,
			data:                 newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				passesRepo *mock.MockIPasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
			) {
				mockActivatePassTransaction(
					unitOfWork,
					passesRepo,
					bookingsRepo,
				)

				passesRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pass.Email,
						data.pass.TotalSlots,
					).
					Return(data.pass, nil)

				bookingsRepo.EXPECT().
					ListWithoutPassByEmail(
						gomock.Any(),
						data.pass.Email,
						2,
					).
					Return([]models.Booking{data.booking}, nil)
			},

			wantError:     true,
			errorContains: "initialUsedSlots should be equal to len bookingsToAssign",
		},
		{
			name:                 "Failure pass activation - update booking error",
			email:                "john@example.com",
			initialAssignedSlots: 1,
			totalPassSlots:       8,
			data:                 newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				passesRepo *mock.MockIPasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
			) {
				mockActivatePassTransaction(
					unitOfWork,
					passesRepo,
					bookingsRepo,
				)

				passesRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pass.Email,
						data.pass.TotalSlots,
					).
					Return(data.pass, nil)

				bookingsRepo.EXPECT().
					ListWithoutPassByEmail(
						gomock.Any(),
						data.pass.Email,
						1,
					).
					Return([]models.Booking{data.booking}, nil)

				bookingsRepo.EXPECT().
					Update(
						gomock.Any(),
						data.booking.ID,
						map[string]any{"pass_id": data.pass.ID},
					).
					Return(models.Booking{}, assert.AnError)
			},

			wantError:     true,
			errorContains: "could not update booking",
		},
		{
			name:                 "Failure pass activation - notifier error",
			email:                "john@example.com",
			initialAssignedSlots: 0,
			totalPassSlots:       8,
			data:                 newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				passesRepo *mock.MockIPasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
			) {
				mockActivatePassTransaction(
					unitOfWork,
					passesRepo,
					bookingsRepo,
				)

				passesRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pass.Email,
						data.pass.TotalSlots,
					).
					Return(data.pass, nil)

				notifier.EXPECT().
					NotifyPassActivation(
						data.pass.Email,
						gomock.Any(),
					).
					Return(assert.AnError)
			},

			wantError:     true,
			errorContains: "could notify pass activation",
		},
		{
			name:                 "Success pass activation - without assigning bookings",
			email:                "john@example.com",
			initialAssignedSlots: 0,
			totalPassSlots:       8,
			data:                 newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				passesRepo *mock.MockIPasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
			) {
				mockActivatePassTransaction(
					unitOfWork,
					passesRepo,
					bookingsRepo,
				)

				passesRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pass.Email,
						data.pass.TotalSlots,
					).
					Return(data.pass, nil)

				notifier.EXPECT().
					NotifyPassActivation(
						data.pass.Email,
						gomock.Any(),
					).
					Return(nil)
			},

			assert: func(
				t *testing.T,
				data testData,
				got PassActivation,
			) {
				t.Helper()

				assert.Equal(t, data.pass, got.Pass)
				assert.Empty(t, got.UpdatedBookings)
			},
		},
		{
			name:                 "Success pass activation - assign existing bookings",
			email:                "john@example.com",
			initialAssignedSlots: 1,
			totalPassSlots:       8,
			data:                 newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				passesRepo *mock.MockIPasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
			) {
				mockActivatePassTransaction(
					unitOfWork,
					passesRepo,
					bookingsRepo,
				)

				passesRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pass.Email,
						data.pass.TotalSlots,
					).
					Return(data.pass, nil)

				bookingsRepo.EXPECT().
					ListWithoutPassByEmail(
						gomock.Any(),
						data.pass.Email,
						1,
					).
					Return([]models.Booking{data.booking}, nil)

				bookingsRepo.EXPECT().
					Update(
						gomock.Any(),
						data.booking.ID,
						map[string]any{"pass_id": data.pass.ID},
					).
					DoAndReturn(func(
						_ context.Context,
						_ uuid.UUID,
						update map[string]any,
					) (models.Booking, error) {
						assert.Equal(t, data.pass.ID, update["pass_id"])

						updated := data.booking
						updated.PassID = optional.Of(data.pass.ID)
						updated.Pass = optional.Of(data.pass)

						return updated, nil
					})

				notifier.EXPECT().
					NotifyPassActivation(
						data.pass.Email,
						gomock.Any(),
					).
					Return(nil)
			},

			assert: func(
				t *testing.T,
				data testData,
				got PassActivation,
			) {
				t.Helper()

				assert.Equal(t, data.pass, got.Pass)

				require.Len(t, got.UpdatedBookings, 1)

				assert.Equal(t, data.booking.ID, got.UpdatedBookings[0].ID)

				require.True(t, got.UpdatedBookings[0].PassID.Exists())
				assert.Equal(t, data.pass.ID, got.UpdatedBookings[0].PassID.Get())

				require.True(t, got.UpdatedBookings[0].Pass.Exists())
				assert.Equal(t, data.pass.ID, got.UpdatedBookings[0].Pass.Get().ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			unitOfWork := mock.NewMockIUnitOfWork(ctrl)
			passesRepo := mock.NewMockIPasses(ctrl)
			bookingsRepo := mock.NewMockIBookings(ctrl)
			notifier := mock.NewMockINotifier(ctrl)

			var data testData

			if tt.data != nil {
				data = tt.data()
			}

			if tt.mocks != nil {
				tt.mocks(
					data,
					unitOfWork,
					passesRepo,
					bookingsRepo,
					notifier,
				)
			}

			service := NewService(
				unitOfWork,
				notifier,
			)

			got, err := service.ActivatePass(
				context.Background(),
				tt.email,
				tt.initialAssignedSlots,
				tt.totalPassSlots,
			)

			if tt.wantError {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.errorContains)

				return
			}

			require.NoError(t, err)

			if tt.assert != nil {
				tt.assert(t, data, got)
			}
		})
	}
}
