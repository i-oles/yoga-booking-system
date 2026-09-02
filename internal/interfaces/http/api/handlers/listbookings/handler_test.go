package listbookings

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"main/internal/domain/models"
	apiErrHandler "main/internal/interfaces/http/api/errs/handler"
	mock "main/mock"
	"main/pkg/optional"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	testBooking := models.Booking{
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
		PassID:    optional.Empty[int](),
		Pass:      optional.Empty[models.Pass](),
		FirstName: "Anna",
		LastName:  "Kowalska",
		Email:     "anna@example.com",
		CreatedAt: time.Now(),
	}

	tests := []struct {
		name  string
		mocks func(
			bookingsRepo *mock.MockIBookings,
		)
		assert func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "success - list bookings",
			mocks: func(
				bookingsRepo *mock.MockIBookings,
			) {
				bookingsRepo.EXPECT().
					List(gomock.Any()).
					Return([]models.Booking{testBooking}, nil)
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
				bookingsRepo *mock.MockIBookings,
			) {
				bookingsRepo.EXPECT().
					List(gomock.Any()).
					Return([]models.Booking{}, nil)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.JSONEq(t, "[]", recorder.Body.String())
			},
		},
		{
			name: "failure - list bookings error",
			mocks: func(
				bookingsRepo *mock.MockIBookings,
			) {
				bookingsRepo.EXPECT().
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

			bookingsRepo := mock.NewMockIBookings(ctrl)

			tt.mocks(bookingsRepo)

			errorHandler := apiErrHandler.NewErrorHandler()

			handler := NewHandler(bookingsRepo, errorHandler)

			router := gin.New()

			router.GET("/api/v1/bookings", handler.Handle)

			request := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/bookings",
				nil,
			)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			tt.assert(t, recorder)
		})
	}
}
