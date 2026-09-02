package activatepass

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"main/internal/application/passes"
	domainErrs "main/internal/domain/errs/api"
	"main/internal/domain/models"
	apiErrHandler "main/internal/interfaces/http/api/errs/handler"
	mockpasses "main/mock/passes"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		body  any
		mocks func(
			service *mockpasses.MockIService,
		)
		assert func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "success - pass activation",
			body: map[string]any{
				"email":                  "anna@example.com",
				"initial_assigned_slots": 0,
				"total_slots":            4,
			},
			mocks: func(
				service *mockpasses.MockIService,
			) {
				service.EXPECT().
					ActivatePass(gomock.Any(), "anna@example.com", 0, 4).
					Return(passes.PassActivation{
						Pass: models.Pass{
							ID:         1,
							Email:      "anna@example.com",
							TotalSlots: 4,
							CreatedAt:  time.Now(),
							UpdatedAt:  time.Now(),
						},
						UpdatedBookings: []models.Booking{},
					}, nil)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "anna@example.com")
			},
		},
		{
			name: "failure - malformed request body",
			body: "{invalid",
			mocks: func(
				service *mockpasses.MockIService,
			) {
				// No service call expected.
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "failure - missing email",
			body: map[string]any{
				"initial_assigned_slots": 0,
				"total_slots":            4,
			},
			mocks: func(
				service *mockpasses.MockIService,
			) {
				// No service call expected.
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "failure - total slots should be greater than 0",
			body: map[string]any{
				"email":                  "anna@example.com",
				"initial_assigned_slots": 0,
				"total_slots":            0,
			},
			mocks: func(
				service *mockpasses.MockIService,
			) {
				// No service call expected.
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "failure - pass activation error",
			body: map[string]any{
				"email":                  "anna@example.com",
				"initial_assigned_slots": 0,
				"total_slots":            4,
			},
			mocks: func(
				service *mockpasses.MockIService,
			) {
				service.EXPECT().
					ActivatePass(gomock.Any(), "anna@example.com", 0, 4).
					Return(passes.PassActivation{}, assert.AnError)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "failure - initial assigned slots exceed total slots",
			body: map[string]any{
				"email":                  "anna@example.com",
				"initial_assigned_slots": 5,
				"total_slots":            4,
			},
			mocks: func(
				service *mockpasses.MockIService,
			) {
				service.EXPECT().
					ActivatePass(gomock.Any(), "anna@example.com", 5, 4).
					Return(passes.PassActivation{}, domainErrs.ErrValidation(assert.AnError))
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

			service := mockpasses.NewMockIService(ctrl)

			tt.mocks(service)

			errorHandler := apiErrHandler.NewErrorHandler()

			handler := NewHandler(service, errorHandler)

			router := gin.New()

			router.PUT("/api/v1/passes", handler.Handle)

			bodyBytes, ok := tt.body.(string)

			var reqBody []byte

			if ok {
				reqBody = []byte(bodyBytes)
			} else {
				var err error

				reqBody, err = json.Marshal(tt.body)
				require.NoError(t, err)
			}

			request := httptest.NewRequest(
				http.MethodPut,
				"/api/v1/passes",
				bytes.NewReader(reqBody),
			)
			request.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			tt.assert(t, recorder)
		})
	}
}
