package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CorsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	// fallback (если конфиг не передали)
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{
			"https://diploma-flow.iitu.kz",
			"http://localhost:3000",
			"http://localhost:5173",
		}
	}

	// whitelist map
	originsMap := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		originsMap[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Всегда корректно отвечаем на preflight (OPTIONS).
		// Но CORS-заголовки даём только если Origin разрешён.
		if origin != "" {
			if _, ok := originsMap[origin]; ok {
				h := c.Writer.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Vary", "Origin")

				// credentials можно только когда origin НЕ "*"
				h.Set("Access-Control-Allow-Credentials", "true")
				h.Set("Access-Control-Allow-Headers",
					"Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Trace-ID")
				h.Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
				h.Set("Access-Control-Max-Age", "86400")
			}
			// если Origin не в whitelist — ничего не выставляем,
			// браузер сам заблокирует доступ (как и должно быть)
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
