package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	"google.golang.org/grpc/metadata"
)

func (h *Handler) CreateSupervisorRequestByTeam(c *gin.Context) {
	userID := c.GetInt64("userId")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	teamID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || teamID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	var req struct {
		SupervisorID  int64  `json:"supervisor_id" binding:"required"`
		Message       string `json:"message"`
		ProposedTopic string `json:"proposed_topic"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}
	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(userID, 10),
		"x-user-role", c.GetString("role"),
	)
	// ✅ TEAM-FIRST RPC (no project_id at this stage)
	resp, err := h.adminClient.CreateSupervisorRequestByTeam(ctx, &adminv1.CreateSupervisorRequestByTeamReq{
		TeamId:        teamID,
		SupervisorId:  req.SupervisorID,
		RequestedBy:   userID,
		Message:       req.Message,
		ProposedTopic: req.ProposedTopic,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	// best-effort notification
	_, _ = h.notificationClient.SendNotification(c.Request.Context(), &notificationv1.SendNotificationRequest{
		UserId:  req.SupervisorID,
		Title:   "New supervisor request",
		Message: "A team requested you as a supervisor.",
		Link:    "/dashboard/supervisor-requests",
		Type:    "supervisor_request",
	})

	c.JSON(http.StatusCreated, resp)
}
func (h *Handler) GetAvailableTeams(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")
	if departmentID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "department_id is required"})
		return
	}

	page := int32(1)
	pageSize := int32(20)
	if v := c.Query("page"); v != "" {
		if x, err := strconv.ParseInt(v, 10, 32); err == nil && x > 0 {
			page = int32(x)
		}
	}
	if v := c.Query("page_size"); v != "" {
		if x, err := strconv.ParseInt(v, 10, 32); err == nil && x > 0 {
			pageSize = int32(x)
		}
	}

	resp, err := h.adminClient.ListAvailableTeams(c.Request.Context(), &adminv1.ListAvailableTeamsRequest{
		DepartmentId: departmentID,
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	teams := make([]gin.H, 0, len(resp.Teams))
	for _, t := range resp.Teams {
		if t == nil {
			continue
		}
		members := make([]gin.H, 0, len(t.Members))
		for _, m := range t.Members {
			if m == nil {
				continue
			}
			members = append(members, gin.H{
				"user_id":   m.UserId,
				"full_name": m.FullName,
				"email":     m.Email,
				"role":      m.Role,
			})
		}

		createdAt := ""
		if t.CreatedAt != nil {
			createdAt = t.CreatedAt.AsTime().UTC().Format(time.RFC3339)
		}

		teams = append(teams, gin.H{
			"id":           t.Id,
			"name":         t.Name,
			"member_count": t.MemberCount,
			"members":      members,
			"created_at":   createdAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"teams":       teams,
		"total_count": resp.TotalCount,
	})
}
func (h *Handler) ListSupervisorsForStudents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	resp, err := h.adminClient.ListSupervisors(
		adminPanelCtx(c),
		&adminv1.ListSupervisorsRequest{
			DepartmentId: c.GetInt64("departmentId"),
			UniversityId: c.GetInt64("universityId"),
			Page:         int32(page),
			PageSize:     int32(pageSize),
		},
	)
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	// Возвращаем с teams_count и max_teams
	type supervisorResp struct {
		ID         int64  `json:"id"`
		FullName   string `json:"full_name"`
		Email      string `json:"email"`
		Position   string `json:"position"`
		TeamsCount int32  `json:"teams_count"`
		MaxTeams   int32  `json:"max_teams"`
	}

	supervisors := make([]supervisorResp, 0, len(resp.Supervisors))
	for _, s := range resp.Supervisors {
		supervisors = append(supervisors, supervisorResp{
			ID:         s.Id,
			FullName:   s.FullName,
			Email:      s.Email,
			Position:   s.Position,
			TeamsCount: s.TeamsCount,
			MaxTeams:   s.MaxTeams,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"supervisors": supervisors,
		"total_count": resp.TotalCount,
	})
}
