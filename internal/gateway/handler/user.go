package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (h *Handler) ListUsers(c *gin.Context) {
	universityID := c.GetInt64("universityId")
	departmentID := c.GetInt64("departmentId")
	roleFromToken := c.GetString("role")

	if roleFromToken == "admin" {
		if v := c.Query("department_id"); v != "" {
			if did, err := strconv.ParseInt(v, 10, 64); err == nil && did > 0 {
				departmentID = did
			}
		}
		if v := c.Query("university_id"); v != "" {
			if uid, err := strconv.ParseInt(v, 10, 64); err == nil && uid > 0 {
				universityID = uid
			}
		}
	}

	role := c.Query("role")

	page := int32(1)
	pageSize := int32(20)

	if p := c.Query("page"); p != "" {
		if v, err := strconv.ParseInt(p, 10, 32); err == nil && v > 0 {
			page = int32(v)
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.ParseInt(ps, 10, 32); err == nil && v > 0 {
			pageSize = int32(v)
		}
	}

	res, err := h.authClient.ListUsers(authGatewayCtx(c), &authv1.ListUsersRequest{
		UniversityId: universityID,
		DepartmentId: departmentID,
		Role:         role,
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	requesterID := c.GetInt64("userId")
	requesterRole := c.GetString("role")

	if requesterRole != "admin" && requesterID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied: you can only view your own profile"})
		return
	}

	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(requesterID, 10),
		"x-user-role", requesterRole,
		"x-internal-service", "api_gateway",
	)

	resp, err := h.authClient.GetUser(ctx, &authv1.GetUserRequest{
		UserId: userID,
	})
	if err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.NotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": st.Message()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": mapUserPreviewToResponse(resp.User),
	})
}

func (h *Handler) UpdateUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	requesterID := c.GetInt64("userId")
	requesterRole := c.GetString("role")

	if requesterRole != "admin" && requesterID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied: you can only edit your own profile"})
		return
	}

	var req struct {
		Email        string `json:"email"`
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		Role         string `json:"role"`
		UniversityID int64  `json:"university_id"`
		DepartmentID int64  `json:"department_id"`
	}

	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if requesterRole != "admin" && (req.Role != "" || req.UniversityID > 0 || req.DepartmentID > 0) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "access denied: only admin can change role, university_id, or department_id",
		})
		return
	}

	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(requesterID, 10),
		"x-user-role", requesterRole,
		"x-internal-service", "api_gateway",
	)

	resp, err := h.authClient.UpdateUser(ctx, &authv1.UpdateUserRequest{
		UserId:       userID,
		Email:        req.Email,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         req.Role,
		UniversityId: req.UniversityID,
		DepartmentId: req.DepartmentID,
	})
	if err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.NotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		case codes.AlreadyExists:
			c.JSON(http.StatusConflict, gin.H{"error": "email already taken"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": st.Message()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":    mapUserPreviewToResponse(resp.User),
		"message": "user updated successfully",
	})
}

func mapUserPreviewToResponse(u *authv1.UserPreview) map[string]interface{} {
	if u == nil {
		return nil
	}
	return map[string]interface{}{
		"id":            u.Id,
		"email":         u.Email,
		"first_name":    u.FirstName,
		"last_name":     u.LastName,
		"role":          u.Role,
		"university_id": u.UniversityId,
		"department_id": u.DepartmentId,
	}
}
