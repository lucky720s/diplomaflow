package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type JwtClaims struct {
	jwt.RegisteredClaims
	Id           int64    `json:"Id"`
	Email        string   `json:"Email"`
	Role         string   `json:"Role"` // base role: student|teacher|admin
	DeptRoles    []string `json:"DeptRoles"`
	FirstName    string   `json:"FirstName"`
	LastName     string   `json:"LastName"`
	UniversityID int64    `json:"UniversityID"`
	DepartmentID int64    `json:"DepartmentID"`
}

func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			return
		}

		tokenString := parts[1]
		claims := &JwtClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		c.Set("userId", claims.Id)
		c.Set("role", claims.Role)
		c.Set("deptRoles", claims.DeptRoles)

		c.Set("firstName", claims.FirstName)
		c.Set("lastName", claims.LastName)
		c.Set("universityId", claims.UniversityID)
		c.Set("departmentId", claims.DepartmentID)
		c.Set("email", claims.Email)

		c.Next()
	}
}

func RBACMiddleware(allowedRoles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}

	return func(c *gin.Context) {
		// 1) base role
		role := c.GetString("role")
		if _, ok := allowed[role]; ok {
			c.Next()
			return
		}
		if v, ok := c.Get("deptRoles"); ok && v != nil {
			if arr, ok := v.([]string); ok {
				for _, dr := range arr {
					if _, ok := allowed[dr]; ok {
						c.Next()
						return
					}
				}
			}
		}

		c.AbortWithStatusJSON(403, gin.H{"error": "forbidden"})
	}
}
