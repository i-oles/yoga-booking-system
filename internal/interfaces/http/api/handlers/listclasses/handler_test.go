package listclasses

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"main/internal/application/classes"
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

	testClass := classes.ClassPresentation{
		ID:              uuid.New(),
		StartTime:       time.Date(2026, 8, 10, 18, 30, 0, 0, time.UTC),
		ClassLevel:      "beginner",
		ClassName:       "Morning Yoga",
		CurrentCapacity: 5,
		MaxCapacity:     10,
		Location:        "Warsaw",
	}

	tests := []struct {
		name  string
		body  any
		mocks func(
			service *mockclasses.MockIService,
		)
		assert func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "success - list classes",
			body: map[string]any{
				"only_upcoming_classes": true,
				"classes_limit":         5,
			},
			mocks: func(
				service *mockclasses.MockIService,
			) {
				service.EXPECT().
					ListClasses(gomock.Any(), true, gomock.Any()).
					Return([]classes.ClassPresentation{testClass}, nil)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Contains(t, recorder.Body.String(), testClass.ClassName)
			},
		},
		{
			name: "failure - empty request body",
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
			name: "failure - list classes error",
			body: map[string]any{
				"only_upcoming_classes": true,
			},
			mocks: func(
				service *mockclasses.MockIService,
			) {
				service.EXPECT().
					ListClasses(gomock.Any(), true, gomock.Any()).
					Return(nil, assert.AnError)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "failure - negative classes limit",
			body: map[string]any{
				"only_upcoming_classes": true,
				"classes_limit":         -1,
			},
			mocks: func(
				service *mockclasses.MockIService,
			) {
				service.EXPECT().
					ListClasses(gomock.Any(), true, gomock.Any()).
					Return(nil, domainErrs.ErrValidation(assert.AnError))
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
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

			router.GET("/api/v1/classes", handler.Handle)

			var request *http.Request

			switch body := tt.body.(type) {
			case nil:
				request = httptest.NewRequest(http.MethodGet, "/api/v1/classes", nil)
			case string:
				request = httptest.NewRequest(http.MethodGet, "/api/v1/classes", bytes.NewReader([]byte(body)))
				request.Header.Set("Content-Type", "application/json")
			default:
				reqBody, err := json.Marshal(body)
				require.NoError(t, err)

				request = httptest.NewRequest(http.MethodGet, "/api/v1/classes", bytes.NewReader(reqBody))
				request.Header.Set("Content-Type", "application/json")
			}

			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			tt.assert(t, recorder)
		})
	}
}
