package middleware

import "github.com/gin-gonic/gin"

func NormControlAccessMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		if role == "admin" {
			c.Next()
			return
		}
		v, ok := c.Get("deptRoles")
		if ok {
			if slugs, ok := v.([]string); ok {
				for _, s := range slugs {
					if s == "norm_control" {
						c.Next()
						return
					}
				}
			}
		}

		c.AbortWithStatusJSON(403, gin.H{"error": "forbidden"})
	}
}
