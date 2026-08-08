package createbooking

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"main/internal/application/bookings"
	"main/internal/application/classes"
	viewErrHandler "main/internal/interfaces/http/html/errs/handler"
	mockbookings "main/mock/bookings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const testToken = "12345678901234567890123456789012333333333334"

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	testClass := classes.ClassPresentation{
		ID:              uuid.New(),
		StartTime:       time.Date(2026, 8, 10, 18, 30, 0, 0, time.UTC),
		ClassLevel:      "beginner",
		ClassName:       "Morning Yoga",
		CurrentCapacity: 5,
		MaxCapacity:     10,
		Location:        "Warsaw",
		LocationLink:    "https://maps.google.com/test",
	}

	tests := []struct {
		name  string
		url   string
		mocks func(
			service *mockbookings.MockIService,
		)
		assert func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "success - booking creation",
			url:  "/bookings?token=" + testToken,
			mocks: func(
				service *mockbookings.MockIService,
			) {
				service.EXPECT().
					CreateBooking(gomock.Any(), testToken).
					Return(bookings.BookingCreation{
						Class: testClass,
					}, nil)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Contains(t, recorder.Body.String(), testClass.ClassName)
			},
		},
		{
			name: "failure - invalid token",
			url:  "/bookings?token=invalid",
			mocks: func(
				service *mockbookings.MockIService,
			) {
				// No service call expected.
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "failure - missing token",
			url:  "/bookings",
			mocks: func(
				service *mockbookings.MockIService,
			) {
				// No service call expected.
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "failure - booking creation error",
			url:  "/bookings?token=" + testToken,
			mocks: func(
				service *mockbookings.MockIService,
			) {
				service.EXPECT().
					CreateBooking(gomock.Any(), testToken).
					Return(bookings.BookingCreation{}, assert.AnError)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			service := mockbookings.NewMockIService(ctrl)

			tt.mocks(service)

			errorHandler := viewErrHandler.NewErrorHandler()

			handler := NewHandler(service, errorHandler)

			gin.SetMode(gin.TestMode)

			router := gin.New()

			router.LoadHTMLFiles(
				"../../../../../../web/templates/confirmation_create_booking.tmpl",
				"../../../../../../web/templates/err.tmpl",
			)

			router.GET("/bookings", handler.Handle)

			request := httptest.NewRequest(
				http.MethodGet,
				tt.url,
				nil,
			)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			tt.assert(t, recorder)
		})
	}
}
