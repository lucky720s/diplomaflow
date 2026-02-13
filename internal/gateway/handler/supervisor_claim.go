package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	"google.golang.org/grpc/metadata"
)

func (h *Handler) TeacherClaimTeam(c *gin.Context) {
	teacherID := c.GetInt64("userId")
	role := c.GetString("role")
	if teacherID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if role != "teacher" && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	teamID, err := strconv.ParseInt(c.Param("team_id"), 10, 64)
	if err != nil || teamID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	// IMPORTANT: propagate actor identity to admin_service
	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(teacherID, 10),
		"x-user-role", role,
	)

	createResp, err := h.adminClient.CreateSupervisorRequestByTeam(ctx, &adminv1.CreateSupervisorRequestByTeamReq{
		TeamId:        teamID,
		SupervisorId:  teacherID,
		RequestedBy:   teacherID,
		Message:       "Teacher claimed the team",
		ProposedTopic: "",
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	approveResp, err := h.adminClient.RespondToSupervisorRequest(ctx, &adminv1.RespondToSupervisorRequestReq{
		RequestId:    createResp.RequestId,
		SupervisorId: teacherID,
		Action:       "approve",
		RejectReason: "",
		Comment:      "Claimed",
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, approveResp)
}
