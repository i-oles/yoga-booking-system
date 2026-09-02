package home

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"main/internal/application/classes"
	viewErrHandler "main/internal/interfaces/http/html/errs/handler"
	mockclasses "main/mock/classes"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
		LocationLink:    "https://maps.google.com/test",
	}

	tests := []struct {
		name       string
		isVacation bool
		mocks      func(
			service *mockclasses.MockIService,
		)
		assert func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:       "success - home page with upcoming classes",
			isVacation: false,
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
				assert.NotContains(t, recorder.Body.String(), "jestem na urlopie")
			},
		},
		{
			name:       "success - home page during vacation",
			isVacation: true,
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
				assert.Contains(t, recorder.Body.String(), "jestem na urlopie")
			},
		},
		{
			name:       "failure - list classes error",
			isVacation: false,
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
	}

	gin.SetMode(gin.TestMode)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			service := mockclasses.NewMockIService(ctrl)

			tt.mocks(service)

			errorHandler := viewErrHandler.NewErrorHandler()

			handler := NewHandler(service, errorHandler, tt.isVacation)

			router := gin.New()

			router.LoadHTMLFiles(
				"../../../../../../web/templates/index.html",
				"../../../../../../web/templates/err.tmpl",
			)

			router.GET("/", handler.Handle)

			request := httptest.NewRequest(
				http.MethodGet,
				"/",
				nil,
			)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			tt.assert(t, recorder)
		})
	}
}
