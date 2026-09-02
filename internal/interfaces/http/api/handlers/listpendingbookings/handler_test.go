package listpendingbookings

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"main/internal/domain/models"
	apiErrHandler "main/internal/interfaces/http/api/errs/handler"
	mock "main/mock"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	testPendingBooking := models.PendingBooking{
		ID:      uuid.New(),
		ClassID: uuid.New(),
		Class: models.Class{
			ID:          uuid.New(),
			StartTime:   time.Date(2026, 8, 10, 18, 30, 0, 0, time.UTC),
			ClassLevel:  "beginner",
			ClassName:   "Morning Yoga",
			MaxCapacity: 10,
			Location:    "Warsaw",
		},
		Email:             "anna@example.com",
		FirstName:         "Anna",
		LastName:          "Kowalska",
		ConfirmationToken: "12345678901234567890123456789012333333333334",
		CreatedAt:         time.Now(),
	}

	tests := []struct {
		name  string
		mocks func(
			pendingBookingsRepo *mock.MockIPendingBookings,
		)
		assert func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "success - list pending bookings",
			mocks: func(
				pendingBookingsRepo *mock.MockIPendingBookings,
			) {
				pendingBookingsRepo.EXPECT().
					List(gomock.Any()).
					Return([]models.PendingBooking{testPendingBooking}, nil)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "anna@example.com")
			},
		},
		{
			name: "success - empty list",
			mocks: func(
				pendingBookingsRepo *mock.MockIPendingBookings,
			) {
				pendingBookingsRepo.EXPECT().
					List(gomock.Any()).
					Return([]models.PendingBooking{}, nil)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.JSONEq(t, "[]", recorder.Body.String())
			},
		},
		{
			name: "failure - list pending bookings error",
			mocks: func(
				pendingBookingsRepo *mock.MockIPendingBookings,
			) {
				pendingBookingsRepo.EXPECT().
					List(gomock.Any()).
					Return(nil, assert.AnError)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			pendingBookingsRepo := mock.NewMockIPendingBookings(ctrl)

			tt.mocks(pendingBookingsRepo)

			errorHandler := apiErrHandler.NewErrorHandler()

			handler := NewHandler(pendingBookingsRepo, errorHandler)

			router := gin.New()

			router.GET("/api/v1/pending_bookings", handler.Handle)

			request := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/pending_bookings",
				nil,
			)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			tt.assert(t, recorder)
		})
	}
}
