package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
)

const (
	RefreshTokenCookieName = "refresh_token"
	RefreshTokenPath       = "/api/v1/auth/refresh"
)

func (h *Handler) RefreshToken(c *gin.Context) {
	cookieToken, err := c.Cookie(RefreshTokenCookieName)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token cookie missing"})
		return
	}
	resp, err := h.authClient.RefreshToken(c.Request.Context(), &authv1.RefreshTokenRequest{
		RefreshToken: cookieToken,
		UserAgent:    c.Request.UserAgent(),
		IpAddress:    c.ClientIP(),
	})

	if err != nil {
		c.SetCookie(RefreshTokenCookieName, "", -1, RefreshTokenPath, "", false, true)
		MapGRPCError(c, err)
		return
	}
	c.SetCookie(
		RefreshTokenCookieName,
		resp.RefreshToken,
		3600*24*30,
		RefreshTokenPath,
		"",
		false,
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"access_token": resp.AccessToken,
	})
}
func (h *Handler) Logout(c *gin.Context) {
	c.SetCookie(RefreshTokenCookieName, "", -1, RefreshTokenPath, "", false, true)
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

	c.SetCookie(
		RefreshTokenCookieName,
		resp.RefreshToken,
		3600*24*30,
		RefreshTokenPath,
		"",
		false,
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"access_token": resp.AccessToken,
	})
}

func (h *Handler) ListSessions(c *gin.Context) {
	userID := c.GetInt64("userId")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	resp, err := h.authClient.ListSessions(c.Request.Context(), &authv1.ListSessionsRequest{UserId: userID})
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
	_, err = h.authClient.RevokeSession(c.Request.Context(), &authv1.RevokeSessionRequest{
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

	res, err := h.authClient.AssignRole(c.Request.Context(), &authv1.AssignRoleRequest{
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

	c.JSON(http.StatusOK, gin.H{
		"id":            userID,
		"email":         email,
		"role":          role,
		"first_name":    firstName,
		"last_name":     lastName,
		"university_id": universityID,
		"department_id": departmentID,
	})
}
