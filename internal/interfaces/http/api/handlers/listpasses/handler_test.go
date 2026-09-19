package listpasses

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"main/internal/application/passes"
	apiErrHandler "main/internal/interfaces/http/api/errs/handler"
	mockpasses "main/mock/passes"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	testPass := passes.PassPresentation{
		ID:         1,
		Email:      "anna@example.com",
		TotalSlots: 10,
		UsedSlots:  3,
		CreatedAt:  time.Date(2026, 8, 10, 18, 30, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 8, 10, 18, 30, 0, 0, time.UTC),
	}

	tests := []struct {
		name  string
		mocks func(
			service *mockpasses.MockIService,
		)
		assert func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "success - list passes",
			mocks: func(
				service *mockpasses.MockIService,
			) {
				service.EXPECT().
					ListPasses(gomock.Any()).
					Return([]passes.PassPresentation{testPass}, nil)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "anna@example.com")
			},
		},
		{
			name: "success - empty list",
			mocks: func(
				service *mockpasses.MockIService,
			) {
				service.EXPECT().
					ListPasses(gomock.Any()).
					Return([]passes.PassPresentation{}, nil)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.JSONEq(t, "[]", recorder.Body.String())
			},
		},
		{
			name: "failure - list passes error",
			mocks: func(
				service *mockpasses.MockIService,
			) {
				service.EXPECT().
					ListPasses(gomock.Any()).
					Return(nil, assert.AnError)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
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

			router.GET("/api/v1/passes", handler.Handle)

			request := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/passes",
				nil,
			)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			tt.assert(t, recorder)
		})
	}
}
