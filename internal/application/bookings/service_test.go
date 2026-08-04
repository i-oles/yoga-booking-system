package bookings

import (
	"context"
	"testing"
	"time"

	"main/internal/domain/models"
	"main/internal/domain/repositories"
	"main/internal/infrastructure/errs"
	"main/mock"
	"main/pkg/optional"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	testToken        = "token"
	testLocationLink = "https://google.maps.com"
	testDomain       = "https://test.pl"
)

type testData struct {
	class          models.Class
	pendingBooking models.PendingBooking
	booking        models.Booking
	contact        models.Contact
	passes         []models.Pass
}

func newTestData() testData {
	class := newClass()
	pendingBooking := newPendingBooking(class)
	booking := newBooking(pendingBooking)

	return testData{
		class:          class,
		pendingBooking: pendingBooking,
		booking:        booking,
		contact:        newContact(),
		passes:         []models.Pass{newPass()},
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

func newBooking(pb models.PendingBooking) models.Booking {
	return models.Booking{
		ID:                uuid.New(),
		ClassID:           pb.ClassID,
		Class:             pb.Class,
		ConfirmationToken: pb.ConfirmationToken,
		FirstName:         pb.FirstName,
		LastName:          pb.LastName,
		Email:             pb.Email,
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

func newContact() models.Contact {
	return models.Contact{
		ID:        1,
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
	}
}

func mockCreateBookingTransaction(
	uow *mock.MockIUnitOfWork,
	pendingBookingsRepo *mock.MockIPendingBookings,
	bookingsRepo *mock.MockIBookings,
	contactsRepo *mock.MockIContacts,
	passesRepo *mock.MockIPasses,
) {
	uow.EXPECT().
		WithTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			ctx context.Context,
			fn func(repositories.Repositories) error,
		) error {
			return fn(repositories.Repositories{
				PendingBookings: pendingBookingsRepo,
				Bookings:        bookingsRepo,
				Contacts:        contactsRepo,
				Passes:          passesRepo,
			})
		})
}

func TestService_CreateBooking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		token string
		data  func() testData
		mocks func(
			data testData,
			unitOfWork *mock.MockIUnitOfWork,
			pendingBookingsRepo *mock.MockIPendingBookings,
			bookingsRepo *mock.MockIBookings,
			contactsRepo *mock.MockIContacts,
			passesRepo *mock.MockIPasses,
			locationLinkProvider *mock.MockILinkProvider,
			notifier *mock.MockINotifier,
		)
		assert func(
			t *testing.T,
			data testData,
			got BookingCreation,
		)
		wantError     bool
		errorContains string
	}{
		{
			name:  "Failure booking creation - pending booking not found error",
			token: testToken,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(models.PendingBooking{}, errs.ErrNotFound)
			},
			wantError:     true,
			errorContains: "pending booking for token: token not found",
		},
		{
			name:  "Failure booking creation - pending booking repository error",
			token: testToken,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(models.PendingBooking{}, assert.AnError)
			},
			wantError:     true,
			errorContains: "could not get pending booking",
		},
		{
			name:  "Failure booking creation - booking already exists error",
			token: testToken,
			data:  newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(data.pendingBooking, nil)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBooking.ClassID,
						data.pendingBooking.Email,
					).
					Return(models.Booking{}, nil)
			},
			wantError:     true,
			errorContains: "booking already exists",
		},
		{
			name:  "Failure booking creation - booking not found error",
			token: testToken,
			data:  newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(data.pendingBooking, nil)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBooking.ClassID,
						data.pendingBooking.Email,
					).
					Return(models.Booking{}, assert.AnError)
			},
			wantError:     true,
			errorContains: "could not get booking for email",
		},
		{
			name:  "Failure booking creation - class expired error",
			token: testToken,
			data: func() testData {
				class := newClass()
				class.StartTime = class.StartTime.Add(-100 * time.Hour)
				pendingBooking := newPendingBooking(class)
				booking := newBooking(pendingBooking)

				return testData{
					class:          class,
					pendingBooking: pendingBooking,
					booking:        booking,
					passes:         []models.Pass{newPass()},
					contact:        newContact(),
				}
			},
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(data.pendingBooking, nil)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBooking.ClassID,
						data.pendingBooking.Email,
					).
					Return(models.Booking{}, errs.ErrNotFound)
			},
			wantError:     true,
			errorContains: "has expired",
		},
		{
			name:  "Failure booking creation - count bookings error",
			token: testToken,
			data:  newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(data.pendingBooking, nil)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBooking.ClassID,
						data.pendingBooking.Email,
					).
					Return(models.Booking{}, errs.ErrNotFound)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), data.pendingBooking.Class.ID).
					Return(0, assert.AnError)
			},
			wantError:     true,
			errorContains: "could not count bookings for class",
		},
		{
			name:  "Failure booking creation - class is full error",
			token: testToken,
			data:  newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(data.pendingBooking, nil)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBooking.ClassID,
						data.pendingBooking.Email,
					).
					Return(models.Booking{}, errs.ErrNotFound)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), data.pendingBooking.Class.ID).
					Return(10, nil)
			},

			wantError:     true,
			errorContains: "max capacity of class",
		},
		{
			name:  "Failure booking creation - insert contact error",
			token: testToken,
			data:  newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(data.pendingBooking, nil)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBooking.ClassID,
						data.pendingBooking.Email,
					).
					Return(models.Booking{}, errs.ErrNotFound)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), data.pendingBooking.Class.ID).
					Return(0, nil)

				contactsRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pendingBooking.Email,
						data.pendingBooking.FirstName,
						data.pendingBooking.LastName,
					).
					Return(models.Contact{}, assert.AnError)
			},
			wantError:     true,
			errorContains: "could not insert contact",
		},
		{
			name:  "Failure booking creation - list passes error",
			token: testToken,
			data:  newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(data.pendingBooking, nil)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(gomock.Any(), data.pendingBooking.ClassID, data.pendingBooking.Email).
					Return(models.Booking{}, errs.ErrNotFound)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), data.pendingBooking.Class.ID).
					Return(0, nil)

				contactsRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pendingBooking.Email,
						data.pendingBooking.FirstName,
						data.pendingBooking.LastName,
					).
					Return(models.Contact{}, nil)

				passesRepo.EXPECT().
					ListByEmail(gomock.Any(), data.pendingBooking.Email, threeLastPasses).
					Return(nil, assert.AnError)
			},

			wantError:     true,
			errorContains: "could not get pass",
		},
		{
			name:  "Failure booking creation - count bookings for pass error",
			token: testToken,
			data:  newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(data.pendingBooking, nil)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(gomock.Any(), data.pendingBooking.ClassID, data.pendingBooking.Email).
					Return(models.Booking{}, errs.ErrNotFound)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), data.pendingBooking.Class.ID).
					Return(0, nil)

				contactsRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pendingBooking.Email,
						data.pendingBooking.FirstName,
						data.pendingBooking.LastName,
					).
					Return(models.Contact{}, nil)

				passesRepo.EXPECT().
					ListByEmail(gomock.Any(), data.pendingBooking.Email, threeLastPasses).
					Return([]models.Pass{data.passes[0]}, nil)

				bookingsRepo.EXPECT().
					CountForPassID(gomock.Any(), data.passes[0].ID).
					Return(0, assert.AnError)
			},

			wantError:     true,
			errorContains: "could not count bookings for passID",
		},
		{
			name:  "Failure booking creation - insert booking error",
			token: testToken,
			data:  newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(data.pendingBooking, nil)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(gomock.Any(), data.pendingBooking.ClassID, data.pendingBooking.Email).
					Return(models.Booking{}, errs.ErrNotFound)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), data.pendingBooking.Class.ID).
					Return(0, nil)

				contactsRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pendingBooking.Email,
						data.pendingBooking.FirstName,
						data.pendingBooking.LastName,
					).
					Return(models.Contact{}, nil)

				passesRepo.EXPECT().
					ListByEmail(gomock.Any(), data.pendingBooking.Email, threeLastPasses).
					Return(nil, nil)

				bookingsRepo.EXPECT().
					Insert(gomock.Any(), gomock.Any()).
					Return(uuid.Nil, assert.AnError)
			},

			wantError:     true,
			errorContains: "could not insert booking",
		},
		{
			name:  "Failure booking creation - location link provider error",
			token: testToken,
			data:  newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(data.pendingBooking, nil)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBooking.ClassID,
						data.pendingBooking.Email,
					).
					Return(models.Booking{}, errs.ErrNotFound)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), data.pendingBooking.Class.ID).
					Return(0, nil)

				contactsRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pendingBooking.Email,
						data.pendingBooking.FirstName,
						data.pendingBooking.LastName,
					).
					Return(data.contact, nil)

				passesRepo.EXPECT().
					ListByEmail(gomock.Any(), data.pendingBooking.Email, threeLastPasses).
					Return(nil, nil)

				bookingsRepo.EXPECT().
					Insert(gomock.Any(), gomock.Any()).
					Return(data.booking.ID, nil)

				locationLinkProvider.EXPECT().
					GetLink(data.booking.Class.Location).
					Return("", assert.AnError)
			},
			wantError:     true,
			errorContains: "could not resolve location link",
		},
		{
			name:  "Failure booking creation - notifier error",
			token: testToken,
			data:  newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(data.pendingBooking, nil)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBooking.ClassID,
						data.pendingBooking.Email,
					).
					Return(models.Booking{}, errs.ErrNotFound)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), data.pendingBooking.Class.ID).
					Return(0, nil)

				contactsRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pendingBooking.Email,
						data.pendingBooking.FirstName,
						data.pendingBooking.LastName,
					).
					Return(data.contact, nil)

				passesRepo.EXPECT().
					ListByEmail(gomock.Any(), data.pendingBooking.Email, threeLastPasses).
					Return(nil, nil)

				bookingsRepo.EXPECT().
					Insert(gomock.Any(), gomock.Any()).
					Return(data.booking.ID, nil)

				locationLinkProvider.EXPECT().
					GetLink(data.booking.Class.Location).
					Return(testLocationLink, nil)

				notifier.EXPECT().
					NotifyBookingConfirmation(gomock.Any(), gomock.Any()).
					Return(assert.AnError)
			},

			wantError:     true,
			errorContains: "could not send confirmation email",
		},
		{
			name:  "Success booking creation - contact already exists, without pass",
			token: testToken,
			data:  newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(data.pendingBooking, nil)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBooking.ClassID,
						data.pendingBooking.Email,
					).
					Return(models.Booking{}, errs.ErrNotFound)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), data.pendingBooking.Class.ID).
					Return(0, nil)

				contactsRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pendingBooking.Email,
						data.pendingBooking.FirstName,
						data.pendingBooking.LastName,
					).
					Return(data.contact, errs.ErrAlreadyExist)

				passesRepo.EXPECT().
					ListByEmail(gomock.Any(), data.pendingBooking.Email, threeLastPasses).
					Return(nil, nil)

				bookingsRepo.EXPECT().
					Insert(gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						booking models.Booking,
					) (uuid.UUID, error) {
						assert.Equal(t, data.pendingBooking.ClassID, booking.ClassID)
						assert.Equal(t, data.pendingBooking.Email, booking.Email)
						assert.Equal(t, data.pendingBooking.FirstName, booking.FirstName)
						assert.Equal(t, data.pendingBooking.LastName, booking.LastName)
						assert.Equal(t, data.pendingBooking.ConfirmationToken, booking.ConfirmationToken)

						return booking.ID, nil
					})

				locationLinkProvider.EXPECT().
					GetLink(data.booking.Class.Location).
					Return(testLocationLink, nil)

				notifier.EXPECT().
					NotifyBookingConfirmation(
						gomock.Any(),
						gomock.Any(),
					).
					DoAndReturn(func(
						params models.NotifierParams,
						cancellationLink string,
					) error {
						assert.Equal(t, data.pendingBooking.Email, params.RecipientEmail)
						assert.Equal(t, data.pendingBooking.FirstName, params.RecipientFirstName)
						assert.Equal(t, data.pendingBooking.LastName, params.RecipientLastName)
						assert.Equal(t, data.pendingBooking.Class.ClassName, params.ClassName)

						assert.Contains(t, cancellationLink, "/bookings/")
						assert.Contains(t, cancellationLink, "/cancel_form?token="+testToken)

						return nil
					})
			},
			assert: func(
				t *testing.T,
				data testData,
				got BookingCreation,
			) {
				t.Helper()

				assert.Equal(t, data.class.ID, got.Class.ID)
				assert.Equal(t, data.class.StartTime, got.Class.StartTime)
				assert.Equal(t, data.class.ClassName, got.Class.ClassName)
				assert.Equal(t, data.class.ClassLevel, got.Class.ClassLevel)
				assert.Equal(t, data.class.MaxCapacity, got.Class.MaxCapacity)
				assert.Equal(t, data.class.Location, got.Class.Location)
				assert.Equal(t, testLocationLink, got.Class.LocationLink)
			},
		},
		{
			name:  "Success booking creation - contact does not extst, without pass",
			token: testToken,
			data:  newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(data.pendingBooking, nil)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBooking.ClassID,
						data.pendingBooking.Email,
					).
					Return(models.Booking{}, errs.ErrNotFound)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), data.pendingBooking.Class.ID).
					Return(0, nil)

				contactsRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pendingBooking.Email,
						data.pendingBooking.FirstName,
						data.pendingBooking.LastName,
					).
					Return(data.contact, nil)

				passesRepo.EXPECT().
					ListByEmail(gomock.Any(), data.pendingBooking.Email, threeLastPasses).
					Return(nil, nil)

				bookingsRepo.EXPECT().
					Insert(gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						booking models.Booking,
					) (uuid.UUID, error) {
						assert.Equal(t, data.pendingBooking.ClassID, booking.ClassID)
						assert.Equal(t, data.pendingBooking.Email, booking.Email)
						assert.Equal(t, data.pendingBooking.FirstName, booking.FirstName)
						assert.Equal(t, data.pendingBooking.LastName, booking.LastName)
						assert.Equal(t, data.pendingBooking.ConfirmationToken, booking.ConfirmationToken)

						return booking.ID, nil
					})

				locationLinkProvider.EXPECT().
					GetLink(data.booking.Class.Location).
					Return(testLocationLink, nil)

				notifier.EXPECT().
					NotifyBookingConfirmation(
						gomock.Any(),
						gomock.Any(),
					).
					DoAndReturn(func(
						params models.NotifierParams,
						cancellationLink string,
					) error {
						assert.Equal(t, data.pendingBooking.Email, params.RecipientEmail)
						assert.Equal(t, data.pendingBooking.FirstName, params.RecipientFirstName)
						assert.Equal(t, data.pendingBooking.LastName, params.RecipientLastName)
						assert.Equal(t, data.pendingBooking.Class.ClassName, params.ClassName)

						assert.Contains(t, cancellationLink, "/bookings/")
						assert.Contains(t, cancellationLink, "/cancel_form?token="+testToken)

						return nil
					})
			},

			assert: func(
				t *testing.T,
				data testData,
				got BookingCreation,
			) {
				t.Helper()
				assert.Equal(t, data.class.ID, got.Class.ID)
				assert.Equal(t, data.class.StartTime, got.Class.StartTime)
				assert.Equal(t, data.class.ClassName, got.Class.ClassName)
				assert.Equal(t, data.class.ClassLevel, got.Class.ClassLevel)
				assert.Equal(t, data.class.MaxCapacity, got.Class.MaxCapacity)
				assert.Equal(t, data.class.Location, got.Class.Location)
				assert.Equal(t, testLocationLink, got.Class.LocationLink)
			},
		},
		{
			name:  "Success booking creation - contact does not exist, with not full pass",
			token: testToken,
			data:  newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(data.pendingBooking, nil)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBooking.ClassID,
						data.pendingBooking.Email,
					).
					Return(models.Booking{}, errs.ErrNotFound)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), data.pendingBooking.Class.ID).
					Return(0, nil)

				contactsRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pendingBooking.Email,
						data.pendingBooking.FirstName,
						data.pendingBooking.LastName,
					).
					Return(data.contact, nil)

				passesRepo.EXPECT().
					ListByEmail(gomock.Any(), data.pendingBooking.Email, threeLastPasses).
					Return([]models.Pass{data.passes[0]}, nil)

				bookingsRepo.EXPECT().
					CountForPassID(gomock.Any(), data.passes[0].ID).
					Return(data.passes[0].TotalSlots-1, nil)

				bookingsRepo.EXPECT().
					Insert(gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						booking models.Booking,
					) (uuid.UUID, error) {
						require.True(t, booking.Pass.Exists())
						require.True(t, booking.PassID.Exists())

						assert.Equal(t, data.passes[0].ID, booking.PassID.Get())
						assert.Equal(t, data.passes[0].ID, booking.Pass.Get().ID)
						assert.Equal(t, data.pendingBooking.ClassID, booking.ClassID)
						assert.Equal(t, data.pendingBooking.Email, booking.Email)
						assert.Equal(t, data.pendingBooking.FirstName, booking.FirstName)
						assert.Equal(t, data.pendingBooking.LastName, booking.LastName)
						assert.Equal(t, data.pendingBooking.ConfirmationToken, booking.ConfirmationToken)

						return booking.ID, nil
					})

				bookingsRepo.EXPECT().
					ListByPassID(gomock.Any(), data.passes[0].ID).
					Return([]models.Booking{data.booking}, nil)

				locationLinkProvider.EXPECT().
					GetLink(data.booking.Class.Location).
					Return(testLocationLink, nil)

				notifier.EXPECT().
					NotifyBookingConfirmation(
						gomock.Any(),
						gomock.Any(),
					).
					DoAndReturn(func(
						params models.NotifierParams,
						cancellationLink string,
					) error {
						assert.Equal(t, data.pendingBooking.Email, params.RecipientEmail)
						assert.Equal(t, data.pendingBooking.FirstName, params.RecipientFirstName)
						assert.Equal(t, data.pendingBooking.LastName, params.RecipientLastName)
						assert.Equal(t, data.pendingBooking.Class.ClassLevel, params.ClassLevel)
						assert.Equal(t, data.pendingBooking.Class.StartTime, params.StartTime)
						assert.Equal(t, data.pendingBooking.Class.ClassName, params.ClassName)
						assertPassSlots(t, params.PassSlots, 0, 7, 1)

						assert.Contains(t, cancellationLink, "/bookings/")
						assert.Contains(t, cancellationLink, "/cancel_form?token="+testToken)

						return nil
					})
			},
			assert: func(
				t *testing.T,
				data testData,
				got BookingCreation,
			) {
				t.Helper()
				assert.Equal(t, data.class.ID, got.Class.ID)
				assert.Equal(t, data.class.StartTime, got.Class.StartTime)
				assert.Equal(t, data.class.ClassLevel, got.Class.ClassLevel)
				assert.Equal(t, data.class.ClassName, got.Class.ClassName)
				assert.Equal(t, data.class.MaxCapacity, got.Class.MaxCapacity)
				assert.Equal(t, data.class.Location, got.Class.Location)
				assert.Equal(t, testLocationLink, got.Class.LocationLink)
			},
		},
		{
			name:  "Success booking creation - contact does not exist, with full pass",
			token: testToken,
			data:  newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(data.pendingBooking, nil)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBooking.ClassID,
						data.pendingBooking.Email,
					).
					Return(models.Booking{}, errs.ErrNotFound)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), data.pendingBooking.Class.ID).
					Return(0, nil)

				contactsRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pendingBooking.Email,
						data.pendingBooking.FirstName,
						data.pendingBooking.LastName,
					).
					Return(data.contact, nil)

				passesRepo.EXPECT().
					ListByEmail(gomock.Any(), data.pendingBooking.Email, threeLastPasses).
					Return([]models.Pass{data.passes[0]}, nil)

				bookingsRepo.EXPECT().
					CountForPassID(gomock.Any(), data.passes[0].ID).
					Return(data.passes[0].TotalSlots, nil)

				bookingsRepo.EXPECT().
					Insert(gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						booking models.Booking,
					) (uuid.UUID, error) {
						assert.Equal(t, data.pendingBooking.ClassID, booking.ClassID)
						assert.Equal(t, data.pendingBooking.Email, booking.Email)
						assert.Equal(t, data.pendingBooking.FirstName, booking.FirstName)
						assert.Equal(t, data.pendingBooking.LastName, booking.LastName)
						assert.Equal(t, data.pendingBooking.ConfirmationToken, booking.ConfirmationToken)

						return booking.ID, nil
					})

				locationLinkProvider.EXPECT().
					GetLink(data.booking.Class.Location).
					Return(testLocationLink, nil)

				notifier.EXPECT().
					NotifyBookingConfirmation(
						gomock.Any(),
						gomock.Any(),
					).
					DoAndReturn(func(
						params models.NotifierParams,
						cancellationLink string,
					) error {
						assert.Equal(t, data.pendingBooking.Email, params.RecipientEmail)
						assert.Equal(t, data.pendingBooking.FirstName, params.RecipientFirstName)
						assert.Equal(t, data.pendingBooking.LastName, params.RecipientLastName)
						assert.Equal(t, data.pendingBooking.Class.ClassName, params.ClassName)

						assert.Contains(t, cancellationLink, "/bookings/")
						assert.Contains(t, cancellationLink, "/cancel_form?token="+testToken)

						return nil
					})
			},
			assert: func(
				t *testing.T,
				data testData,
				got BookingCreation,
			) {
				t.Helper()

				assert.Equal(t, data.class.ID, got.Class.ID)
				assert.Equal(t, data.class.StartTime, got.Class.StartTime)
				assert.Equal(t, data.class.ClassName, got.Class.ClassName)
				assert.Equal(t, data.class.ClassLevel, got.Class.ClassLevel)
				assert.Equal(t, data.class.MaxCapacity, got.Class.MaxCapacity)
				assert.Equal(t, data.class.Location, got.Class.Location)
				assert.Equal(t, testLocationLink, got.Class.LocationLink)
			},
		},
		{
			name:  "Success booking creation - contact does not exist, two passes - second not full",
			token: testToken,
			data: func() testData {
				testData := newTestData()
				testData.passes = append(testData.passes, models.Pass{
					ID:         2,
					Email:      "john.example@com",
					TotalSlots: 3,
					CreatedAt:  time.Now().Add(-time.Hour * 24 * 30),
				})

				return testData
			},
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(data.pendingBooking, nil)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBooking.ClassID,
						data.pendingBooking.Email,
					).
					Return(models.Booking{}, errs.ErrNotFound)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), data.pendingBooking.Class.ID).
					Return(0, nil)

				contactsRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pendingBooking.Email,
						data.pendingBooking.FirstName,
						data.pendingBooking.LastName,
					).
					Return(data.contact, nil)

				passesRepo.EXPECT().
					ListByEmail(gomock.Any(), data.pendingBooking.Email, threeLastPasses).
					Return(data.passes, nil)

				bookingsRepo.EXPECT().
					CountForPassID(gomock.Any(), data.passes[0].ID).
					Return(data.passes[0].TotalSlots, nil)

				bookingsRepo.EXPECT().
					CountForPassID(gomock.Any(), data.passes[1].ID).
					Return(data.passes[1].TotalSlots-2, nil)

				bookingsRepo.EXPECT().
					Insert(gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						booking models.Booking,
					) (uuid.UUID, error) {
						require.True(t, booking.Pass.Exists())
						require.True(t, booking.PassID.Exists())

						assert.Equal(t, data.passes[1].ID, booking.PassID.Get())
						assert.Equal(t, data.passes[1].ID, booking.Pass.Get().ID)
						assert.Equal(t, data.pendingBooking.ClassID, booking.ClassID)
						assert.Equal(t, data.pendingBooking.Email, booking.Email)
						assert.Equal(t, data.pendingBooking.FirstName, booking.FirstName)
						assert.Equal(t, data.pendingBooking.LastName, booking.LastName)
						assert.Equal(t, data.pendingBooking.ConfirmationToken, booking.ConfirmationToken)

						return booking.ID, nil
					})

				bookingsRepo.EXPECT().
					ListByPassID(gomock.Any(), data.passes[1].ID).
					Return([]models.Booking{newBooking(newPendingBooking(newClass())), data.booking}, nil)

				locationLinkProvider.EXPECT().
					GetLink(data.booking.Class.Location).
					Return(testLocationLink, nil)

				notifier.EXPECT().
					NotifyBookingConfirmation(
						gomock.Any(),
						gomock.Any(),
					).
					DoAndReturn(func(
						params models.NotifierParams,
						cancellationLink string,
					) error {
						assert.Equal(t, data.pendingBooking.Email, params.RecipientEmail)
						assert.Equal(t, data.pendingBooking.FirstName, params.RecipientFirstName)
						assert.Equal(t, data.pendingBooking.LastName, params.RecipientLastName)
						assert.Equal(t, data.pendingBooking.Class.ClassName, params.ClassName)
						assert.Equal(t, data.pendingBooking.Class.ClassLevel, params.ClassLevel)
						assert.Equal(t, data.pendingBooking.Class.StartTime, params.StartTime)
						assert.Equal(t, data.pendingBooking.Class.Location, params.Location)

						assertPassSlots(t, params.PassSlots, 0, 1, 2)

						assert.Contains(t, cancellationLink, "/bookings/")
						assert.Contains(t, cancellationLink, "/cancel_form?token="+testToken)

						return nil
					})
			},
			assert: func(
				t *testing.T,
				data testData,
				got BookingCreation,
			) {
				t.Helper()

				assert.Equal(t, data.class.ID, got.Class.ID)
				assert.Equal(t, data.class.StartTime, got.Class.StartTime)
				assert.Equal(t, data.class.ClassName, got.Class.ClassName)
				assert.Equal(t, data.class.ClassLevel, got.Class.ClassLevel)
				assert.Equal(t, data.class.MaxCapacity, got.Class.MaxCapacity)
				assert.Equal(t, data.class.Location, got.Class.Location)
				assert.Equal(t, testLocationLink, got.Class.LocationLink)
			},
		},
		{
			name:  "Success booking creation - contact does not exist, two passes - first not full pass",
			token: testToken,
			data: func() testData {
				testData := newTestData()
				testData.passes = append(testData.passes, models.Pass{
					ID:         2,
					Email:      "john.example@com",
					TotalSlots: 3,
					CreatedAt:  time.Now().Add(-time.Hour * 24 * 30),
				})

				return testData
			},
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				pendingBookingsRepo *mock.MockIPendingBookings,
				bookingsRepo *mock.MockIBookings,
				contactsRepo *mock.MockIContacts,
				passesRepo *mock.MockIPasses,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCreateBookingTransaction(
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
				)

				pendingBookingsRepo.EXPECT().
					GetByConfirmationToken(gomock.Any(), testToken).
					Return(data.pendingBooking, nil)

				bookingsRepo.EXPECT().
					GetByEmailAndClassID(
						gomock.Any(),
						data.pendingBooking.ClassID,
						data.pendingBooking.Email,
					).
					Return(models.Booking{}, errs.ErrNotFound)

				bookingsRepo.EXPECT().
					CountForClassID(gomock.Any(), data.pendingBooking.Class.ID).
					Return(0, nil)

				contactsRepo.EXPECT().
					Insert(
						gomock.Any(),
						data.pendingBooking.Email,
						data.pendingBooking.FirstName,
						data.pendingBooking.LastName,
					).
					Return(data.contact, nil)

				passesRepo.EXPECT().
					ListByEmail(gomock.Any(), data.pendingBooking.Email, threeLastPasses).
					Return(data.passes, nil)

				bookingsRepo.EXPECT().
					CountForPassID(gomock.Any(), data.passes[0].ID).
					Return(1, nil)

				bookingsRepo.EXPECT().
					Insert(gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						booking models.Booking,
					) (uuid.UUID, error) {
						require.True(t, booking.Pass.Exists())
						require.True(t, booking.PassID.Exists())

						assert.Equal(t, data.passes[0].ID, booking.PassID.Get())
						assert.Equal(t, data.passes[0].ID, booking.Pass.Get().ID)
						assert.Equal(t, data.pendingBooking.ClassID, booking.ClassID)
						assert.Equal(t, data.pendingBooking.Email, booking.Email)
						assert.Equal(t, data.pendingBooking.FirstName, booking.FirstName)
						assert.Equal(t, data.pendingBooking.LastName, booking.LastName)
						assert.Equal(t, data.pendingBooking.ConfirmationToken, booking.ConfirmationToken)

						return booking.ID, nil
					})

				bookingsRepo.EXPECT().
					ListByPassID(gomock.Any(), data.passes[0].ID).
					Return([]models.Booking{newBooking(newPendingBooking(newClass())), data.booking}, nil)

				locationLinkProvider.EXPECT().
					GetLink(data.booking.Class.Location).
					Return(testLocationLink, nil)

				notifier.EXPECT().
					NotifyBookingConfirmation(
						gomock.Any(),
						gomock.Any(),
					).
					DoAndReturn(func(
						params models.NotifierParams,
						cancellationLink string,
					) error {
						assert.Equal(t, data.pendingBooking.Email, params.RecipientEmail)
						assert.Equal(t, data.pendingBooking.FirstName, params.RecipientFirstName)
						assert.Equal(t, data.pendingBooking.LastName, params.RecipientLastName)
						assert.Equal(t, data.pendingBooking.Class.ClassName, params.ClassName)
						assert.Equal(t, data.pendingBooking.Class.ClassLevel, params.ClassLevel)
						assert.Equal(t, data.pendingBooking.Class.StartTime, params.StartTime)
						assert.Equal(t, data.pendingBooking.Class.Location, params.Location)

						assertPassSlots(t, params.PassSlots, 0, 6, 2)

						assert.Contains(t, cancellationLink, "/bookings/")
						assert.Contains(t, cancellationLink, "/cancel_form?token="+testToken)

						return nil
					})
			},
			assert: func(
				t *testing.T,
				data testData,
				got BookingCreation,
			) {
				t.Helper()

				assert.Equal(t, data.class.ID, got.Class.ID)
				assert.Equal(t, data.class.StartTime, got.Class.StartTime)
				assert.Equal(t, data.class.ClassName, got.Class.ClassName)
				assert.Equal(t, data.class.ClassLevel, got.Class.ClassLevel)
				assert.Equal(t, data.class.MaxCapacity, got.Class.MaxCapacity)
				assert.Equal(t, data.class.Location, got.Class.Location)
				assert.Equal(t, testLocationLink, got.Class.LocationLink)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			unitOfWork := mock.NewMockIUnitOfWork(ctrl)
			pendingBookingsRepo := mock.NewMockIPendingBookings(ctrl)
			bookingsRepo := mock.NewMockIBookings(ctrl)
			contactsRepo := mock.NewMockIContacts(ctrl)
			passesRepo := mock.NewMockIPasses(ctrl)
			locationLinkProvider := mock.NewMockILinkProvider(ctrl)
			notifier := mock.NewMockINotifier(ctrl)

			var data testData

			if tt.data != nil {
				data = tt.data()
			}

			if tt.mocks != nil {
				tt.mocks(
					data,
					unitOfWork,
					pendingBookingsRepo,
					bookingsRepo,
					contactsRepo,
					passesRepo,
					locationLinkProvider,
					notifier,
				)
			}

			service := NewService(
				unitOfWork,
				bookingsRepo,
				notifier,
				locationLinkProvider,
				testDomain,
			)

			got, err := service.CreateBooking(
				context.Background(),
				tt.token,
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

func mockCancelBookingTransaction(
	uow *mock.MockIUnitOfWork,
	bookingsRepo *mock.MockIBookings,
) {
	uow.EXPECT().
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

func TestService_CancelBooking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		token string
		data  func() testData
		mocks func(
			data testData,
			unitOfWork *mock.MockIUnitOfWork,
			bookingsRepo *mock.MockIBookings,
			locationLinkProvider *mock.MockILinkProvider,
			notifier *mock.MockINotifier,
		)
		wantError     bool
		errorContains string
	}{
		{
			name:  "Failure booking cancellation - booking not found error",
			token: testToken,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCancelBookingTransaction(
					unitOfWork,
					bookingsRepo,
				)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), gomock.Any()).
					Return(models.Booking{}, errs.ErrNotFound)
			},
			wantError:     true,
			errorContains: "booking with id",
		},
		{
			name:  "Failure booking cancellation - repository get booking error",
			token: testToken,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCancelBookingTransaction(
					unitOfWork,
					bookingsRepo,
				)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), gomock.Any()).
					Return(models.Booking{}, assert.AnError)
			},
			wantError:     true,
			errorContains: "could not get booking for id",
		},
		{
			name:  "Failure booking cancellation - invalid cancellation token error",
			token: "invalid-token",
			data: func() testData {
				data := newTestData()
				data.booking.ConfirmationToken = "valid-token"

				return data
			},
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCancelBookingTransaction(
					unitOfWork,
					bookingsRepo,
				)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(data.booking, nil)
			},
			wantError:     true,
			errorContains: "invalid token",
		},
		{
			name:  "Failure booking cancellation - class expired error",
			token: testToken,
			data: func() testData {
				data := newTestData()
				data.booking.Class.StartTime = time.Now().Add(-time.Hour)

				return data
			},

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCancelBookingTransaction(
					unitOfWork,
					bookingsRepo,
				)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(data.booking, nil)
			},

			wantError:     true,
			errorContains: "has expired",
		},
		{
			name:  "Failure booking cancellation - delete booking not found error",
			token: testToken,
			data:  newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCancelBookingTransaction(
					unitOfWork,
					bookingsRepo,
				)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(data.booking, nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), data.booking.ID).
					Return(errs.ErrNoRowsAffected)
			},

			wantError:     true,
			errorContains: "delete booking failure",
		},
		{
			name:  "Failure booking cancellation - delete booking repository error",
			token: testToken,
			data:  newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCancelBookingTransaction(
					unitOfWork,
					bookingsRepo,
				)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(data.booking, nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), data.booking.ID).
					Return(assert.AnError)
			},

			wantError:     true,
			errorContains: "could not delete booking",
		},
		{
			name:  "Failure booking cancellation - list bookings by pass error",
			token: testToken,
			data: func() testData {
				data := newTestData()
				data.booking.Pass = optional.Of(newPass())

				return data
			},

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCancelBookingTransaction(
					unitOfWork,
					bookingsRepo,
				)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(data.booking, nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), data.booking.ID).
					Return(nil)

				bookingsRepo.EXPECT().
					ListByPassID(gomock.Any(), data.booking.Pass.Get().ID).
					Return(nil, assert.AnError)
			},

			wantError:     true,
			errorContains: "could not build passSlots",
		},
		{
			name:  "Failure booking cancellation - location link provider error",
			token: testToken,
			data:  newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCancelBookingTransaction(
					unitOfWork,
					bookingsRepo,
				)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(data.booking, nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), data.booking.ID).
					Return(nil)

				locationLinkProvider.EXPECT().
					GetLink(data.booking.Class.Location).
					Return("", assert.AnError)
			},

			wantError:     true,
			errorContains: "could not resolve location link",
		},
		{
			name:  "Failure booking cancellation - notifier error",
			token: testToken,
			data:  newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCancelBookingTransaction(
					unitOfWork,
					bookingsRepo,
				)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(data.booking, nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), data.booking.ID).
					Return(nil)

				locationLinkProvider.EXPECT().
					GetLink(data.booking.Class.Location).
					Return(testLocationLink, nil)

				notifier.EXPECT().
					NotifyBookingCancellation(gomock.Any()).
					Return(assert.AnError)
			},

			wantError:     true,
			errorContains: "could not notify booking cancellation",
		},
		{
			name:  "Success booking cancellation - booking without pass",
			token: testToken,
			data:  newTestData,

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCancelBookingTransaction(
					unitOfWork,
					bookingsRepo,
				)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(data.booking, nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), data.booking.ID).
					Return(nil)

				locationLinkProvider.EXPECT().
					GetLink(data.booking.Class.Location).
					Return(testLocationLink, nil)

				notifier.EXPECT().
					NotifyBookingCancellation(gomock.Any()).
					DoAndReturn(func(params models.NotifierParams) error {
						assert.Equal(t, data.booking.Email, params.RecipientEmail)
						assert.Equal(t, data.booking.FirstName, params.RecipientFirstName)
						assert.Equal(t, data.booking.LastName, params.RecipientLastName)
						assert.Equal(t, data.booking.Class.ClassName, params.ClassName)
						assert.Equal(t, data.booking.Class.ClassLevel, params.ClassLevel)
						assert.Equal(t, data.booking.Class.StartTime, params.StartTime)
						assert.Equal(t, data.booking.Class.Location, params.Location)
						assert.Equal(t, testLocationLink, params.LocationLink)
						assert.Empty(t, params.PassSlots)

						return nil
					})
			},
		},
		{
			name:  "Success booking cancellation - booking with pass",
			token: testToken,
			data: func() testData {
				data := newTestData()
				pass := newPass()
				data.booking.Pass = optional.Of(pass)
				data.booking.PassID = optional.Of(pass.ID)

				return data
			},

			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockCancelBookingTransaction(
					unitOfWork,
					bookingsRepo,
				)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(data.booking, nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), data.booking.ID).
					Return(nil)

				bookingsRepo.EXPECT().
					ListByPassID(gomock.Any(), data.booking.Pass.Get().ID).
					Return([]models.Booking{}, nil)

				locationLinkProvider.EXPECT().
					GetLink(data.booking.Class.Location).
					Return(testLocationLink, nil)

				notifier.EXPECT().
					NotifyBookingCancellation(gomock.Any()).
					DoAndReturn(func(params models.NotifierParams) error {
						assert.Equal(t, data.booking.Email, params.RecipientEmail)
						assert.Equal(t, data.booking.FirstName, params.RecipientFirstName)
						assert.Equal(t, data.booking.LastName, params.RecipientLastName)
						assert.Equal(t, data.booking.Class.ClassName, params.ClassName)
						assert.Equal(t, data.booking.Class.ClassLevel, params.ClassLevel)
						assert.Equal(t, data.booking.Class.StartTime, params.StartTime)
						assert.Equal(t, data.booking.Class.Location, params.Location)
						assert.Equal(t, testLocationLink, params.LocationLink)
						assert.NotEmpty(t, params.PassSlots)
						assert.Len(t, params.PassSlots, data.booking.Pass.Get().TotalSlots)
						assert.Equal(t, models.BlankStatus, params.PassSlots[0].Status)

						return nil
					})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			unitOfWork := mock.NewMockIUnitOfWork(ctrl)
			bookingsRepo := mock.NewMockIBookings(ctrl)
			locationLinkProvider := mock.NewMockILinkProvider(ctrl)
			notifier := mock.NewMockINotifier(ctrl)

			var data testData
			if tt.data != nil {
				data = tt.data()
			}

			if tt.mocks != nil {
				tt.mocks(
					data,
					unitOfWork,
					bookingsRepo,
					locationLinkProvider,
					notifier,
				)
			}

			service := NewService(
				unitOfWork,
				bookingsRepo,
				notifier,
				locationLinkProvider,
				testDomain,
			)

			err := service.CancelBooking(
				context.Background(),
				data.booking.ID,
				tt.token,
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

func mockDeleteBookingTransaction(
	uow *mock.MockIUnitOfWork,
	bookingsRepo *mock.MockIBookings,
) {
	uow.EXPECT().
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

func TestService_DeleteBooking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		data  func() testData
		mocks func(
			data testData,
			unitOfWork *mock.MockIUnitOfWork,
			bookingsRepo *mock.MockIBookings,
			locationLinkProvider *mock.MockILinkProvider,
			notifier *mock.MockINotifier,
		)
		wantError     bool
		errorContains string
	}{
		{
			name: "Failure booking deletion - booking not found error",
			data: newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockDeleteBookingTransaction(unitOfWork, bookingsRepo)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(models.Booking{}, errs.ErrNotFound)
			},
			wantError:     true,
			errorContains: "could get booking for id",
		},
		{
			name: "Failure booking deletion - repository get booking error",
			data: newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockDeleteBookingTransaction(unitOfWork, bookingsRepo)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(models.Booking{}, assert.AnError)
			},
			wantError:     true,
			errorContains: "could get booking for id",
		},
		{
			name: "Failure booking deletion - delete booking not found error",
			data: newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockDeleteBookingTransaction(unitOfWork, bookingsRepo)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(data.booking, nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), data.booking.ID).
					Return(errs.ErrNoRowsAffected)
			},
			wantError:     true,
			errorContains: "could not delete booking",
		},
		{
			name: "Failure booking deletion - delete booking repository error",
			data: newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockDeleteBookingTransaction(unitOfWork, bookingsRepo)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(data.booking, nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), data.booking.ID).
					Return(assert.AnError)
			},
			wantError:     true,
			errorContains: "could not delete booking",
		},
		{
			name: "Failure booking deletion - list bookings by pass error",
			data: func() testData {
				data := newTestData()

				pass := newPass()
				data.booking.Pass = optional.Of(pass)

				return data
			},
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockDeleteBookingTransaction(unitOfWork, bookingsRepo)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(data.booking, nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), data.booking.ID).
					Return(nil)

				bookingsRepo.EXPECT().
					ListByPassID(gomock.Any(), data.booking.Pass.Get().ID).
					Return(nil, assert.AnError)
			},
			wantError:     true,
			errorContains: "could not build passSlots",
		},
		{
			name: "Failure booking deletion - location link provider error",
			data: newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockDeleteBookingTransaction(unitOfWork, bookingsRepo)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(data.booking, nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), data.booking.ID).
					Return(nil)

				locationLinkProvider.EXPECT().
					GetLink(data.booking.Class.Location).
					Return("", assert.AnError)
			},
			wantError:     true,
			errorContains: "could not resolve location link",
		},
		{
			name: "Failure booking deletion - notifier error",
			data: newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockDeleteBookingTransaction(unitOfWork, bookingsRepo)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(data.booking, nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), data.booking.ID).
					Return(nil)

				locationLinkProvider.EXPECT().
					GetLink(data.booking.Class.Location).
					Return(testLocationLink, nil)

				notifier.EXPECT().
					NotifyBookingCancellation(gomock.Any()).
					Return(assert.AnError)
			},
			wantError:     true,
			errorContains: "could not nofify booking cancellation",
		},
		{
			name: "Success booking deletion - booking without pass",
			data: newTestData,
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockDeleteBookingTransaction(unitOfWork, bookingsRepo)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(data.booking, nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), data.booking.ID).
					Return(nil)

				locationLinkProvider.EXPECT().
					GetLink(data.booking.Class.Location).
					Return(testLocationLink, nil)

				notifier.EXPECT().
					NotifyBookingCancellation(gomock.Any()).
					DoAndReturn(func(params models.NotifierParams) error {
						assert.Equal(t, data.booking.Email, params.RecipientEmail)
						assert.Equal(t, data.booking.Class.ClassName, params.ClassName)
						assert.Equal(t, testLocationLink, params.LocationLink)
						assert.Empty(t, params.PassSlots)

						return nil
					})
			},
		},
		{
			name: "Success booking deletion - booking with pass",
			data: func() testData {
				data := newTestData()

				pass := newPass()
				data.booking.Pass = optional.Of(pass)
				data.booking.PassID = optional.Of(pass.ID)

				return data
			},
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockDeleteBookingTransaction(
					unitOfWork,
					bookingsRepo,
				)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(data.booking, nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), data.booking.ID).
					Return(nil)

				bookingsRepo.EXPECT().
					ListByPassID(gomock.Any(), data.booking.Pass.Get().ID).
					Return([]models.Booking{}, nil)

				locationLinkProvider.EXPECT().
					GetLink(data.booking.Class.Location).
					Return(testLocationLink, nil)

				notifier.EXPECT().
					NotifyBookingCancellation(gomock.Any()).
					DoAndReturn(func(params models.NotifierParams) error {
						assert.Equal(t, data.booking.Email, params.RecipientEmail)
						assert.Equal(t, data.booking.FirstName, params.RecipientFirstName)
						assert.Equal(t, data.booking.LastName, params.RecipientLastName)

						assert.Equal(t, data.booking.Class.ClassName, params.ClassName)
						assert.Equal(t, data.booking.Class.ClassLevel, params.ClassLevel)
						assert.Equal(t, data.booking.Class.StartTime, params.StartTime)
						assert.Equal(t, data.booking.Class.Location, params.Location)

						assert.Equal(t, testLocationLink, params.LocationLink)

						assert.Len(
							t,
							params.PassSlots,
							data.booking.Pass.Get().TotalSlots,
						)

						assertPassSlots(
							t,
							params.PassSlots,
							0,
							8,
							0,
						)

						return nil
					})
			},
		},
		{
			name: "Success booking deletion - booking with pass and existing bookings",
			data: func() testData {
				data := newTestData()

				pass := newPass()
				data.booking.Pass = optional.Of(pass)
				data.booking.PassID = optional.Of(pass.ID)

				return data
			},
			mocks: func(
				data testData,
				unitOfWork *mock.MockIUnitOfWork,
				bookingsRepo *mock.MockIBookings,
				locationLinkProvider *mock.MockILinkProvider,
				notifier *mock.MockINotifier,
			) {
				mockDeleteBookingTransaction(
					unitOfWork,
					bookingsRepo,
				)

				bookingsRepo.EXPECT().
					GetByID(gomock.Any(), data.booking.ID).
					Return(data.booking, nil)

				bookingsRepo.EXPECT().
					Delete(gomock.Any(), data.booking.ID).
					Return(nil)

				existingBooking1 := newBooking(newPendingBooking(newClass()))
				existingBooking1.Class.StartTime = time.Now().Add(-time.Hour * 24 * 7)

				existingBooking2 := newBooking(newPendingBooking(newClass()))
				existingBooking2.Class.StartTime = time.Now().Add(time.Hour * 24 * 7)

				bookingsRepo.EXPECT().
					ListByPassID(
						gomock.Any(),
						data.booking.Pass.Get().ID,
					).
					Return(
						[]models.Booking{
							existingBooking1,
							existingBooking2,
						},
						nil,
					)

				locationLinkProvider.EXPECT().
					GetLink(data.booking.Class.Location).
					Return(testLocationLink, nil)

				notifier.EXPECT().
					NotifyBookingCancellation(gomock.Any()).
					DoAndReturn(func(params models.NotifierParams) error {
						assert.Equal(
							t,
							data.booking.Email,
							params.RecipientEmail,
						)

						assert.Equal(
							t,
							data.booking.Class.ClassName,
							params.ClassName,
						)

						assert.Equal(
							t,
							testLocationLink,
							params.LocationLink,
						)

						assert.Len(
							t,
							params.PassSlots,
							data.booking.Pass.Get().TotalSlots,
						)

						assertPassSlots(
							t,
							params.PassSlots,
							1,
							6,
							1,
						)

						return nil
					})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			unitOfWork := mock.NewMockIUnitOfWork(ctrl)
			bookingsRepo := mock.NewMockIBookings(ctrl)
			locationLinkProvider := mock.NewMockILinkProvider(ctrl)
			notifier := mock.NewMockINotifier(ctrl)

			data := tt.data()

			tt.mocks(
				data,
				unitOfWork,
				bookingsRepo,
				locationLinkProvider,
				notifier,
			)

			service := NewService(
				unitOfWork,
				bookingsRepo,
				notifier,
				locationLinkProvider,
				testDomain,
			)

			err := service.DeleteBooking(
				context.Background(),
				data.booking.ID,
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

func assertPassSlots(
	t *testing.T,
	slots []models.PassSlot,
	expectedUsed int,
	expectedBlank int,
	expectedFuture int,
) {
	t.Helper()

	var used int

	var blank int

	var future int

	for _, slot := range slots {
		switch slot.Status {
		case models.PastStatus:
			used++
		case models.BlankStatus:
			blank++
		case models.FutureStatus:
			future++
		}
	}

	assert.Equal(t, expectedUsed, used)
	assert.Equal(t, expectedBlank, blank)
	assert.Equal(t, expectedFuture, future)
}
