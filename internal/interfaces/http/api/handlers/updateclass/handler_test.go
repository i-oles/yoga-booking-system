package updateclass

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
			name: "success - class update",
			url:  "/api/v1/classes/" + testClassID.String(),
			body: map[string]any{
				"class_name": "Evening Yoga",
			},
			mocks: func(
				service *mockclasses.MockIService,
			) {
				service.EXPECT().
					UpdateClass(gomock.Any(), testClassID, gomock.Any()).
					Return(classes.ClassData{
						ID:              testClassID,
						StartTime:       time.Date(2026, 8, 10, 18, 30, 0, 0, time.UTC),
						ClassLevel:      "beginner",
						ClassName:       "Evening Yoga",
						CurrentCapacity: 5,
						MaxCapacity:     10,
						Location:        "Warsaw",
					}, nil)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "Evening Yoga")
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
			body: map[string]any{
				"class_name": "Evening Yoga",
			},
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
			name: "failure - class update error",
			url:  "/api/v1/classes/" + testClassID.String(),
			body: map[string]any{
				"class_name": "Evening Yoga",
			},
			mocks: func(
				service *mockclasses.MockIService,
			) {
				service.EXPECT().
					UpdateClass(gomock.Any(), testClassID, gomock.Any()).
					Return(classes.ClassData{}, assert.AnError)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "failure - message required when updating location",
			url:  "/api/v1/classes/" + testClassID.String(),
			body: map[string]any{
				"location": "Krakow",
			},
			mocks: func(
				service *mockclasses.MockIService,
			) {
				service.EXPECT().
					UpdateClass(gomock.Any(), testClassID, gomock.Any()).
					Return(classes.ClassData{}, domainErrs.ErrValidation(assert.AnError))
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "failure - message required when updating start time",
			url:  "/api/v1/classes/" + testClassID.String(),
			body: map[string]any{
				"start_time": time.Date(2026, 9, 10, 18, 30, 0, 0, time.UTC).Format(time.RFC3339),
			},
			mocks: func(
				service *mockclasses.MockIService,
			) {
				service.EXPECT().
					UpdateClass(gomock.Any(), testClassID, gomock.Any()).
					Return(classes.ClassData{}, domainErrs.ErrValidation(assert.AnError))
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
				"class_name": "Evening Yoga",
			},
			mocks: func(
				service *mockclasses.MockIService,
			) {
				service.EXPECT().
					UpdateClass(gomock.Any(), testClassID, gomock.Any()).
					Return(classes.ClassData{}, domainErrs.ErrNotFound(assert.AnError))
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

			router.PATCH("/api/v1/classes/:class_id", handler.Handle)

			bodyStr, ok := tt.body.(string)

			var reqBody []byte

			if ok {
				reqBody = []byte(bodyStr)
			} else {
				var err error

				reqBody, err = json.Marshal(tt.body)
				require.NoError(t, err)
			}

			request := httptest.NewRequest(
				http.MethodPatch,
				tt.url,
				bytes.NewReader(reqBody),
			)
			request.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			tt.assert(t, recorder)
		})
	}
}
