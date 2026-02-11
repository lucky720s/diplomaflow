package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/metadata"
)

func (h *Handler) CreateTeam(c *gin.Context) {
	var reqBody struct {
		Name      string  `json:"name" binding:"required"`
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

	departmentID := c.GetInt64("departmentId")
	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-department-id", strconv.FormatInt(departmentID, 10),
		"x-university-id", strconv.FormatInt(c.GetInt64("universityId"), 10),
		"x-user-id", strconv.FormatInt(leaderID, 10),
	)

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
		MemberIds: cleanMemberIDs,
		LeaderId:  leaderID,
	}

	res, err := h.teamClient.CreateTeam(ctx, req)
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
	departmentID := c.GetInt64("departmentId")

	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-department-id", strconv.FormatInt(departmentID, 10),
		"x-user-id", strconv.FormatInt(userID, 10),
	)

	res, err := h.teamClient.GetAvailableStudents(ctx, &teamv1.GetAvailableStudentsRequest{
		UniversityId:  uniID,
		DepartmentId:  departmentID,
		ExcludeUserId: userID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
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

func (h *Handler) ListTeams(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")

	page := int32(1)
	pageSize := int32(20)
	if p := c.Query("page"); p != "" {
		if v, err := strconv.ParseInt(p, 10, 32); err == nil {
			page = int32(v)
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.ParseInt(ps, 10, 32); err == nil {
			pageSize = int32(v)
		}
	}

	res, err := h.teamClient.ListTeams(c.Request.Context(), &teamv1.ListTeamsRequest{
		DepartmentId: departmentID,
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) UpdateTeam(c *gin.Context) {
	teamID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	userID := c.GetInt64("userId")

	var reqBody struct {
		Name string `json:"name"`
	}
	if bindErr := c.ShouldBindJSON(&reqBody); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}
	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(userID, 10),
	)
	res, err := h.teamClient.UpdateTeam(ctx, &teamv1.UpdateTeamRequest{
		Team: &teamv1.Team{
			Id:   teamID,
			Name: reqBody.Name,
		},
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"name"},
		},
		RequesterId: userID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) DeleteTeam(c *gin.Context) {
	teamID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	userID := c.GetInt64("userId")
	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(userID, 10),
	)
	_, err = h.teamClient.DeleteTeam(ctx, &teamv1.DeleteTeamRequest{
		TeamId:      teamID,
		RequesterId: userID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) AddMember(c *gin.Context) {
	teamID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	var reqBody struct {
		UserID int64  `json:"user_id" binding:"required"`
		Role   string `json:"role"`
	}
	if bindErr := c.ShouldBindJSON(&reqBody); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	userID := c.GetInt64("userId")
	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(userID, 10),
	)

	res, err := h.teamClient.AddMember(ctx, &teamv1.AddMemberRequest{
		TeamId: teamID,
		UserId: reqBody.UserID,
		Role:   reqBody.Role,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) RemoveMember(c *gin.Context) {
	teamID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	memberID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	requesterID := c.GetInt64("userId")

	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(requesterID, 10),
	)

	_, err = h.teamClient.RemoveMember(ctx, &teamv1.RemoveMemberRequest{
		TeamId:      teamID,
		UserId:      memberID,
		RequesterId: requesterID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// LeaveTeam - студент выходит из команды
func (h *Handler) LeaveTeam(c *gin.Context) {
	teamID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	userID := c.GetInt64("userId")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	res, err := h.teamClient.LeaveTeam(c.Request.Context(), &teamv1.LeaveTeamRequest{
		TeamId: teamID,
		UserId: userID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       res.Success,
		"message":       res.Message,
		"team_deleted":  res.TeamDeleted,
		"new_leader_id": res.NewLeaderId,
	})
}

// TransferLeadership - лидер передаёт лидерство другому участнику
func (h *Handler) TransferLeadership(c *gin.Context) {
	teamID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	userID := c.GetInt64("userId")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var reqBody struct {
		NewLeaderID int64 `json:"new_leader_id" binding:"required"`
	}
	err = c.ShouldBindJSON(&reqBody)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.teamClient.TransferLeadership(c.Request.Context(), &teamv1.TransferLeadershipRequest{
		TeamId:          teamID,
		CurrentLeaderId: userID,
		NewLeaderId:     reqBody.NewLeaderID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      res.Success,
		"message":      res.Message,
		"updated_team": res.UpdatedTeam,
	})
}
func (h *Handler) JoinTeamByCode(c *gin.Context) {
	if c.GetString("role") != "student" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only students can join"})
		return
	}
	userID := c.GetInt64("userId")
	universityID := c.GetInt64("universityId")
	departmentID := c.GetInt64("departmentId")

	var req struct {
		Code string `json:"code" binding:"required,len=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(userID, 10),
		"x-user-role", c.GetString("role"),
		"x-university-id", strconv.FormatInt(universityID, 10),
		"x-department-id", strconv.FormatInt(departmentID, 10),
	)

	resp, err := h.teamClient.JoinTeamByCode(ctx, &teamv1.JoinTeamByCodeRequest{InviteCode: req.Code})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
func (h *Handler) RegenerateInviteCode(c *gin.Context) {
	teamID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid team id"})
		return
	}

	userID := c.GetInt64("userId")

	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(userID, 10),
	)

	resp, err := h.teamClient.RegenerateInviteCode(ctx, &teamv1.RegenerateInviteCodeRequest{TeamId: teamID})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
