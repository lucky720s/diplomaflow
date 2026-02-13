package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/metadata"
)

func outgoingCtx(c *gin.Context) context.Context {
	userID := c.GetInt64("userId")
	role := c.GetString("role")
	universityID := c.GetInt64("universityId")
	departmentID := c.GetInt64("departmentId")

	// Всегда пробрасываем всё: team_service местами требует univ/dept (requireAuth) [[12]]
	return metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(userID, 10),
		"x-user-role", role,
		"x-university-id", strconv.FormatInt(universityID, 10),
		"x-department-id", strconv.FormatInt(departmentID, 10),
	)
}

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

	unique := make(map[int64]bool)
	var cleanMemberIDs []int64
	for _, id := range reqBody.MemberIDs {
		if id == leaderID {
			continue
		}
		if !unique[id] {
			unique[id] = true
			cleanMemberIDs = append(cleanMemberIDs, id)
		}
	}

	ctx := outgoingCtx(c)

	res, err := h.teamClient.CreateTeam(ctx, &teamv1.CreateTeamRequest{
		Name:      reqBody.Name,
		MemberIds: cleanMemberIDs,
		LeaderId:  leaderID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) GetTeam(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	// GetTeam в team_service не требует requireAuth, но metadata не мешает
	res, err := h.teamClient.GetTeam(outgoingCtx(c), &teamv1.GetTeamRequest{TeamId: id})
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

	ctx := outgoingCtx(c)

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
	ctx := outgoingCtx(c)

	res, err := h.teamClient.GetMyInvites(ctx, &teamv1.GetMyInvitesRequest{UserId: userID})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) RespondToInvite(c *gin.Context) {
	userID := c.GetInt64("userId")
	inviteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || inviteID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invite id"})
		return
	}

	var jsonReq struct {
		Accept bool `json:"accept"`
	}
	if bindErr := c.ShouldBindJSON(&jsonReq); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	ctx := outgoingCtx(c)
	_, err = h.teamClient.RespondToInvite(ctx, &teamv1.RespondToInviteRequest{
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

	res, err := h.teamClient.GetMyTeam(outgoingCtx(c), &teamv1.GetMyTeamRequest{UserId: userID})
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
		if v, err := strconv.ParseInt(p, 10, 32); err == nil && v > 0 {
			page = int32(v)
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.ParseInt(ps, 10, 32); err == nil && v > 0 {
			pageSize = int32(v)
		}
	}

	res, err := h.teamClient.ListTeams(outgoingCtx(c), &teamv1.ListTeamsRequest{
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
	if err != nil || teamID <= 0 {
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

	ctx := outgoingCtx(c)
	res, err := h.teamClient.UpdateTeam(ctx, &teamv1.UpdateTeamRequest{
		Team: &teamv1.Team{Id: teamID, Name: reqBody.Name},
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
	if err != nil || teamID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	userID := c.GetInt64("userId")

	_, err = h.teamClient.DeleteTeam(outgoingCtx(c), &teamv1.DeleteTeamRequest{
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
	if err != nil || teamID <= 0 {
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

	res, err := h.teamClient.AddMember(outgoingCtx(c), &teamv1.AddMemberRequest{
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
	if err != nil || teamID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	memberID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || memberID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	requesterID := c.GetInt64("userId")

	_, err = h.teamClient.RemoveMember(outgoingCtx(c), &teamv1.RemoveMemberRequest{
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

func (h *Handler) LeaveTeam(c *gin.Context) {
	teamID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || teamID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	userID := c.GetInt64("userId")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	res, err := h.teamClient.LeaveTeam(outgoingCtx(c), &teamv1.LeaveTeamRequest{
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

func (h *Handler) TransferLeadership(c *gin.Context) {
	teamID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || teamID <= 0 {
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
	if bindErr := c.ShouldBindJSON(&reqBody); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	res, err := h.teamClient.TransferLeadership(outgoingCtx(c), &teamv1.TransferLeadershipRequest{
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

	var req struct {
		Code string `json:"code" binding:"required,len=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.teamClient.JoinTeamByCode(outgoingCtx(c), &teamv1.JoinTeamByCodeRequest{
		InviteCode: req.Code,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) RegenerateInviteCode(c *gin.Context) {
	teamID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || teamID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	resp, err := h.teamClient.RegenerateInviteCode(outgoingCtx(c), &teamv1.RegenerateInviteCodeRequest{
		TeamId: teamID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
