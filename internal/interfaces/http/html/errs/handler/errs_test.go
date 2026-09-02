package viewErrHandler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domainErrs "main/internal/domain/errs/view"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestErrorHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		hxRequest bool
		assert    func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "business error - booking not found maps to 404",
			err: &domainErrs.BusinessError{
				Code:    domainErrs.BookingNotFoundCode,
				Message: "booking not found message",
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusNotFound, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "booking not found message")
			},
		},
		{
			name: "business error - class expired maps to 409",
			err: &domainErrs.BusinessError{
				Code:    domainErrs.ClassExpiredCode,
				ClassID: uuidPtr(uuid.New()),
				Message: "class expired message",
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusConflict, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "class expired message")
			},
		},
		{
			name: "business error - class empty maps to 404",
			err: &domainErrs.BusinessError{
				Code:    domainErrs.ClassEmptyCode,
				ClassID: uuidPtr(uuid.New()),
				Message: "class empty message",
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusNotFound, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "class empty message")
			},
		},
		{
			name: "business error - booking already exists maps to 409",
			err: &domainErrs.BusinessError{
				Code:    domainErrs.BookingAlreadyExistsCode,
				ClassID: uuidPtr(uuid.New()),
				Message: "booking already exists message",
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusConflict, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "booking already exists message")
			},
		},
		{
			name: "business error - too many pending bookings maps to 429",
			err: &domainErrs.BusinessError{
				Code:    domainErrs.TooManyPendingBookingsCode,
				ClassID: uuidPtr(uuid.New()),
				Message: "too many pending bookings message",
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusTooManyRequests, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "too many pending bookings message")
			},
		},
		{
			name: "business error - class fully booked maps to 409",
			err: &domainErrs.BusinessError{
				Code:    domainErrs.ClassFullyBookedCode,
				ClassID: uuidPtr(uuid.New()),
				Message: "class fully booked message",
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusConflict, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "class fully booked message")
			},
		},
		{
			name: "business error - too late to book maps to 409",
			err: &domainErrs.BusinessError{
				Code:    domainErrs.TooLateToBook,
				ClassID: uuidPtr(uuid.New()),
				Message: "too late to book message",
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusConflict, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "too late to book message")
			},
		},
		{
			name: "business error - pending booking not found maps to 404",
			err: &domainErrs.BusinessError{
				Code:    domainErrs.PendingBookingNotFoundCode,
				Message: "pending booking not found message",
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusNotFound, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "pending booking not found message")
			},
		},
		{
			name: "business error - invalid cancellation link maps to 404",
			err: &domainErrs.BusinessError{
				Code:    domainErrs.InvalidCancellationLinkCode,
				Message: "invalid cancellation link message",
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusNotFound, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "invalid cancellation link message")
			},
		},
		{
			name: "business error - someone booked class faster maps to 409",
			err: fmt.Errorf("wrapped: %w", &domainErrs.BusinessError{
				Code:    domainErrs.SomeoneBookedClassFasterCode,
				Message: "someone booked class faster message",
			}),
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusConflict, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "someone booked class faster message")
			},
		},
		{
			name: "business error - unknown code falls back to 500 with a single write",
			err: &domainErrs.BusinessError{
				Code:    9999,
				Message: "unknown code message",
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assert.Equal(t, 1, strings.Count(recorder.Body.String(), "upss błąd!"),
					"response body must be written exactly once, not duplicated")
			},
		},
		{
			name: "non-business error renders err.tmpl",
			err:  assert.AnError,
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "upss błąd!")
				assert.Empty(t, recorder.Header().Get("Hx-Redirect"))
			},
		},
		{
			name:      "non-business error with HX-Request redirects instead of rendering",
			err:       assert.AnError,
			hxRequest: true,
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assert.Equal(t, "/error", recorder.Header().Get("Hx-Redirect"))
				assert.Empty(t, recorder.Body.String())
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()

			ctx, engine := gin.CreateTestContext(recorder)

			engine.LoadHTMLFiles("../../../../../../web/templates/err.tmpl")

			ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)

			if tt.hxRequest {
				ctx.Request.Header.Set("Hx-Request", "true")
			}

			handler := NewErrorHandler()

			handler.Handle(ctx, "err.tmpl", tt.err)
			ctx.Writer.WriteHeaderNow()

			tt.assert(t, recorder)
		})
	}
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	return &id
}
