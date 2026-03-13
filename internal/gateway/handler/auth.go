package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	"google.golang.org/grpc/metadata"
)

const (
	RefreshTokenCookieName = "refresh_token"
	RefreshTokenPath       = "/api/v1/auth/refresh"
	AccessTokenCookieName  = "access_token"
	AccessTokenPath        = "/"
)

func (h *Handler) RefreshToken(c *gin.Context) {
	cookieToken, err := c.Cookie(RefreshTokenCookieName)
	if err != nil || strings.TrimSpace(cookieToken) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token cookie missing"})
		return
	}

	ctx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-internal-service", "api_gateway")
	resp, err := h.authClient.RefreshToken(ctx, &authv1.RefreshTokenRequest{
		RefreshToken: cookieToken,
		UserAgent:    c.Request.UserAgent(),
		IpAddress:    c.ClientIP(),
	})
	if err != nil {
		sec := secureCookie(c)
		clearRefreshCookie(c, sec)
		clearAccessCookie(c, sec)
		MapGRPCError(c, err)
		return
	}

	sec := secureCookie(c)
	setRefreshCookie(c, resp.RefreshToken, 3600*24*30, sec)
	setAccessCookie(c, resp.AccessToken, 3600*24, sec)
	c.JSON(http.StatusOK, gin.H{
		"ok": true,
	})

}

func (h *Handler) Logout(c *gin.Context) {
	userID := c.GetInt64("userId")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	cookieToken, _ := c.Cookie(RefreshTokenCookieName)
	if cookieToken != "" {
		parts := strings.Split(cookieToken, ".")
		if len(parts) == 2 {
			if sid, err := strconv.ParseUint(parts[0], 10, 64); err == nil && sid > 0 {
				_, _ = h.authClient.RevokeSession(authGatewayCtx(c), &authv1.RevokeSessionRequest{
					UserId:    userID,
					SessionId: sid,
				})
			}
		}
	}

	sec := secureCookie(c)
	clearRefreshCookie(c, sec)
	clearAccessCookie(c, sec)

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (h *Handler) Register(c *gin.Context) {
	var req authv1.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	resp, err := h.authClient.Register(c.Request.Context(), &req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) Login(c *gin.Context) {
	var req authv1.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	req.UserAgent = c.Request.UserAgent()
	req.IpAddress = c.ClientIP()

	resp, err := h.authClient.Login(c.Request.Context(), &req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	sec := secureCookie(c)

	setRefreshCookie(c, resp.RefreshToken, 3600*24*30, sec)
	setAccessCookie(c, resp.AccessToken, 3600*24, sec)

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
	})
}
func (h *Handler) ListSessions(c *gin.Context) {
	userID := c.GetInt64("userId")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	resp, err := h.authClient.ListSessions(authGatewayCtx(c), &authv1.ListSessionsRequest{UserId: userID})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) RevokeSession(c *gin.Context) {
	userID := c.GetInt64("userId")
	sessionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	_, err = h.authClient.RevokeSession(authGatewayCtx(c), &authv1.RevokeSessionRequest{
		UserId:    userID,
		SessionId: sessionID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session revoked"})
}

func (h *Handler) AssignRole(c *gin.Context) {
	var req struct {
		UserID int64  `json:"user_id" binding:"required"`
		Role   string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.authClient.AssignRole(authGatewayCtx(c), &authv1.AssignRoleRequest{
		UserId: req.UserID,
		Role:   req.Role,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}
func (h *Handler) GetMe(c *gin.Context) {
	userID := c.GetInt64("userId")
	email := c.GetString("email")
	firstName := c.GetString("firstName")
	lastName := c.GetString("lastName")
	role := c.GetString("role")
	universityID := c.GetInt64("universityId")
	departmentID := c.GetInt64("departmentId")
	var deptRoles []string
	if v, ok := c.Get("deptRoles"); ok && v != nil {
		if arr, ok := v.([]string); ok {
			deptRoles = arr
		}
	}
	if deptRoles == nil {
		deptRoles = []string{}
	}
	c.JSON(http.StatusOK, gin.H{
		"id":            userID,
		"email":         email,
		"role":          role,
		"first_name":    firstName,
		"last_name":     lastName,
		"university_id": universityID,
		"department_id": departmentID,
		"dept_roles":    deptRoles,
	})
}
func authGatewayCtx(c *gin.Context) context.Context {
	return metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-internal-service", "api_gateway",
		"x-user-id", strconv.FormatInt(c.GetInt64("userId"), 10),
		"x-user-role", c.GetString("role"),
		"x-university-id", strconv.FormatInt(c.GetInt64("universityId"), 10),
		"x-department-id", strconv.FormatInt(c.GetInt64("departmentId"), 10),
	)
}

func secureCookie(c *gin.Context) bool {
	// Если запрос пришёл по TLS напрямую
	if c.Request.TLS != nil {
		return true
	}
	// Если стоит reverse proxy/ingress, обычно прокидывает X-Forwarded-Proto
	xfp := strings.TrimSpace(strings.ToLower(c.GetHeader("X-Forwarded-Proto")))
	return xfp == "https"
}

func (h *Handler) LogoutCleanup(c *gin.Context) {
	sec := secureCookie(c)
	clearRefreshCookie(c, sec)
	clearAccessCookie(c, sec)
	c.JSON(http.StatusOK, gin.H{"message": "cookies cleared"})
}
func cookieSameSite(secure bool) http.SameSite {
	if secure {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func setRefreshCookie(c *gin.Context, token string, maxAge int, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    token,
		Path:     RefreshTokenPath,
		MaxAge:   maxAge,
		Secure:   secure,
		HttpOnly: true,
		SameSite: cookieSameSite(secure),
	})
}

func clearRefreshCookie(c *gin.Context, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     RefreshTokenPath,
		MaxAge:   -1,
		Secure:   secure,
		HttpOnly: true,
		SameSite: cookieSameSite(secure),
	})
}

func setAccessCookie(c *gin.Context, token string, maxAge int, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     AccessTokenCookieName,
		Value:    token,
		Path:     AccessTokenPath,
		MaxAge:   maxAge,
		Secure:   secure,
		HttpOnly: true,
		SameSite: cookieSameSite(secure),
	})
}

func clearAccessCookie(c *gin.Context, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     AccessTokenCookieName,
		Value:    "",
		Path:     AccessTokenPath,
		MaxAge:   -1,
		Secure:   secure,
		HttpOnly: true,
		SameSite: cookieSameSite(secure),
	})
}
