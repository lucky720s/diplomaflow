package handler

import (
	"net/http"
	"strconv"

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
