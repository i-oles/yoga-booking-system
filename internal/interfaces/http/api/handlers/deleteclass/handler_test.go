package deleteclass

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainErrs "main/internal/domain/errs/api"
	apiErrHandler "main/internal/interfaces/http/api/errs/handler"
	mockclasses "main/mock/classes"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	testClassID := uuid.New()

	tests := []struct {
		name  string
		url   string
		body  any
		mocks func(
			service *mockclasses.MockIService,
		)
		assert func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "success - class deletion",
			url:  "/api/v1/classes/" + testClassID.String(),
			body: map[string]any{
				"message": "Cancelled",
			},
			mocks: func(
				service *mockclasses.MockIService,
			) {
				service.EXPECT().
					DeleteClass(gomock.Any(), testClassID, gomock.Any()).
					Return(nil)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Contains(t, recorder.Body.String(), testClassID.String())
			},
		},
		{
			name: "failure - empty request body",
			url:  "/api/v1/classes/" + testClassID.String(),
			body: nil,
			mocks: func(
				service *mockclasses.MockIService,
			) {
				// No service call expected.
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "failure - malformed request body",
			url:  "/api/v1/classes/" + testClassID.String(),
			body: "{invalid",
			mocks: func(
				service *mockclasses.MockIService,
			) {
				// No service call expected.
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "failure - invalid class id",
			url:  "/api/v1/classes/not-a-uuid",
			body: map[string]any{},
			mocks: func(
				service *mockclasses.MockIService,
			) {
				// No service call expected.
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "failure - class deletion error",
			url:  "/api/v1/classes/" + testClassID.String(),
			body: map[string]any{
				"message": "Cancelled",
			},
			mocks: func(
				service *mockclasses.MockIService,
			) {
				service.EXPECT().
					DeleteClass(gomock.Any(), testClassID, gomock.Any()).
					Return(assert.AnError)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "failure - message required when class has bookings",
			url:  "/api/v1/classes/" + testClassID.String(),
			body: map[string]any{},
			mocks: func(
				service *mockclasses.MockIService,
			) {
				service.EXPECT().
					DeleteClass(gomock.Any(), testClassID, gomock.Any()).
					Return(domainErrs.ErrValidation(assert.AnError))
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "failure - class not found",
			url:  "/api/v1/classes/" + testClassID.String(),
			body: map[string]any{
				"message": "Cancelled",
			},
			mocks: func(
				service *mockclasses.MockIService,
			) {
				service.EXPECT().
					DeleteClass(gomock.Any(), testClassID, gomock.Any()).
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

			service := mockclasses.NewMockIService(ctrl)

			tt.mocks(service)

			errorHandler := apiErrHandler.NewErrorHandler()

			handler := NewHandler(service, errorHandler)

			router := gin.New()

			router.DELETE("/api/v1/classes/:class_id", handler.Handle)

			var request *http.Request

			switch body := tt.body.(type) {
			case nil:
				request = httptest.NewRequest(http.MethodDelete, tt.url, nil)
			case string:
				request = httptest.NewRequest(http.MethodDelete, tt.url, bytes.NewReader([]byte(body)))
				request.Header.Set("Content-Type", "application/json")
			default:
				reqBody, err := json.Marshal(body)
				require.NoError(t, err)

				request = httptest.NewRequest(http.MethodDelete, tt.url, bytes.NewReader(reqBody))
				request.Header.Set("Content-Type", "application/json")
			}

			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			tt.assert(t, recorder)
		})
	}
}
