package createcontacts

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	repoErrs "main/internal/infrastructure/errs"

	"main/internal/domain/models"
	apiErrHandler "main/internal/interfaces/http/api/errs/handler"
	mock "main/mock"

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
			contactsRepo *mock.MockIContacts,
		)
		assert func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "success - contacts creation",
			body: []map[string]any{
				{
					"email":      "anna@example.com",
					"first_name": "Anna",
					"last_name":  "Kowalska",
				},
			},
			mocks: func(
				contactsRepo *mock.MockIContacts,
			) {
				contactsRepo.EXPECT().
					Insert(gomock.Any(), "anna@example.com", "Anna", "Kowalska").
					Return(models.Contact{
						ID:        1,
						Email:     "anna@example.com",
						FirstName: "Anna",
						LastName:  "Kowalska",
					}, nil)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "anna@example.com")
			},
		},
		{
			name: "success - skips contacts that already exist",
			body: []map[string]any{
				{
					"email":      "anna@example.com",
					"first_name": "Anna",
					"last_name":  "Kowalska",
				},
				{
					"email":      "john@example.com",
					"first_name": "John",
					"last_name":  "Smith",
				},
			},
			mocks: func(
				contactsRepo *mock.MockIContacts,
			) {
				contactsRepo.EXPECT().
					Insert(gomock.Any(), "anna@example.com", "Anna", "Kowalska").
					Return(models.Contact{}, repoErrs.ErrAlreadyExist)

				contactsRepo.EXPECT().
					Insert(gomock.Any(), "john@example.com", "John", "Smith").
					Return(models.Contact{
						ID:        2,
						Email:     "john@example.com",
						FirstName: "John",
						LastName:  "Smith",
					}, nil)
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusOK, recorder.Code)

				var response []map[string]any

				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Len(t, response, 1)
				assert.Contains(t, recorder.Body.String(), "john@example.com")
				assert.NotContains(t, recorder.Body.String(), "anna@example.com")
			},
		},
		{
			name: "failure - malformed request body",
			body: "{invalid",
			mocks: func(
				contactsRepo *mock.MockIContacts,
			) {
				// No repository call expected.
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "failure - missing email",
			body: []map[string]any{
				{
					"first_name": "Anna",
					"last_name":  "Kowalska",
				},
			},
			mocks: func(
				contactsRepo *mock.MockIContacts,
			) {
				// No repository call expected.
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "failure - contact insert error",
			body: []map[string]any{
				{
					"email":      "anna@example.com",
					"first_name": "Anna",
					"last_name":  "Kowalska",
				},
			},
			mocks: func(
				contactsRepo *mock.MockIContacts,
			) {
				contactsRepo.EXPECT().
					Insert(gomock.Any(), "anna@example.com", "Anna", "Kowalska").
					Return(models.Contact{}, assert.AnError)
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

			router.POST("/api/v1/contacts", handler.Handle)

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
				"/api/v1/contacts",
				bytes.NewReader(reqBody),
			)
			request.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			tt.assert(t, recorder)
		})
	}
}
