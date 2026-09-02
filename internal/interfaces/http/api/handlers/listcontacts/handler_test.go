package listcontacts

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"main/internal/domain/models"
	apiErrHandler "main/internal/interfaces/http/api/errs/handler"
	mock "main/mock"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	testContact := models.Contact{
		ID:        1,
		Email:     "anna@example.com",
		FirstName: "Anna",
		LastName:  "Kowalska",
	}

	tests := []struct {
		name  string
		mocks func(
			contactsRepo *mock.MockIContacts,
		)
		assert func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "success - list contacts",
			mocks: func(
				contactsRepo *mock.MockIContacts,
			) {
				contactsRepo.EXPECT().
					List(gomock.Any()).
					Return([]models.Contact{testContact}, nil)
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
				contactsRepo *mock.MockIContacts,
			) {
				contactsRepo.EXPECT().
					List(gomock.Any()).
					Return([]models.Contact{}, nil)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.JSONEq(t, "[]", recorder.Body.String())
			},
		},
		{
			name: "failure - list contacts error",
			mocks: func(
				contactsRepo *mock.MockIContacts,
			) {
				contactsRepo.EXPECT().
					List(gomock.Any()).
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

			contactsRepo := mock.NewMockIContacts(ctrl)

			tt.mocks(contactsRepo)

			errorHandler := apiErrHandler.NewErrorHandler()

			handler := NewHandler(contactsRepo, errorHandler)

			router := gin.New()

			router.GET("/api/v1/contacts", handler.Handle)

			request := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/contacts",
				nil,
			)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			tt.assert(t, recorder)
		})
	}
}
