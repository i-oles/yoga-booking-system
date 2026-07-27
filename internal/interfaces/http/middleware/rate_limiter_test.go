package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func TestGlobalRateLimit(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requests       int
		expectedStatus []int
	}{
		{
			name:     "allows burst requests",
			requests: 2,
			expectedStatus: []int{
				http.StatusOK,
				http.StatusOK,
			},
		},
		{
			name:     "blocks requests over limit",
			requests: 3,
			expectedStatus: []int{
				http.StatusOK,
				http.StatusOK,
				http.StatusTooManyRequests,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			limiter := rate.NewLimiter(rate.Limit(1), 2)

			router := gin.New()
			router.Use(GlobalRateLimit(limiter))

			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "ok",
				})
			})

			for idx := range tt.requests {
				recorder := httptest.NewRecorder()

				req := httptest.NewRequest(
					http.MethodGet,
					"/test",
					nil,
				)

				router.ServeHTTP(recorder, req)

				if recorder.Code != tt.expectedStatus[idx] {
					t.Errorf(
						"request %d: expected %d, got %d",
						idx+1,
						tt.expectedStatus[idx],
						recorder.Code,
					)
				}
			}
		})
	}
}
