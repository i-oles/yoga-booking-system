package pendingbookings

import (
	"context"
	"testing"
	"time"

	"main/internal/domain/models"
	"main/internal/domain/repositories"
	"main/internal/infrastructure/errs"
	"main/mock"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func mockCreatePendingBookingTransaction(
	unitOfWork *mock.MockIUnitOfWork,
	pendingBookingsRepo *mock.MockIPendingBookings,
	bookingsRepo *mock.MockIBookings,
	classesRepo *mock.MockIClasses,
) {
	unitOfWork.EXPECT().
		WithTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			ctx context.Context,
			fn func(repositories.Repositories) error,
		) error {
			return fn(repositories.Repositories{
				PendingBookings: pendingBookingsRepo,
				Bookings:        bookingsRepo,
				Classes:         classesRepo,
			})
		})
}

var (
	testToken        = "token"
	testLocationLink = "https://google.maps.com"
	testDomain       = "https://test.pl"
)

type testData struct {
	class                models.Class
	booking              models.Booking
	pendingBookingParams models.PendingBookingParams
}

func newTestData() testData {
	class := newClass()
	pendingBookingParams := models.PendingBookingParams{
		ClassID:   class.ID,
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
	}

	booking := newBooking(class, pendingBookingParams)

	return testData{
		class:                class,
		booking:              booking,
		pendingBookingParams: pendingBookingParams,
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

func newPendingBooking(class models.Class) models.PendingBooking {
	return models.PendingBooking{
		ID:                uuid.New(),
		ClassID:           class.ID,
		Class:             class,
		ConfirmationToken: testToken,
		FirstName:         "John",
		LastName:          "Doe",
		Email:             "john@example.com",
		CreatedAt:         time.Now().Add(-20 * time.Minute),
	}
}

func newBooking(
	class models.Class,
	params models.PendingBookingParams,
) models.Booking {
	return models.Booking{
		ID:                uuid.New(),
		ClassID:           class.ID,
		Class:             class,
		ConfirmationToken: testToken,
		FirstName:         params.FirstName,
		LastName:          params.LastName,
		Email:             params.Email,
		CreatedAt:         time.Now(),
	}
}

func TestService_CreatePendingBooking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		data  func() testData
		mocks func(
			data testData,
			unitOfWork *mock.MockIUnitOfWork,
			tokenGenerator *mock.MockITokenGenerator,
			pendingBookingsRepo *mock.MockIPendingBookings,
			bookingsRepo *mock.MockIBookings,
			classesRepo *mock.MockIClasses,
			notifier *mock.MockINotifier,
		)
		wantError     bool
		errorContains string
	}{
		{
			name: "Failure pending booking creation - booking already exists",
			data: newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				tokenGenerator *mock.MockITokenGenerator,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				classesRepo *mock.MockIClasses,
				notifier *mock.MockINotifier,
			) {
				mockCreatePendingBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					classesRepo,
				)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBookingParams.ClassID,
						data.pendingBookingParams.Email,
					).
					Return(data.booking, nil)
			},

			wantError:     true,
			errorContains: "already exists",
		},
		{
			name: "Failure pending booking creation - get booking error",
			data: newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				tokenGenerator *mock.MockITokenGenerator,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				classesRepo *mock.MockIClasses,
				notifier *mock.MockINotifier,
			) {
				mockCreatePendingBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					classesRepo,
				)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBookingParams.ClassID,
						data.pendingBookingParams.Email,
					).
					Return(models.Booking{}, assert.AnError)
			},

			wantError:     true,
			errorContains: "could not get booking",
		},
		{
			name: "Failure pending booking creation - list pending bookings error",
			data: newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				tokenGenerator *mock.MockITokenGenerator,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				classesRepo *mock.MockIClasses,
				notifier *mock.MockINotifier,
			) {
				mockCreatePendingBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					classesRepo,
				)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBookingParams.ClassID,
						data.pendingBookingParams.Email,
					).
					Return(models.Booking{}, errs.ErrNotFound)

				pendingBookingsRepo.EXPECT().
					List(gomock.Any()).
					Return(nil, assert.AnError)
			},

			wantError:     true,
			errorContains: "could not list pending bookings",
		},
		{
			name: "Failure pending booking creation - pending bookings limit exceeded",
			data: newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				tokenGenerator *mock.MockITokenGenerator,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				classesRepo *mock.MockIClasses,
				notifier *mock.MockINotifier,
			) {
				mockCreatePendingBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					classesRepo,
				)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBookingParams.ClassID,
						data.pendingBookingParams.Email,
					).
					Return(models.Booking{}, errs.ErrNotFound)

				pendingBookings := make([]models.PendingBooking, allowedTotalPendingBookingsLimit)

				pendingBookingsRepo.EXPECT().
					List(gomock.Any()).
					Return(pendingBookings, nil)
			},

			wantError:     true,
			errorContains: "limit: 200 of pending bookings exceeded",
		},
		{
			name: "Failure pending booking creation - too many pending bookings",
			data: newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				tokenGenerator *mock.MockITokenGenerator,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				classesRepo *mock.MockIClasses,
				notifier *mock.MockINotifier,
			) {
				mockCreatePendingBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					classesRepo,
				)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBookingParams.ClassID,
						data.pendingBookingParams.Email,
					).
					Return(models.Booking{}, errs.ErrNotFound)

				pendingBookingsRepo.EXPECT().
					List(gomock.Any()).
					Return([]models.PendingBooking{
						newPendingBooking(data.class),
						newPendingBooking(data.class),
					}, nil)
			},

			wantError:     true,
			errorContains: "found 2 pending bookings per user",
		},
		{
			name: "Failure pending booking creation - count bookings error",
			data: newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				tokenGenerator *mock.MockITokenGenerator,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				classesRepo *mock.MockIClasses,
				notifier *mock.MockINotifier,
			) {
				mockCreatePendingBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					classesRepo,
				)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBookingParams.ClassID,
						data.pendingBookingParams.Email,
					).
					Return(models.Booking{}, errs.ErrNotFound)

				pendingBookingsRepo.EXPECT().
					List(gomock.Any()).
					Return(nil, nil)

				bookingsRepo.EXPECT().
					CountForClassID(
						gomock.Any(),
						data.class.ID,
					).
					Return(0, assert.AnError)
			},

			wantError:     true,
			errorContains: "could not count bookings",
		},
		{
			name: "Failure pending booking creation - get class error",
			data: newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				tokenGenerator *mock.MockITokenGenerator,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				classesRepo *mock.MockIClasses,
				notifier *mock.MockINotifier,
			) {
				mockCreatePendingBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					classesRepo,
				)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBookingParams.ClassID,
						data.pendingBookingParams.Email,
					).
					Return(models.Booking{}, errs.ErrNotFound)

				pendingBookingsRepo.EXPECT().
					List(gomock.Any()).
					Return(nil, nil)

				bookingsRepo.EXPECT().
					CountForClassID(
						gomock.Any(),
						data.class.ID,
					).
					Return(3, nil)

				classesRepo.EXPECT().
					Get(
						gomock.Any(),
						data.class.ID,
					).
					Return(models.Class{}, assert.AnError)
			},

			wantError:     true,
			errorContains: "could not get class",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			unitOfWork := mock.NewMockIUnitOfWork(ctrl)
			tokenGenerator := mock.NewMockITokenGenerator(ctrl)

			pendingBookingsRepo := mock.NewMockIPendingBookings(ctrl)
			bookingsRepo := mock.NewMockIBookings(ctrl)
			classesRepo := mock.NewMockIClasses(ctrl)

			notifier := mock.NewMockINotifier(ctrl)

			var data testData
			if tt.data != nil {
				data = tt.data()
			}

			if tt.mocks != nil {
				tt.mocks(
					data,
					unitOfWork,
					tokenGenerator,
					pendingBookingsRepo,
					bookingsRepo,
					classesRepo,
					notifier,
				)
			}

			service := NewService(
				unitOfWork,
				tokenGenerator,
				notifier,
				"https://test.pl",
			)

			err := service.CreatePendingBooking(
				context.Background(),
				data.pendingBookingParams,
			)

			if tt.wantError {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.errorContains)

				return
			}

			require.NoError(t, err)
		})
	}
}
