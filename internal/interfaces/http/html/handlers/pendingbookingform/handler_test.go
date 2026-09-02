package pendingbookingform

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	testClassID := uuid.New()

	tests := []struct {
		name   string
		url    string
		assert func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "success - pending booking form rendered",
			url:  "/classes/" + testClassID.String() + "/pending_bookings/form",
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "book-block-"+testClassID.String())
				assert.Contains(t, recorder.Body.String(), `value="`+testClassID.String()+`"`)
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := NewHandler()

			router := gin.New()

			router.LoadHTMLFiles(
				"../../../../../../web/templates/pending_booking_form.tmpl",
			)

			router.GET("/classes/:class_id/pending_bookings/form", handler.Handle)

			request := httptest.NewRequest(
				http.MethodGet,
				tt.url,
				nil,
			)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			tt.assert(t, recorder)
		})
	}
}
