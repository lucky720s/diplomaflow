package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
)

func (h *Handler) CreateTeam(c *gin.Context) {
	var reqBody struct {
		Name      string  `json:"name" binding:"required"`
		ProjectID int64   `json:"project_id"`
		MemberIDs []int64 `json:"member_ids"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json: " + err.Error()})
		return
	}
	leaderID := c.GetInt64("userId")
	if leaderID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	uniqueMembers := make(map[int64]bool)
	var cleanMemberIDs []int64
	for _, id := range reqBody.MemberIDs {
		if id == leaderID {
			continue
		}
		if !uniqueMembers[id] {
			uniqueMembers[id] = true
			cleanMemberIDs = append(cleanMemberIDs, id)
		}
	}
	req := &teamv1.CreateTeamRequest{
		Name:      reqBody.Name,
		ProjectId: reqBody.ProjectID,
		MemberIds: cleanMemberIDs,
		LeaderId:  leaderID,
	}
	res, err := h.teamClient.CreateTeam(c.Request.Context(), req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusCreated, res)
}

func (h *Handler) GetTeam(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	res, err := h.teamClient.GetTeam(c.Request.Context(), &teamv1.GetTeamRequest{TeamId: id})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetAvailableStudents(c *gin.Context) {
	uniID := c.GetInt64("universityId")
	userID := c.GetInt64("userId")

	res, err := h.teamClient.GetAvailableStudents(c.Request.Context(), &teamv1.GetAvailableStudentsRequest{UniversityId: uniID, ExcludeUserId: userID})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
func (h *Handler) AssignProjectToTeam(c *gin.Context) {
	teamIDStr := c.Param("id")
	teamID, err := strconv.ParseInt(teamIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	var req struct {
		ProjectID int64 `json:"project_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = h.teamClient.AssignProject(c.Request.Context(), &teamv1.AssignProjectRequest{
		TeamId:    teamID,
		ProjectId: req.ProjectID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
func (h *Handler) GetMyInvites(c *gin.Context) {
	userID := c.GetInt64("userId")
	res, err := h.teamClient.GetMyInvites(c.Request.Context(), &teamv1.GetMyInvitesRequest{UserId: userID})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) RespondToInvite(c *gin.Context) {
	userID := c.GetInt64("userId")
	inviteID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var jsonReq struct {
		Accept bool `json:"accept"`
	}
	if err := c.ShouldBindJSON(&jsonReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.teamClient.RespondToInvite(c.Request.Context(), &teamv1.RespondToInviteRequest{
		InviteId: inviteID,
		UserId:   userID,
		Accept:   jsonReq.Accept,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
func (h *Handler) GetMyTeam(c *gin.Context) {
	userID := c.GetInt64("userId")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	res, err := h.teamClient.GetMyTeam(c.Request.Context(), &teamv1.GetMyTeamRequest{
		UserId: userID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}
