package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const traceIDHeader = "X-Trace-ID"

func TraceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(traceIDHeader)
		if traceID == "" {
			traceID = uuid.NewString()
		}

		c.Set("trace_id", traceID)
		c.Writer.Header().Set(traceIDHeader, traceID)

		c.Next()
	}
}

func RequestTimingMiddleware(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		traceID := c.GetString("trace_id")

		log.Info("http_request_started",
			zap.String("trace_id", traceID),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("client_ip", c.ClientIP()),
			zap.Time("started_at", start),
		)

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		size := c.Writer.Size()

		userID := c.GetInt64("userId")
		role := c.GetString("role")

		log.Info("http_request_finished",
			zap.String("trace_id", traceID),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Int("bytes", size),
			zap.Duration("latency", latency),
			zap.Int64("user_id", userID),
			zap.String("role", role),
		)

		log.Info("http_request_duration",
			zap.String("trace_id", traceID),
			zap.String("path", path),
			zap.Duration("latency", latency),
		)
	}
}
