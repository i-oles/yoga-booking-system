package createclasses

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainErrs "main/internal/domain/errs/api"
	"main/internal/domain/models"
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

	testStartTime := time.Date(2026, 8, 10, 18, 30, 0, 0, time.UTC)

	tests := []struct {
		name  string
		body  any
		mocks func(
			service *mockclasses.MockIService,
		)
		assert func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "success - classes creation",
			body: []map[string]any{
				{
					"start_time":   testStartTime.Format(time.RFC3339),
					"class_level":  "beginner",
					"class_name":   "Morning Yoga",
					"max_capacity": 10,
					"location":     "Warsaw",
				},
			},
			mocks: func(
				service *mockclasses.MockIService,
			) {
				service.EXPECT().
					CreateClasses(gomock.Any(), gomock.Any()).
					Return([]models.Class{
						{
							ID:          uuid.New(),
							StartTime:   testStartTime,
							ClassLevel:  "beginner",
							ClassName:   "Morning Yoga",
							MaxCapacity: 10,
							Location:    "Warsaw",
						},
					}, nil)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusCreated, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "Morning Yoga")
			},
		},
		{
			name: "success - multiple classes creation",
			body: []map[string]any{
				{
					"start_time":   testStartTime.Format(time.RFC3339),
					"class_level":  "beginner",
					"class_name":   "Morning Yoga",
					"max_capacity": 10,
					"location":     "Warsaw",
				},
				{
					"start_time":   testStartTime.Add(24 * time.Hour).Format(time.RFC3339),
					"class_level":  "advanced",
					"class_name":   "Evening Yoga",
					"max_capacity": 8,
					"location":     "Krakow",
				},
			},
			mocks: func(
				service *mockclasses.MockIService,
			) {
				service.EXPECT().
					CreateClasses(gomock.Any(), gomock.Any()).
					Return([]models.Class{
						{
							ID:          uuid.New(),
							StartTime:   testStartTime,
							ClassLevel:  "beginner",
							ClassName:   "Morning Yoga",
							MaxCapacity: 10,
							Location:    "Warsaw",
						},
						{
							ID:          uuid.New(),
							StartTime:   testStartTime.Add(24 * time.Hour),
							ClassLevel:  "advanced",
							ClassName:   "Evening Yoga",
							MaxCapacity: 8,
							Location:    "Krakow",
						},
					}, nil)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusCreated, recorder.Code)

				var response []map[string]any

				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Len(t, response, 2)
				assert.Contains(t, recorder.Body.String(), "Morning Yoga")
				assert.Contains(t, recorder.Body.String(), "Evening Yoga")
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
			name: "failure - missing class name",
			body: []map[string]any{
				{
					"start_time":   testStartTime.Format(time.RFC3339),
					"class_level":  "beginner",
					"max_capacity": 10,
					"location":     "Warsaw",
				},
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
			name: "failure - max capacity too low",
			body: []map[string]any{
				{
					"start_time":   testStartTime.Format(time.RFC3339),
					"class_level":  "beginner",
					"class_name":   "Morning Yoga",
					"max_capacity": 0,
					"location":     "Warsaw",
				},
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
			name: "failure - classes creation error",
			body: []map[string]any{
				{
					"start_time":   testStartTime.Format(time.RFC3339),
					"class_level":  "beginner",
					"class_name":   "Morning Yoga",
					"max_capacity": 10,
					"location":     "Warsaw",
				},
			},
			mocks: func(
				service *mockclasses.MockIService,
			) {
				service.EXPECT().
					CreateClasses(gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "failure - classes validation error",
			body: []map[string]any{
				{
					"start_time":   testStartTime.Format(time.RFC3339),
					"class_level":  "beginner",
					"class_name":   "Morning Yoga",
					"max_capacity": 10,
					"location":     "Warsaw",
				},
			},
			mocks: func(
				service *mockclasses.MockIService,
			) {
				service.EXPECT().
					CreateClasses(gomock.Any(), gomock.Any()).
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

			router.POST("/api/v1/classes", handler.Handle)

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
				http.MethodPost,
				"/api/v1/classes",
				bytes.NewReader(reqBody),
			)
			request.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			tt.assert(t, recorder)
		})
	}
}
