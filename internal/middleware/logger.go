package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger logs details of each HTTP request using slog
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		requestID := c.GetString("request_id")

		attributes := []slog.Attr{
			slog.Int("status", status),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("query", query),
			slog.String("ip", c.ClientIP()),
			slog.Duration("latency", latency),
			slog.String("user_agent", c.Request.UserAgent()),
			slog.String("request_id", requestID),
		}

		if len(c.Errors) > 0 {
			for _, e := range c.Errors.Errors() {
				slog.Error("request error", slog.String("error", e), slog.String("request_id", requestID))
			}
		}

		if status >= 500 {
			slog.LogAttrs(c.Request.Context(), slog.LevelError, "request failed", attributes...)
		} else if status >= 400 {
			slog.LogAttrs(c.Request.Context(), slog.LevelWarn, "request warning", attributes...)
		} else {
			slog.LogAttrs(c.Request.Context(), slog.LevelInfo, "request completed", attributes...)
		}
	}
}
