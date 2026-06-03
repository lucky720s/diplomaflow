package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/metadata"
)

func outgoingCtx(c *gin.Context) context.Context {
	userID := c.GetInt64("userId")
	role := c.GetString("role")
	universityID := c.GetInt64("universityId")
	departmentID := c.GetInt64("departmentId")
	traceID := c.GetString("trace_id")

	// Всегда пробрасываем всё: team_service местами требует univ/dept (requireAuth) [[12]]
	return metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(userID, 10),
		"x-user-role", role,
		"x-university-id", strconv.FormatInt(universityID, 10),
		"x-department-id", strconv.FormatInt(departmentID, 10),
		"x-trace-id", traceID,
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

	resp, err := h.teamClient.RespondToInvite(outgoingCtx(c), &teamv1.RespondToInviteRequest{
		InviteId: inviteID,
		UserId:   userID,
		Accept:   jsonReq.Accept,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": resp.Message,
	})
}
func (h *Handler) GetMyTeam(c *gin.Context) {
	userID := c.GetInt64("userId")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	teamResp, err := h.teamClient.GetMyTeam(outgoingCtx(c), &teamv1.GetMyTeamRequest{UserId: userID})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	if teamResp == nil || !teamResp.HasTeam || teamResp.Team == nil || teamResp.Team.TeamId == 0 {
		c.JSON(http.StatusOK, gin.H{"has_team": false})
		return
	}

	team := teamResp.Team
	teamID := team.TeamId

	ids := make([]int64, 0, len(team.Members))
	seen := map[int64]struct{}{}
	leaderID := int64(0)

	for _, m := range team.Members {
		if m == nil || m.UserId == 0 {
			continue
		}
		if m.Role == "leader" {
			leaderID = m.UserId
		}
		if _, ok := seen[m.UserId]; ok {
			continue
		}
		seen[m.UserId] = struct{}{}
		ids = append(ids, m.UserId)
	}
	if leaderID == 0 {
		leaderID = userID
	}
	authCtx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-internal-service", "team_service")
	au, _ := h.authClient.BatchGetUserPreviews(authCtx, &authv1.BatchGetUserPreviewsRequest{Ids: ids})

	users := map[int64]*authv1.UserPreview{}
	if au != nil {
		for _, u := range au.Users {
			if u != nil && u.Id != 0 {
				users[u.Id] = u
			}
		}
	}

	pbMembers := make([]gin.H, 0, len(team.Members))
	for _, m := range team.Members {
		if m == nil {
			continue
		}

		first := m.FirstName
		last := m.LastName
		email := m.Email

		if u := users[m.UserId]; u != nil {
			if u.FirstName != "" {
				first = u.FirstName
			}
			if u.LastName != "" {
				last = u.LastName
			}
			if u.Email != "" {
				email = u.Email
			}
		}

		full := (first + " " + last)
		if full == " " {
			full = ""
		}

		pbMembers = append(pbMembers, gin.H{
			"user_id":    m.UserId,
			"role":       m.Role,
			"first_name": first,
			"last_name":  last,
			"email":      email,
			"full_name":  full,
			"fullName":   full,
		})
	}

	var supervisorInfo gin.H = nil
	var supervisorID int64 = 0
	supResp, supErr := h.adminClient.ListSupervisorRequests(outgoingCtx(c), &adminv1.ListSupervisorRequestsReq{
		DepartmentId: c.GetInt64("departmentId"),
		Status:       "approved",
		TeamId:       teamID,
		Page:         1,
		PageSize:     1,
	})

	if supErr == nil && supResp != nil && len(supResp.Requests) > 0 && supResp.Requests[0] != nil {
		r := supResp.Requests[0]
		supervisorID = r.SupervisorId
		supervisorInfo = gin.H{
			"id":        r.SupervisorId,
			"full_name": r.SupervisorName,
			"fullName":  r.SupervisorName,
			"email":     r.SupervisorEmail,
			"position":  "Scientific Supervisor",
		}
	}
	if supervisorID == 0 {
		det, detErr := h.adminClient.GetTeamDetails(outgoingCtx(c), &adminv1.GetTeamDetailsRequest{TeamId: teamID})
		if detErr == nil && det != nil && det.Team != nil && det.Team.Supervisor != nil && det.Team.Supervisor.Id != 0 {
			s := det.Team.Supervisor
			supervisorID = s.Id
			supervisorInfo = gin.H{
				"id":        s.Id,
				"full_name": s.FullName,
				"fullName":  s.FullName,
				"email":     s.Email,
				"position":  s.Position,
			}
		}
	}
	var projectID int64 = 0

	candidates := make([]int64, 0, len(ids)+2)
	addCand := func(v int64) {
		if v <= 0 {
			return
		}
		for _, x := range candidates {
			if x == v {
				return
			}
		}
		candidates = append(candidates, v)
	}

	addCand(leaderID)
	addCand(userID)
	for _, id := range ids {
		addCand(id)
	}

	for _, sid := range candidates {
		pr, perr := h.projectClient.GetStudentProjects(outgoingCtx(c), &projectv1.GetStudentProjectsRequest{
			StudentId: sid,
		})
		if perr != nil || pr == nil {
			continue
		}
		for _, p := range pr.Projects {
			if p != nil && p.TeamId == teamID {
				projectID = p.ProjectId
				break
			}
		}
		if projectID != 0 {
			break
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"has_team": true,
		"team": gin.H{
			"team_id":               team.TeamId,
			"name":                  team.Name,
			"role":                  team.Role,
			"members":               pbMembers,
			"member_count":          team.MemberCount,
			"pending_invites_count": team.PendingInvitesCount,
			"invite_code":           team.InviteCode,
			"composition_locked":    team.CompositionLocked,
			"supervisor":            supervisorInfo,
			"supervisor_id":         supervisorID,
			"project_id":            projectID,
		},
	})
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
