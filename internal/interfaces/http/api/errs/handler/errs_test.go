package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	domainErrs "main/internal/domain/errs/api"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestErrorHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		assert func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "api error - bad request maps to 400",
			err: &domainErrs.APIError{
				Code: domainErrs.BadRequestCode,
				Err:  errors.New("bad request message"),
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "bad request message")
			},
		},
		{
			name: "api error - conflict maps to 409",
			err: &domainErrs.APIError{
				Code: domainErrs.ConflictCode,
				Err:  errors.New("conflict message"),
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusConflict, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "conflict message")
			},
		},
		{
			name: "api error - not found maps to 404",
			err: &domainErrs.APIError{
				Code: domainErrs.NotFoundCode,
				Err:  errors.New("not found message"),
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusNotFound, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "not found message")
			},
		},
		{
			name: "api error - unknown code falls back to 500",
			err: &domainErrs.APIError{
				Code: 9999,
				Err:  errors.New("unknown code message"),
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "unknown code message")
			},
		},
		{
			name: "non-api error falls back to 500",
			err:  assert.AnError,
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assert.Contains(t, recorder.Body.String(), assert.AnError.Error())
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()

			ctx, _ := gin.CreateTestContext(recorder)

			ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)

			handler := NewErrorHandler()

			handler.Handle(ctx, tt.err)

			tt.assert(t, recorder)
		})
	}
}
