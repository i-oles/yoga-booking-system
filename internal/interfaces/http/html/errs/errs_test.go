package viewerrs

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHandleError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		hxRequest bool
		assert    func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "plain navigation renders err.tmpl instead of a blank body",
			hxRequest: false,
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "upss błąd!")
				assert.Empty(t, recorder.Header().Get("Hx-Redirect"))
			},
		},
		{
			name:      "htmx request redirects instead of rendering",
			hxRequest: true,
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assert.Equal(t, "/error", recorder.Header().Get("Hx-Redirect"))
				assert.Empty(t, recorder.Body.String())
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()

			ctx, engine := gin.CreateTestContext(recorder)

			engine.LoadHTMLFiles("../../../../../web/templates/err.tmpl")

			ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)

			if tt.hxRequest {
				ctx.Request.Header.Set("Hx-Request", "true")
			}

			HandleError(ctx, assert.AnError, http.StatusBadRequest)
			ctx.Writer.WriteHeaderNow()

			tt.assert(t, recorder)
		})
	}
}
