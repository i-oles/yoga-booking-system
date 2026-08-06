package reminder

import (
	"context"
	"testing"
	"time"

	"main/internal/domain/models"
	"main/internal/domain/repositories"
	"main/mock"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func mockRemindBookingTransaction(
	unitOfWork *mock.MockIUnitOfWork,
	bookingsRepo *mock.MockIBookings,
) {
	unitOfWork.EXPECT().
		WithTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			ctx context.Context,
			fn func(repositories.Repositories) error,
		) error {
			return fn(repositories.Repositories{
				Bookings: bookingsRepo,
			})
		})
}

var (
	testToken  = "token"
	testDomain = "https://test.pl"
)

type testData struct {
	class   models.Class
	booking models.Booking
	pass    models.Pass
}

func newTestData() testData {
	class := newClass()
	pass := newPass()
	booking := newBooking(class)

	return testData{
		class:   class,
		booking: booking,
		pass:    pass,
	}
}

func newClass() models.Class {
	return models.Class{
		ID:          uuid.New(),
		StartTime:   time.Now().Add(12 * time.Hour),
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
		ConfirmationToken: testToken,
		FirstName:         "John",
		LastName:          "Doe",
		Email:             "john@example.com",
		CreatedAt:         time.Now().Add(-48 * time.Hour),
	}
}

func newPass() models.Pass {
	return models.Pass{
		ID:         1,
		Email:      "john@example.com",
		TotalSlots: 8,
		CreatedAt:  time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt:  time.Now(),
	}
}

func TestService_RemindBookings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string

		data func() testData

		mocks func(
			data testData,
			unitOfWork *mock.MockIUnitOfWork,
			classesRepo *mock.MockIClasses,
			bookingsRepo *mock.MockIBookings,
			notifier *mock.MockINotifier,
			locationLinkProvider *mock.MockILinkProvider,
		)

		wantError     bool
		errorContains string
	}{
		{
			name: "Failure remind bookings - list classes error",
			data: newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				classesRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classesRepo.EXPECT().
					List(gomock.Any()).
					Return(nil, assert.AnError)
			},

			wantError:     true,
			errorContains: "could not list classes",
		},
		{
			name: "Success remind bookings - no classes",
			data: newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				classesRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classesRepo.EXPECT().
					List(gomock.Any()).
					Return(nil, nil)
			},
		},
		{
			name: "Success remind bookings - past class ignored",
			data: func() testData {
				data := newTestData()
				data.class.StartTime = time.Now().Add(-time.Hour)

				return data
			},

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				classesRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classesRepo.EXPECT().
					List(gomock.Any()).
					Return([]models.Class{data.class}, nil)
			},
		},
		{
			name: "Success remind bookings - too early to remind",
			data: func() testData {
				data := newTestData()
				data.class.StartTime = time.Now().Add(48 * time.Hour)

				return data
			},

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				classesRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classesRepo.EXPECT().
					List(gomock.Any()).
					Return([]models.Class{data.class}, nil)
			},
		},
		{
			name: "Success remind bookings - class without bookings",
			data: newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				classesRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classesRepo.EXPECT().
					List(gomock.Any()).
					Return([]models.Class{data.class}, nil)

				bookingsRepo.EXPECT().
					ListByClassID(
						gomock.Any(),
						data.class.ID,
					).
					Return(nil, nil)
			},
		},
		{
			name: "Failure remind bookings - list bookings error",
			data: newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				classesRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classesRepo.EXPECT().
					List(gomock.Any()).
					Return([]models.Class{data.class}, nil)

				bookingsRepo.EXPECT().
					ListByClassID(
						gomock.Any(),
						data.class.ID,
					).
					Return(nil, assert.AnError)
			},

			wantError:     true,
			errorContains: "could not list bookings",
		},
		{
			name: "Success remind bookings - booking already reminded",
			data: func() testData {
				data := newTestData()
				now := time.Now()
				data.booking.RemindedAt = &now

				return data
			},

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				classesRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classesRepo.EXPECT().
					List(gomock.Any()).
					Return([]models.Class{data.class}, nil)

				bookingsRepo.EXPECT().
					ListByClassID(
						gomock.Any(),
						data.class.ID,
					).
					Return([]models.Booking{data.booking}, nil)
			},
		},
		{
			name: "Success remind bookings - booking created on class day",
			data: func() testData {
				data := newTestData()

				data.class.StartTime = time.Now().Add(12 * time.Hour)

				data.booking.CreatedAt = time.Date(
					data.class.StartTime.Year(),
					data.class.StartTime.Month(),
					data.class.StartTime.Day(),
					10, 0, 0, 0,
					time.UTC,
				)

				return data
			},

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				classesRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classesRepo.EXPECT().
					List(gomock.Any()).
					Return([]models.Class{data.class}, nil)

				bookingsRepo.EXPECT().
					ListByClassID(
						gomock.Any(),
						data.class.ID,
					).
					Return([]models.Booking{data.booking}, nil)
			},
		},
		{
			name: "Success remind bookings - booking created previous day",
			data: func() testData {
				data := newTestData()

				data.class.StartTime = time.Now().Add(12 * time.Hour)

				prev := data.class.StartTime.Add(-24 * time.Hour)

				data.booking.CreatedAt = time.Date(
					prev.Year(),
					prev.Month(),
					prev.Day(),
					12, 0, 0, 0,
					time.UTC,
				)

				return data
			},

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				classesRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classesRepo.EXPECT().
					List(gomock.Any()).
					Return([]models.Class{data.class}, nil)

				bookingsRepo.EXPECT().
					ListByClassID(
						gomock.Any(),
						data.class.ID,
					).
					Return([]models.Booking{data.booking}, nil)
			},
		},
		{
			name: "Failure remind bookings - update booking error",
			data: newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				classesRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classesRepo.EXPECT().
					List(gomock.Any()).
					Return([]models.Class{data.class}, nil)

				bookingsRepo.EXPECT().
					ListByClassID(
						gomock.Any(),
						data.class.ID,
					).
					Return([]models.Booking{data.booking}, nil)

				mockRemindBookingTransaction(
					unitOfWork,
					bookingsRepo,
				)

				bookingsRepo.EXPECT().
					Update(
						gomock.Any(),
						data.booking.ID,
						gomock.Any(),
					).
					Return(models.Booking{}, assert.AnError)
			},

			wantError:     true,
			errorContains: "could not update booking",
		},
		{
			name: "Failure remind bookings - get location link error",
			data: newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				classesRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				classesRepo.EXPECT().
					List(gomock.Any()).
					Return([]models.Class{data.class}, nil)

				bookingsRepo.EXPECT().
					ListByClassID(
						gomock.Any(),
						data.class.ID,
					).
					Return([]models.Booking{data.booking}, nil)

				mockRemindBookingTransaction(
					unitOfWork,
					bookingsRepo,
				)

				bookingsRepo.EXPECT().
					Update(
						gomock.Any(),
						data.booking.ID,
						gomock.Any(),
					).
					DoAndReturn(func(
						_ context.Context,
						_ uuid.UUID,
						_ map[string]any,
					) (models.Booking, error) {
						return data.booking, nil
					})

				locationLinkProvider.EXPECT().
					GetLink(data.class.Location).
					Return("", assert.AnError)
			},

			wantError:     true,
			errorContains: "could not get location link",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			unitOfWork := mock.NewMockIUnitOfWork(ctrl)
			classesRepo := mock.NewMockIClasses(ctrl)
			bookingsRepo := mock.NewMockIBookings(ctrl)
			notifier := mock.NewMockINotifier(ctrl)
			locationLinkProvider := mock.NewMockILinkProvider(ctrl)

			var data testData
			if tt.data != nil {
				data = tt.data()
			}

			if tt.mocks != nil {
				tt.mocks(
					data,
					unitOfWork,
					classesRepo,
					bookingsRepo,
					notifier,
					locationLinkProvider,
				)
			}

			service := New(
				unitOfWork,
				classesRepo,
				bookingsRepo,
				notifier,
				locationLinkProvider,
				testDomain,
			)

			err := service.RemindBookings(context.Background())

			if tt.wantError {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.errorContains)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestIsTimeToRemind(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		start time.Time
		want  bool
	}{
		{
			name:  "within 24h (10h)",
			start: now.Add(10 * time.Hour),
			want:  true,
		},
		{
			name:  "just below 24h",
			start: now.Add(24*time.Hour - time.Nanosecond),
			want:  true,
		},
		{
			name:  "exactly 24h",
			start: now.Add(24 * time.Hour),
			want:  false,
		},
		{
			name:  "just above 24h",
			start: now.Add(24*time.Hour + time.Nanosecond),
			want:  false,
		},
		{
			name:  "exactly now",
			start: now,
			want:  false,
		},
		{
			name:  "in the past",
			start: now.Add(-1 * time.Hour),
			want:  false,
		},
		{
			name:  "zero time",
			start: time.Time{},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isTimeToRemind(tt.start, now)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsBookedSameOrPreviousDayAsClassDay(t *testing.T) {
	t.Parallel()

	loc := time.UTC

	tests := []struct {
		name string
		a    time.Time
		b    time.Time
		want bool
	}{
		{
			name: "same day",
			a:    time.Date(2026, 4, 6, 10, 0, 0, 0, loc),
			b:    time.Date(2026, 4, 6, 18, 0, 0, 0, loc),
			want: true,
		},
		{
			name: "previous day",
			a:    time.Date(2026, 4, 5, 23, 59, 0, 0, loc),
			b:    time.Date(2026, 4, 6, 0, 1, 0, 0, loc),
			want: true,
		},
		{
			name: "two days before",
			a:    time.Date(2026, 4, 4, 12, 0, 0, 0, loc),
			b:    time.Date(2026, 4, 6, 12, 0, 0, 0, loc),
			want: false,
		},
		{
			name: "next day",
			a:    time.Date(2026, 4, 7, 10, 0, 0, 0, loc),
			b:    time.Date(2026, 4, 6, 10, 0, 0, 0, loc),
			want: false,
		},
		{
			name: "zero time values",
			a:    time.Time{},
			b:    time.Time{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isBookedSameOrPreviousDayAsClassDay(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
