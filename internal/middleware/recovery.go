package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"voca-test/internal/util"
)

// Recovery handles panics and logs them via slog
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID := c.GetString("request_id")
				slog.Error("panic recovered",
					slog.Any("error", err),
					slog.String("request_id", requestID),
				)
				util.ErrorResponse(c, http.StatusInternalServerError, "Internal Server Error", nil)
				c.Abort()
			}
		}()
		c.Next()
	}
}
