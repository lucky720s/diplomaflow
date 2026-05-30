package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodySize enforces a single upload size limit at the gateway. It rejects
// requests whose declared Content-Length exceeds the limit with a JSON 413 and
// wraps the body with http.MaxBytesReader as a hard guard for chunked/unknown
// length uploads. Keep maxBytes aligned with nginx client_max_body_size and the
// file service limit so behaviour is uniform across the product.
func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes <= 0 {
			c.Next()
			return
		}
		if c.Request.ContentLength > maxBytes {
			abortTooLarge(c, maxBytes)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

func abortTooLarge(c *gin.Context, maxBytes int64) {
	c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
		"error":     "file too large",
		"max_bytes": maxBytes,
	})
}
