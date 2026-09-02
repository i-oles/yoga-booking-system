package pendingbooking

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	domainErrs "main/internal/domain/errs/view"
	"main/internal/domain/models"
	viewErrHandler "main/internal/interfaces/http/html/errs/handler"
	mockpendingbookings "main/mock/pendingbookings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	testClassID := uuid.New()

	tests := []struct {
		name  string
		form  url.Values
		mocks func(
			service *mockpendingbookings.MockIService,
		)
		assert func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "success - pending booking creation",
			form: url.Values{
				"email":      {"anna@example.com"},
				"class_id":   {testClassID.String()},
				"first_name": {"Anna"},
				"last_name":  {"Kowalska"},
			},
			mocks: func(
				service *mockpendingbookings.MockIService,
			) {
				service.EXPECT().
					CreatePendingBooking(gomock.Any(), models.PendingBookingParams{
						ClassID:   testClassID,
						FirstName: "Anna",
						LastName:  "Kowalska",
						Email:     "anna@example.com",
					}).
					Return(nil)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "kliknij na link")
			},
		},
		{
			name: "failure - invalid email",
			form: url.Values{
				"email":      {"not-an-email"},
				"class_id":   {testClassID.String()},
				"first_name": {"Anna"},
				"last_name":  {"Kowalska"},
			},
			mocks: func(
				service *mockpendingbookings.MockIService,
			) {
				// No service call expected.
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "failure - missing class id",
			form: url.Values{
				"email":      {"anna@example.com"},
				"first_name": {"Anna"},
				"last_name":  {"Kowalska"},
			},
			mocks: func(
				service *mockpendingbookings.MockIService,
			) {
				// No service call expected.
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "failure - first name too short",
			form: url.Values{
				"email":      {"anna@example.com"},
				"class_id":   {testClassID.String()},
				"first_name": {"An"},
				"last_name":  {"Kowalska"},
			},
			mocks: func(
				service *mockpendingbookings.MockIService,
			) {
				// No service call expected.
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "failure - pending booking creation error",
			form: url.Values{
				"email":      {"anna@example.com"},
				"class_id":   {testClassID.String()},
				"first_name": {"Anna"},
				"last_name":  {"Kowalska"},
			},
			mocks: func(
				service *mockpendingbookings.MockIService,
			) {
				service.EXPECT().
					CreatePendingBooking(gomock.Any(), models.PendingBookingParams{
						ClassID:   testClassID,
						FirstName: "Anna",
						LastName:  "Kowalska",
						Email:     "anna@example.com",
					}).
					Return(assert.AnError)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "failure - booking already exists",
			form: url.Values{
				"email":      {"anna@example.com"},
				"class_id":   {testClassID.String()},
				"first_name": {"Anna"},
				"last_name":  {"Kowalska"},
			},
			mocks: func(
				service *mockpendingbookings.MockIService,
			) {
				businessErr := domainErrs.ErrBookingAlreadyExists(
					testClassID, "anna@example.com", assert.AnError,
				)

				service.EXPECT().
					CreatePendingBooking(gomock.Any(), models.PendingBookingParams{
						ClassID:   testClassID,
						FirstName: "Anna",
						LastName:  "Kowalska",
						Email:     "anna@example.com",
					}).
					Return(fmt.Errorf("create pending booking transaction failed: %w", businessErr))
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "failure - class expired",
			form: url.Values{
				"email":      {"anna@example.com"},
				"class_id":   {testClassID.String()},
				"first_name": {"Anna"},
				"last_name":  {"Kowalska"},
			},
			mocks: func(
				service *mockpendingbookings.MockIService,
			) {
				businessErr := domainErrs.ErrClassExpired(testClassID, assert.AnError)

				service.EXPECT().
					CreatePendingBooking(gomock.Any(), models.PendingBookingParams{
						ClassID:   testClassID,
						FirstName: "Anna",
						LastName:  "Kowalska",
						Email:     "anna@example.com",
					}).
					Return(fmt.Errorf("create pending booking transaction failed: %w", businessErr))
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "failure - too many pending bookings",
			form: url.Values{
				"email":      {"anna@example.com"},
				"class_id":   {testClassID.String()},
				"first_name": {"Anna"},
				"last_name":  {"Kowalska"},
			},
			mocks: func(
				service *mockpendingbookings.MockIService,
			) {
				businessErr := domainErrs.ErrTooManyPendingBookings(
					testClassID, "anna@example.com", assert.AnError,
				)

				service.EXPECT().
					CreatePendingBooking(gomock.Any(), models.PendingBookingParams{
						ClassID:   testClassID,
						FirstName: "Anna",
						LastName:  "Kowalska",
						Email:     "anna@example.com",
					}).
					Return(fmt.Errorf("create pending booking transaction failed: %w", businessErr))
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "failure - class fully booked",
			form: url.Values{
				"email":      {"anna@example.com"},
				"class_id":   {testClassID.String()},
				"first_name": {"Anna"},
				"last_name":  {"Kowalska"},
			},
			mocks: func(
				service *mockpendingbookings.MockIService,
			) {
				businessErr := domainErrs.ErrClassFullyBooked(testClassID, assert.AnError)

				service.EXPECT().
					CreatePendingBooking(gomock.Any(), models.PendingBookingParams{
						ClassID:   testClassID,
						FirstName: "Anna",
						LastName:  "Kowalska",
						Email:     "anna@example.com",
					}).
					Return(fmt.Errorf("create pending booking transaction failed: %w", businessErr))
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			service := mockpendingbookings.NewMockIService(ctrl)

			tt.mocks(service)

			errorHandler := viewErrHandler.NewErrorHandler()

			handler := NewHandler(service, errorHandler)

			router := gin.New()

			router.LoadHTMLFiles(
				"../../../../../../web/templates/pending_booking.tmpl",
				"../../../../../../web/templates/pending_booking_form.tmpl",
				"../../../../../../web/templates/err.tmpl",
			)

			router.POST("/pending_bookings", handler.Handle)

			request := httptest.NewRequest(
				http.MethodPost,
				"/pending_bookings",
				strings.NewReader(tt.form.Encode()),
			)
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			tt.assert(t, recorder)
		})
	}
}
