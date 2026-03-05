package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (h *Handler) ListUsers(c *gin.Context) {
	role := c.Query("role")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	resp, err := h.authClient.ListUsers(authGatewayCtx(c), &authv1.ListUsersRequest{
		UniversityId: c.GetInt64("universityId"),
		DepartmentId: c.GetInt64("departmentId"),
		Role:         role,
		Page:         int32(page),
		PageSize:     int32(pageSize),
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	// Если role=teacher — обогащаем teams_count и max_teams
	if role == "teacher" && len(resp.Users) > 0 {
		supResp, supErr := h.adminClient.ListSupervisors(adminPanelCtx(c), &adminv1.ListSupervisorsRequest{
			UniversityId: c.GetInt64("universityId"),
			DepartmentId: c.GetInt64("departmentId"),
			Page:         1,
			PageSize:     200,
		})

		if supErr == nil && supResp != nil {
			supMap := make(map[int64]*adminv1.SupervisorDetails)
			for _, s := range supResp.Supervisors {
				supMap[s.Id] = s
			}

			type teacherResp struct {
				ID           int64  `json:"id"`
				Email        string `json:"email"`
				FirstName    string `json:"first_name"`
				LastName     string `json:"last_name"`
				Role         string `json:"role"`
				UniversityID int64  `json:"university_id"`
				DepartmentID int64  `json:"department_id"`
				TeamsCount   int32  `json:"teams_count"`
				MaxTeams     int32  `json:"max_teams"`
			}

			users := make([]teacherResp, 0, len(resp.Users))
			for _, u := range resp.Users {
				t := teacherResp{
					ID:           u.Id,
					Email:        u.Email,
					FirstName:    u.FirstName,
					LastName:     u.LastName,
					Role:         u.Role,
					UniversityID: u.UniversityId,
					DepartmentID: u.DepartmentId,
				}
				if sup, ok := supMap[u.Id]; ok {
					t.TeamsCount = sup.TeamsCount
					t.MaxTeams = sup.MaxTeams
				}
				users = append(users, t)
			}

			c.JSON(http.StatusOK, gin.H{
				"users":       users,
				"total_count": resp.TotalCount,
			})
			return
		}
		// fallback если admin_service недоступен
	}

	c.JSON(http.StatusOK, gin.H{
		"users":       resp.Users,
		"total_count": resp.TotalCount,
	})
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
