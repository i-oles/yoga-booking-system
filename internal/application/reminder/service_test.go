package reminder

import (
	"context"
	"testing"
	"time"

	"main/internal/domain/models"
	"main/internal/domain/repositories"
	"main/internal/infrastructure/sender"
	"main/mock"
	"main/pkg/optional"

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
	testToken        = "token"
	testLocationLink = "https://google.maps.com"
	testDomain       = "https://test.pl"
	testOwnerEmail   = "owner@test.pl"
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
			name: "Bookings not reminded - no classes",
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
			name: "Bookings not reminded - past class ignored",
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
			name: "Bookings not reminded - too early to remind",
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
			name: "Bookings not reminded - class without bookings",
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
			name: "Bookings not reminded - booking already reminded",
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
			name: "Bookings not reminded - booking created on class day",
			data: func() testData {
				data := newTestData()

				classStartTimeUTC := data.class.StartTime.UTC()

				data.booking.CreatedAt = time.Date(
					classStartTimeUTC.Year(),
					classStartTimeUTC.Month(),
					classStartTimeUTC.Day(),
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
			name: "Bookings not reminded - booking created previous day",
			data: func() testData {
				data := newTestData()

				prev := data.class.StartTime.Add(-24 * time.Hour).UTC()

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
					Update(gomock.Any(), data.booking.ID, gomock.Any()).
					Return(data.booking, nil)

				locationLinkProvider.EXPECT().
					GetLink(data.class.Location).
					Return("", assert.AnError)
			},
			wantError:     true,
			errorContains: "could not get location link",
		},
		{
			name: "Failure remind bookings - list bookings for pass error",
			data: func() testData {
				data := newTestData()
				data.booking.Pass = optional.Of(data.pass)
				data.booking.PassID = optional.Of(data.pass.ID)

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
					Return(data.booking, nil)

				locationLinkProvider.EXPECT().
					GetLink(data.class.Location).
					Return(testLocationLink, nil)

				bookingsRepo.EXPECT().
					ListByPassID(
						gomock.Any(),
						data.pass.ID,
					).
					Return(nil, assert.AnError)
			},

			wantError:     true,
			errorContains: "could not list bookings for pass",
		},
		{
			name: "Failure remind bookings - notifier error",
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
					Return(data.booking, nil)

				locationLinkProvider.EXPECT().
					GetLink(data.class.Location).
					Return(testLocationLink, nil)

				notifier.EXPECT().
					NotifyBookingReminder(
						gomock.Any(),
						testDomain+"/bookings/"+data.booking.ID.String()+"/cancel_form?token="+testToken,
					).
					Return(assert.AnError)
			},

			wantError:     true,
			errorContains: "could not notify booking",
		},
		{
			name: "Success remind bookings - booking without pass",
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
					Return(data.booking, nil)

				locationLinkProvider.EXPECT().
					GetLink(data.class.Location).
					Return(testLocationLink, nil)

				notifier.EXPECT().
					NotifyBookingReminder(
						gomock.Any(),
						testDomain+"/bookings/"+data.booking.ID.String()+"/cancel_form?token="+testToken,
					).
					DoAndReturn(func(
						params models.NotifierParams,
						cancelURL string,
					) error {
						assert.Equal(t, data.booking.Email, params.RecipientEmail)
						assert.Equal(t, data.booking.FirstName, params.RecipientFirstName)
						assert.Equal(t, data.booking.LastName, params.RecipientLastName)

						assert.Equal(t, data.class.ClassName, params.ClassName)
						assert.Equal(t, data.class.ClassLevel, params.ClassLevel)
						assert.Equal(t, data.class.StartTime, params.StartTime)
						assert.Equal(t, data.class.Location, params.Location)
						assert.Equal(t, testLocationLink, params.LocationLink)

						assert.Empty(t, params.PassSlots)

						assert.Contains(t, cancelURL, data.booking.ID.String())
						assert.Contains(t, cancelURL, testToken)

						return nil
					})
			},
		},
		{
			name: "Success remind bookings - booking with pass",
			data: func() testData {
				data := newTestData()
				data.booking.Pass = optional.Of(data.pass)
				data.booking.PassID = optional.Of(data.pass.ID)

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
					Return(data.booking, nil)

				locationLinkProvider.EXPECT().
					GetLink(data.class.Location).
					Return(testLocationLink, nil)

				bookingsRepo.EXPECT().
					ListByPassID(
						gomock.Any(),
						data.pass.ID,
					).
					Return([]models.Booking{data.booking}, nil)

				notifier.EXPECT().
					NotifyBookingReminder(
						gomock.Any(),
						gomock.Any(),
					).
					DoAndReturn(func(
						params models.NotifierParams,
						cancelURL string,
					) error {
						assert.Equal(t, data.booking.Email, params.RecipientEmail)
						assert.Equal(t, data.booking.FirstName, params.RecipientFirstName)
						assert.Equal(t, data.booking.LastName, params.RecipientLastName)

						assert.Equal(t, data.class.ClassName, params.ClassName)
						assert.Equal(t, data.class.ClassLevel, params.ClassLevel)
						assert.Equal(t, data.class.StartTime, params.StartTime)
						assert.Equal(t, data.class.Location, params.Location)
						assert.Equal(t, testLocationLink, params.LocationLink)

						assert.Len(t, params.PassSlots, data.pass.TotalSlots)
						assert.Contains(t, cancelURL, data.booking.ID.String())
						assert.Contains(t, cancelURL, testToken)

						return nil
					})
			},
		},
		{
			name: "Success remind bookings - multiple bookings, one already reminded is skipped",
			data: newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				classesRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				alreadyRemindedAt := time.Now().Add(-time.Hour)
				alreadyRemindedBooking := newBooking(data.class)
				alreadyRemindedBooking.RemindedAt = &alreadyRemindedAt

				classesRepo.EXPECT().
					List(gomock.Any()).
					Return([]models.Class{data.class}, nil)

				bookingsRepo.EXPECT().
					ListByClassID(
						gomock.Any(),
						data.class.ID,
					).
					Return([]models.Booking{alreadyRemindedBooking, data.booking}, nil)

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
					Return(data.booking, nil)

				locationLinkProvider.EXPECT().
					GetLink(data.class.Location).
					Return(testLocationLink, nil)

				notifier.EXPECT().
					NotifyBookingReminder(
						gomock.Any(),
						gomock.Any(),
					).
					Return(nil)
			},
		},
		{
			name: "Failure remind bookings - stops processing remaining classes after first error",
			data: newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				classesRepo *mock.MockIClasses,
				bookingsRepo *mock.MockIBookings,
				notifier *mock.MockINotifier,
				locationLinkProvider *mock.MockILinkProvider,
			) {
				secondClass := newClass()

				classesRepo.EXPECT().
					List(gomock.Any()).
					Return([]models.Class{data.class, secondClass}, nil)

				bookingsRepo.EXPECT().
					ListByClassID(
						gomock.Any(),
						data.class.ID,
					).
					Return(nil, assert.AnError)
			},

			wantError:     true,
			errorContains: "could not send reminders for class",
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
				mock.NewMockIEmailSender(ctrl),
				locationLinkProvider,
				testDomain,
				testOwnerEmail,
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

func TestService_RemindToScheduleMoreClasses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string

		mocks func(
			classesRepo *mock.MockIClasses,
			emailSender *mock.MockIEmailSender,
		)

		wantError     bool
		errorContains string
	}{
		{
			name: "Failure - could not count upcoming classes",

			mocks: func(
				classesRepo *mock.MockIClasses,
				emailSender *mock.MockIEmailSender,
			) {
				classesRepo.EXPECT().
					CountUpcomingClasses(gomock.Any()).
					Return(0, assert.AnError)
			},

			wantError:     true,
			errorContains: "could not count upcoming classes",
		},
		{
			name: "Success - enough upcoming classes, no reminder sent",

			mocks: func(
				classesRepo *mock.MockIClasses,
				emailSender *mock.MockIEmailSender,
			) {
				classesRepo.EXPECT().
					CountUpcomingClasses(gomock.Any()).
					Return(5, nil)
			},
		},
		{
			name: "Success - upcoming classes count at threshold sends reminder",

			mocks: func(
				classesRepo *mock.MockIClasses,
				emailSender *mock.MockIEmailSender,
			) {
				classesRepo.EXPECT().
					CountUpcomingClasses(gomock.Any()).
					Return(4, nil)

				emailSender.EXPECT().
					Send(gomock.Any()).
					DoAndReturn(func(messages ...sender.Message) error {
						require.Len(t, messages, 1)

						msg := messages[0]
						assert.Equal(t, testOwnerEmail, msg.From)
						assert.Equal(t, testOwnerEmail, msg.To)
						assert.Contains(t, msg.Subject, "4")

						return nil
					})
			},
		},
		{
			name: "Success - no upcoming classes sends reminder",

			mocks: func(
				classesRepo *mock.MockIClasses,
				emailSender *mock.MockIEmailSender,
			) {
				classesRepo.EXPECT().
					CountUpcomingClasses(gomock.Any()).
					Return(0, nil)

				emailSender.EXPECT().
					Send(gomock.Any()).
					Return(nil)
			},
		},
		{
			name: "Failure - could not send reminder",

			mocks: func(
				classesRepo *mock.MockIClasses,
				emailSender *mock.MockIEmailSender,
			) {
				classesRepo.EXPECT().
					CountUpcomingClasses(gomock.Any()).
					Return(1, nil)

				emailSender.EXPECT().
					Send(gomock.Any()).
					Return(assert.AnError)
			},

			wantError:     true,
			errorContains: "could not send reminder to owner to schedule more classes",
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
			emailSender := mock.NewMockIEmailSender(ctrl)
			locationLinkProvider := mock.NewMockILinkProvider(ctrl)

			if tt.mocks != nil {
				tt.mocks(classesRepo, emailSender)
			}

			service := New(
				unitOfWork,
				classesRepo,
				bookingsRepo,
				notifier,
				emailSender,
				locationLinkProvider,
				testDomain,
				testOwnerEmail,
			)

			err := service.RemindToScheduleMoreClasses(context.Background())

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
		{
			name: "mixed extreme timezones two real days apart",
			a:    time.Date(2026, 9, 1, 23, 0, 0, 0, time.UTC).In(time.FixedZone("UTC+14", 14*3600)),
			b:    time.Date(2026, 9, 3, 1, 0, 0, 0, time.UTC).In(time.FixedZone("UTC-12", -12*3600)),
			want: false,
		},
		{
			name: "mixed timezones same real day",
			a:    time.Date(2026, 9, 3, 8, 0, 0, 0, time.UTC).In(time.FixedZone("UTC+9", 9*3600)),
			b:    time.Date(2026, 9, 3, 20, 0, 0, 0, time.UTC).In(time.FixedZone("UTC-5", -5*3600)),
			want: true,
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
