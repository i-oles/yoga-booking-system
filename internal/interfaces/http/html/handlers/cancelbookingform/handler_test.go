package cancelbookingform

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"main/internal/application/bookings"
	"main/internal/application/classes"
	domainErrs "main/internal/domain/errs/view"
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

	testBookingID := uuid.New()

	testClass := classes.BookingCancellationClass{
		ID:         uuid.New(),
		StartTime:  time.Date(2026, 8, 10, 18, 30, 0, 0, time.UTC),
		ClassLevel: "beginner",
		ClassName:  "Morning Yoga",
		Location:   "Warsaw",
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
			name: "success - booking cancellation form",
			url:  "/bookings/" + testBookingID.String() + "/cancel_form?token=" + testToken,
			mocks: func(
				service *mockbookings.MockIService,
			) {
				service.EXPECT().
					GetBookingCancellationForm(gomock.Any(), testBookingID, testToken).
					Return(bookings.BookingCancellationForm{
						Class:             testClass,
						BookingID:         testBookingID,
						ConfirmationToken: testToken,
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
			url:  "/bookings/" + testBookingID.String() + "/cancel_form?token=invalid",
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
			url:  "/bookings/" + testBookingID.String() + "/cancel_form",
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
			name: "failure - invalid booking id",
			url:  "/bookings/not-a-uuid/cancel_form?token=" + testToken,
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
			name: "failure - booking cancellation form error",
			url:  "/bookings/" + testBookingID.String() + "/cancel_form?token=" + testToken,
			mocks: func(
				service *mockbookings.MockIService,
			) {
				service.EXPECT().
					GetBookingCancellationForm(gomock.Any(), testBookingID, testToken).
					Return(bookings.BookingCancellationForm{}, assert.AnError)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "failure - booking not found",
			url:  "/bookings/" + testBookingID.String() + "/cancel_form?token=" + testToken,
			mocks: func(
				service *mockbookings.MockIService,
			) {
				businessErr := domainErrs.ErrBookingNotFound(assert.AnError)

				service.EXPECT().
					GetBookingCancellationForm(gomock.Any(), testBookingID, testToken).
					Return(bookings.BookingCancellationForm{},
						fmt.Errorf("could not get booking: %w", businessErr))
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "failure - invalid cancellation link",
			url:  "/bookings/" + testBookingID.String() + "/cancel_form?token=" + testToken,
			mocks: func(
				service *mockbookings.MockIService,
			) {
				businessErr := domainErrs.ErrInvalidCancellationLink(assert.AnError)

				service.EXPECT().
					GetBookingCancellationForm(gomock.Any(), testBookingID, testToken).
					Return(bookings.BookingCancellationForm{},
						fmt.Errorf("could not get booking: %w", businessErr))
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			service := mockbookings.NewMockIService(ctrl)

			tt.mocks(service)

			errorHandler := viewErrHandler.NewErrorHandler()

			handler := NewHandler(service, errorHandler)

			router := gin.New()

			router.LoadHTMLFiles(
				"../../../../../../web/templates/cancel_booking_form.tmpl",
				"../../../../../../web/templates/err.tmpl",
			)

			router.GET("/bookings/:id/cancel_form", handler.Handle)

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
