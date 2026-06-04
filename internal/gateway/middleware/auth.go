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

const accessTokenCookieName = "access_token"

func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := ""

		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
				return
			}
			tokenString = strings.TrimSpace(parts[1])
		} else {
			if v, err := c.Cookie(accessTokenCookieName); err == nil {
				tokenString = strings.TrimSpace(v)
			}
		}

		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

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

// RequireDepartmentRole authorizes a request only when the caller holds at
// least one of the given *department* roles (from JWT DeptRoles). Unlike
// RBACMiddleware it never accepts a base role match — a plain `teacher` is not a
// `commission` or `dean_office`. A base-role `admin` is allowed through as an
// explicit override and is flagged in context ("adminOverride") so handlers can
// record it in audit/history.
func RequireDepartmentRole(roles ...string) gin.HandlerFunc {
	required := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		required[r] = struct{}{}
	}

	return func(c *gin.Context) {
		// Admin override — explicit, auditable, never silent.
		if c.GetString("role") == "admin" {
			c.Set("adminOverride", true)
			c.Next()
			return
		}
		if v, ok := c.Get("deptRoles"); ok && v != nil {
			if arr, ok := v.([]string); ok {
				for _, dr := range arr {
					if _, ok := required[dr]; ok {
						c.Next()
						return
					}
				}
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden: required department role"})
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
