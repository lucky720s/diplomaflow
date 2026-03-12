package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	"google.golang.org/grpc/metadata"
)

func parsePathID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + name})
		return 0, false
	}
	return id, true
}

func adminTechCtx(c *gin.Context) context.Context {
	// outgoingCtx уже прокидывает x-user-*, x-university-id, x-department-id, x-trace-id [[13]]
	ctx := outgoingCtx(c)

	uid := c.GetInt64("userId")
	role := c.GetString("role")
	univID := c.GetInt64("universityId")
	deptID := c.GetInt64("departmentId")
	traceID := c.GetString("trace_id") // FIX: было traceId [[13]]

	pairs := []string{
		"x-user-id", strconv.FormatInt(uid, 10),
		"x-user-role", role,
		"x-university-id", strconv.FormatInt(univID, 10),
		"x-department-id", strconv.FormatInt(deptID, 10),
		"x-internal-service", "api_gateway",
	}
	if traceID != "" {
		pairs = append(pairs, "x-trace-id", traceID)
	}

	return metadata.AppendToOutgoingContext(ctx, pairs...)
}

func (h *Handler) AdminTechListProjects(c *gin.Context) {
	// query params (всё опционально)
	var (
		deptID    int64
		teamID    int64
		studentID int64
		page      int32 = 1
		pageSize  int32 = 20
	)

	if v := strings.TrimSpace(c.Query("department_id")); v != "" {
		if x, err := strconv.ParseInt(v, 10, 64); err == nil && x > 0 {
			deptID = x
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department_id"})
			return
		}
	}
	if v := strings.TrimSpace(c.Query("team_id")); v != "" {
		if x, err := strconv.ParseInt(v, 10, 64); err == nil && x > 0 {
			teamID = x
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team_id"})
			return
		}
	}
	if v := strings.TrimSpace(c.Query("student_id")); v != "" {
		if x, err := strconv.ParseInt(v, 10, 64); err == nil && x > 0 {
			studentID = x
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student_id"})
			return
		}
	}
	if v := strings.TrimSpace(c.Query("page")); v != "" {
		if x, err := strconv.ParseInt(v, 10, 32); err == nil && x > 0 {
			page = int32(x)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
			return
		}
	}
	if v := strings.TrimSpace(c.Query("page_size")); v != "" {
		if x, err := strconv.ParseInt(v, 10, 32); err == nil && x > 0 {
			pageSize = int32(x)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page_size"})
			return
		}
	}

	resp, err := h.adminClient.ListProjectsAdmin(
		adminTechCtx(c),
		&adminv1.ListProjectsAdminRequest{
			DepartmentId: deptID, // если 0 — admin_service возьмёт из metadata [[12]]
			TeamId:       teamID,
			StudentId:    studentID,
			Status:       c.Query("status"),
			Q:            c.Query("q"),
			Page:         page,
			PageSize:     pageSize,
		},
	)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AdminTechGetProject(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}

	resp, err := h.adminClient.GetProjectAdmin(
		adminTechCtx(c),
		&adminv1.GetProjectAdminRequest{ProjectId: id},
	)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AdminTechCreateProject(c *gin.Context) {
	var req adminv1.CreateProjectAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// team-first: команда обязательна (дублируем ранней валидацией)
	if req.TeamId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team_id is required"})
		return
	}

	resp, err := h.adminClient.CreateProjectAdmin(adminTechCtx(c), &req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) AdminTechUpdateProject(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}

	var req adminv1.UpdateProjectAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ProjectId = id

	resp, err := h.adminClient.UpdateProjectAdmin(adminTechCtx(c), &req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AdminTechArchiveProject(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)

	resp, err := h.adminClient.ArchiveProjectAdmin(
		adminTechCtx(c),
		&adminv1.ArchiveProjectAdminRequest{
			ProjectId: id,
			Reason:    body.Reason,
		},
	)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AdminTechDeleteProject(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)

	resp, err := h.adminClient.DeleteArchivedProjectAdmin(
		adminTechCtx(c),
		&adminv1.DeleteArchivedProjectAdminRequest{
			ProjectId: id,
			Reason:    body.Reason,
		},
	)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
func (h *Handler) AdminTechListTeams(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")

	// admin может override department_id через query
	if c.GetString("role") == "admin" {
		if q := c.Query("department_id"); q != "" {
			if v, err := strconv.ParseInt(q, 10, 64); err == nil && v > 0 {
				departmentID = v
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department_id"})
				return
			}
		}
	}
	if departmentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "department_id is required"})
		return
	}

	page := int32(1)
	pageSize := int32(20)

	if p := c.Query("page"); p != "" {
		if v, err := strconv.ParseInt(p, 10, 32); err == nil && v > 0 {
			page = int32(v)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
			return
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.ParseInt(ps, 10, 32); err == nil && v > 0 {
			pageSize = int32(v)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page_size"})
			return
		}
	}

	resp, err := h.adminClient.ListAllTeams(adminTechCtx(c), &adminv1.ListAllTeamsRequest{
		DepartmentId: departmentID,
		Status:       c.Query("status"),
		Search:       c.Query("search"),
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AdminTechCreateTeam(c *gin.Context) {
	var body struct {
		Name         string  `json:"name" binding:"required"`
		UniversityID int64   `json:"university_id"`
		DepartmentID int64   `json:"department_id"`
		LeaderID     int64   `json:"leader_id" binding:"required"`
		MemberIDs    []int64 `json:"member_ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if body.UniversityID == 0 {
		body.UniversityID = c.GetInt64("universityId")
	}
	if body.DepartmentID == 0 {
		body.DepartmentID = c.GetInt64("departmentId")
	}

	if body.UniversityID <= 0 || body.DepartmentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "university_id and department_id are required"})
		return
	}
	if body.LeaderID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "leader_id is required"})
		return
	}

	resp, err := h.adminClient.CreateTeamAdmin(adminTechCtx(c), &adminv1.CreateTeamAdminRequest{
		Name:         body.Name,
		UniversityId: body.UniversityID,
		DepartmentId: body.DepartmentID,
		LeaderId:     body.LeaderID,
		MemberIds:    body.MemberIDs,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) AdminTechGetTeam(c *gin.Context) {
	teamID, ok := parsePathID(c, "id")
	if !ok {
		return
	}

	resp, err := h.adminClient.GetTeamDetails(adminTechCtx(c), &adminv1.GetTeamDetailsRequest{
		TeamId: teamID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AdminTechUpdateTeam(c *gin.Context) {
	teamID, ok := parsePathID(c, "id")
	if !ok {
		return
	}

	var body struct {
		Name         string  `json:"name"`
		SupervisorID int64   `json:"supervisor_id"`
		MemberIDs    []int64 `json:"member_ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.adminClient.UpdateTeamAdmin(adminTechCtx(c), &adminv1.UpdateTeamAdminRequest{
		TeamId:       teamID,
		Name:         body.Name,
		SupervisorId: body.SupervisorID,
		MemberIds:    body.MemberIDs,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AdminTechDeleteTeam(c *gin.Context) {
	teamID, ok := parsePathID(c, "id")
	if !ok {
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)

	_, err := h.adminClient.DeleteTeamAdmin(adminTechCtx(c), &adminv1.DeleteTeamAdminRequest{
		TeamId: teamID,
		Reason: body.Reason,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) AdminTechListTeamProjects(c *gin.Context) {
	teamID, ok := parsePathID(c, "id")
	if !ok {
		return
	}

	page := int32(1)
	pageSize := int32(20)
	if p := c.Query("page"); p != "" {
		if v, err := strconv.ParseInt(p, 10, 32); err == nil && v > 0 {
			page = int32(v)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
			return
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.ParseInt(ps, 10, 32); err == nil && v > 0 {
			pageSize = int32(v)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page_size"})
			return
		}
	}

	resp, err := h.adminClient.ListProjectsAdmin(adminTechCtx(c), &adminv1.ListProjectsAdminRequest{
		TeamId:   teamID,
		Page:     page,
		PageSize: pageSize,
		Status:   c.Query("status"),
		Q:        c.Query("q"),
		// DepartmentId можно не передавать: admin_service возьмёт из metadata, если 0 [[15]]
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
