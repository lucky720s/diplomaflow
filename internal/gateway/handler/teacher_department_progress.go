package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
)

// Teacher-facing, read-only department-progress endpoints.
//
// Access is NOT a base-role check. `department_progress_viewer` is a dynamic
// department role (roles + user_role_assignments), so we verify it live against
// the auth service for the EXACT requested department.

const departmentProgressViewerSlug = "department_progress_viewer"

var dpAllowedSortBy = map[string]struct{}{
	"":                    {},
	"team_name":           {},
	"project_title":       {},
	"current_stage":       {},
	"progress_percentage": {},
	"last_activity_at":    {},
	"final_grade":         {},
	"admission_status":    {},
}

var dpAllowedSortOrder = map[string]struct{}{
	"":     {},
	"asc":  {},
	"desc": {},
}

var dpAllowedAdmissionStatus = map[string]struct{}{
	"":                  {},
	"not_decided":       {},
	"admitted":          {},
	"not_admitted":      {},
	"revision_required": {},
}

// dpResolveDepartmentStrict picks requested department.
// Explicit ?department_id wins, but invalid explicit value is a real 400.
// If query is absent, fallback to JWT departmentId is allowed.
func dpResolveDepartmentStrict(c *gin.Context) (int64, bool) {
	if q := strings.TrimSpace(c.Query("department_id")); q != "" {
		id, err := strconv.ParseInt(q, 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department_id"})
			return 0, false
		}
		return id, true
	}

	id := c.GetInt64("departmentId")
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "department_id is required"})
		return 0, false
	}
	return id, true
}

func parseOptionalInt64Query(c *gin.Context, name string) (int64, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return 0, true
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + name})
		return 0, false
	}
	return v, true
}

func parsePositiveInt32Query(c *gin.Context, name string, def int32, max int32) (int32, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return def, true
	}
	v, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || v <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + name})
		return 0, false
	}
	out := int32(v)
	if max > 0 && out > max {
		out = max
	}
	return out, true
}

func parseGradeQuery(c *gin.Context, name string) (int32, bool, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return 0, false, true
	}
	v, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || v < 0 || v > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + name})
		return 0, false, false
	}
	return int32(v), true, true
}

func validateDPSort(c *gin.Context, sortBy, sortOrder string) bool {
	if _, ok := dpAllowedSortBy[strings.TrimSpace(sortBy)]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sort_by"})
		return false
	}
	if _, ok := dpAllowedSortOrder[strings.ToLower(strings.TrimSpace(sortOrder))]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sort_order"})
		return false
	}
	return true
}

func validateDPAdmissionStatus(c *gin.Context, status string) bool {
	if _, ok := dpAllowedAdmissionStatus[strings.TrimSpace(status)]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid admission_status"})
		return false
	}
	return true
}

// requireDepartmentProgressAccess enforces read access to department-wide progress.
func (h *Handler) requireDepartmentProgressAccess(c *gin.Context, departmentID int64) bool {
	role := c.GetString("role")
	if role == "admin" {
		return true
	}
	if role != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return false
	}

	userID := c.GetInt64("userId")
	if userID <= 0 || departmentID <= 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return false
	}

	resp, err := h.authClient.ListUserDepartmentRoles(authIAMCtx(c), &authv1.ListUserDepartmentRolesRequest{
		UserId:       userID,
		DepartmentId: departmentID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return false
	}

	for _, slug := range resp.GetSlugs() {
		if slug == departmentProgressViewerSlug {
			return true
		}
	}

	c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	return false
}

// GET /api/v1/teacher/department-progress/summary
func (h *Handler) TeacherDPSummary(c *gin.Context) {
	deptID, ok := dpResolveDepartmentStrict(c)
	if !ok {
		return
	}
	if !h.requireDepartmentProgressAccess(c, deptID) {
		return
	}

	resp, err := h.adminClient.GetDepartmentProgressSummary(adminPanelCtx(c), &adminv1.DepartmentProgressSummaryRequest{
		DepartmentId: deptID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GET /api/v1/teacher/department-progress/teams
func (h *Handler) TeacherDPListTeams(c *gin.Context) {
	deptID, ok := dpResolveDepartmentStrict(c)
	if !ok {
		return
	}
	if !h.requireDepartmentProgressAccess(c, deptID) {
		return
	}

	sortBy := strings.TrimSpace(c.Query("sort_by"))
	sortOrder := strings.ToLower(strings.TrimSpace(c.Query("sort_order")))
	admissionStatus := strings.TrimSpace(c.Query("admission_status"))

	if !validateDPSort(c, sortBy, sortOrder) {
		return
	}
	if !validateDPAdmissionStatus(c, admissionStatus) {
		return
	}

	supervisorID, ok := parseOptionalInt64Query(c, "supervisor_id")
	if !ok {
		return
	}

	page, ok := parsePositiveInt32Query(c, "page", 1, 0)
	if !ok {
		return
	}

	pageSize, ok := parsePositiveInt32Query(c, "page_size", 20, 100)
	if !ok {
		return
	}

	minGrade, hasMinGrade, ok := parseGradeQuery(c, "min_grade")
	if !ok {
		return
	}

	maxGrade, hasMaxGrade, ok := parseGradeQuery(c, "max_grade")
	if !ok {
		return
	}

	if hasMinGrade && hasMaxGrade && minGrade > maxGrade {
		c.JSON(http.StatusBadRequest, gin.H{"error": "min_grade cannot be greater than max_grade"})
		return
	}

	req := &adminv1.ListDepartmentProgressTeamsRequest{
		DepartmentId:      deptID,
		Search:            strings.TrimSpace(c.Query("search")),
		SupervisorId:      supervisorID,
		CurrentStage:      strings.TrimSpace(c.Query("current_stage")),
		TopicStatus:       strings.TrimSpace(c.Query("topic_status")),
		NormControlStatus: strings.TrimSpace(c.Query("norm_control_status")),
		AntiplagiatStatus: strings.TrimSpace(c.Query("antiplagiat_status")),
		PreDefenseStatus:  strings.TrimSpace(c.Query("pre_defense_status")),
		AdmissionStatus:   admissionStatus,
		Page:              page,
		PageSize:          pageSize,
		SortBy:            sortBy,
		SortOrder:         sortOrder,
	}

	if hasMinGrade {
		req.MinGrade = minGrade
		req.HasMinGrade = true
	}
	if hasMaxGrade {
		req.MaxGrade = maxGrade
		req.HasMaxGrade = true
	}

	resp, err := h.adminClient.ListDepartmentProgressTeams(adminPanelCtx(c), req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GET /api/v1/teacher/department-progress/teams/:team_id
func (h *Handler) TeacherDPTeamDetails(c *gin.Context) {
	deptID, ok := dpResolveDepartmentStrict(c)
	if !ok {
		return
	}
	if !h.requireDepartmentProgressAccess(c, deptID) {
		return
	}

	teamID, err := strconv.ParseInt(c.Param("team_id"), 10, 64)
	if err != nil || teamID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team_id"})
		return
	}

	resp, err := h.adminClient.GetDepartmentProgressTeamDetails(adminPanelCtx(c), &adminv1.GetDepartmentProgressTeamDetailsRequest{
		TeamId:       teamID,
		DepartmentId: deptID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
