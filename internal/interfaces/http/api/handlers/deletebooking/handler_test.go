package deletebooking

import (
	"net/http"
	"net/http/httptest"
	"testing"

	domainErrs "main/internal/domain/errs/api"
	apiErrHandler "main/internal/interfaces/http/api/errs/handler"
	mockbookings "main/mock/bookings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	testBookingID := uuid.New()

	tests := []struct {
		name  string
		url   string
		mocks func(
			service *mockbookings.MockIService,
		)
		assert func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "success - booking deletion",
			url:  "/api/v1/bookings/" + testBookingID.String(),
			mocks: func(
				service *mockbookings.MockIService,
			) {
				service.EXPECT().
					DeleteBooking(gomock.Any(), testBookingID).
					Return(nil)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Contains(t, recorder.Body.String(), testBookingID.String())
			},
		},
		{
			name: "failure - invalid booking id",
			url:  "/api/v1/bookings/not-a-uuid",
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
			name: "failure - booking deletion error",
			url:  "/api/v1/bookings/" + testBookingID.String(),
			mocks: func(
				service *mockbookings.MockIService,
			) {
				service.EXPECT().
					DeleteBooking(gomock.Any(), testBookingID).
					Return(assert.AnError)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "failure - booking not found",
			url:  "/api/v1/bookings/" + testBookingID.String(),
			mocks: func(
				service *mockbookings.MockIService,
			) {
				service.EXPECT().
					DeleteBooking(gomock.Any(), testBookingID).
					Return(domainErrs.ErrNotFound(assert.AnError))
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

			errorHandler := apiErrHandler.NewErrorHandler()

			handler := NewHandler(service, errorHandler)

			router := gin.New()

			router.DELETE("/api/v1/bookings/:booking_id", handler.Handle)

			request := httptest.NewRequest(
				http.MethodDelete,
				tt.url,
				nil,
			)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			tt.assert(t, recorder)
		})
	}
}
