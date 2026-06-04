package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"google.golang.org/grpc/metadata"
)

// Supervisor-facing project endpoints (/api/v1/supervisors/projects/*).
//
// Access model: supervisor_id is ALWAYS taken from the JWT (c.GetInt64("userId")),
// never from the client. The admin service enforces ownership and returns
// NotFound (404) / PermissionDenied (403), which MapGRPCError translates. The
// caller_role is forwarded so the admin service can grant an admin bypass.

// supervisorProjectIDParam parses and validates the :project_id path param.
func supervisorProjectIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("project_id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return 0, false
	}
	return id, true
}

// requireSupervisorProjectAccess is the gateway-side guard used by endpoints that
// are served by another service (workflow). It asks the admin service whether the
// JWT user owns the project, mapping missing -> 404 and not-owned -> 403.
func (h *Handler) requireSupervisorProjectAccess(c *gin.Context, projectID int64) bool {
	role := c.GetString("role")
	resp, err := h.adminClient.CheckSupervisorProjectAccess(adminPanelCtx(c), &adminv1.CheckSupervisorProjectAccessRequest{
		SupervisorId: c.GetInt64("userId"),
		ProjectId:    projectID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return false
	}
	if !resp.GetExists() {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return false
	}
	if role == "admin" {
		return true
	}
	if !resp.GetHasAccess() {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not the supervisor of this project"})
		return false
	}
	return true
}

// GET /api/v1/supervisors/projects
func (h *Handler) ListSupervisorProjects(c *gin.Context) {
	page, ok := parsePositiveInt32Query(c, "page", 1, 0)
	if !ok {
		return
	}
	pageSize, ok := parsePositiveInt32Query(c, "page_size", 20, 100)
	if !ok {
		return
	}

	resp, err := h.adminClient.ListSupervisorProjects(adminPanelCtx(c), &adminv1.ListSupervisorProjectsRequest{
		SupervisorId: c.GetInt64("userId"),
		Status:       strings.TrimSpace(c.Query("status")),
		CurrentState: strings.TrimSpace(c.Query("current_state")),
		Search:       strings.TrimSpace(c.Query("search")),
		Page:         page,
		PageSize:     pageSize,
		Sort:         strings.TrimSpace(c.Query("sort")),
		Order:        strings.TrimSpace(c.Query("order")),
		CallerRole:   c.GetString("role"),
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	writeProtoJSON(c, http.StatusOK, resp)
}

// GET /api/v1/supervisors/projects/:project_id
func (h *Handler) GetSupervisorProjectDetails(c *gin.Context) {
	projectID, ok := supervisorProjectIDParam(c)
	if !ok {
		return
	}
	resp, err := h.adminClient.GetSupervisorProjectDetails(adminPanelCtx(c), &adminv1.GetSupervisorProjectDetailsRequest{
		SupervisorId: c.GetInt64("userId"),
		ProjectId:    projectID,
		CallerRole:   c.GetString("role"),
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	writeProtoJSON(c, http.StatusOK, resp)
}

// GET /api/v1/supervisors/projects/:project_id/submissions
func (h *Handler) ListSupervisorProjectSubmissions(c *gin.Context) {
	projectID, ok := supervisorProjectIDParam(c)
	if !ok {
		return
	}
	stepID, ok := parseOptionalInt64Query(c, "step_id")
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

	resp, err := h.adminClient.ListSupervisorProjectSubmissions(adminPanelCtx(c), &adminv1.ListSupervisorProjectSubmissionsRequest{
		SupervisorId: c.GetInt64("userId"),
		ProjectId:    projectID,
		StepId:       stepID,
		Status:       strings.TrimSpace(c.Query("status")),
		Page:         page,
		PageSize:     pageSize,
		CallerRole:   c.GetString("role"),
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	writeProtoJSON(c, http.StatusOK, resp)
}

// GET /api/v1/supervisors/projects/:project_id/submissions/:submission_id
func (h *Handler) GetSupervisorProjectSubmission(c *gin.Context) {
	projectID, ok := supervisorProjectIDParam(c)
	if !ok {
		return
	}
	submissionID := strings.TrimSpace(c.Param("submission_id"))
	if submissionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid submission_id"})
		return
	}

	resp, err := h.adminClient.GetSupervisorProjectSubmission(adminPanelCtx(c), &adminv1.GetSupervisorProjectSubmissionRequest{
		SupervisorId: c.GetInt64("userId"),
		ProjectId:    projectID,
		SubmissionId: submissionID,
		CallerRole:   c.GetString("role"),
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	writeProtoJSON(c, http.StatusOK, resp)
}

// GET /api/v1/supervisors/projects/:project_id/grades
func (h *Handler) GetSupervisorProjectGrades(c *gin.Context) {
	projectID, ok := supervisorProjectIDParam(c)
	if !ok {
		return
	}
	resp, err := h.adminClient.GetSupervisorProjectGrades(adminPanelCtx(c), &adminv1.GetSupervisorProjectGradesRequest{
		SupervisorId: c.GetInt64("userId"),
		ProjectId:    projectID,
		CallerRole:   c.GetString("role"),
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	writeProtoJSON(c, http.StatusOK, resp)
}

// GET /api/v1/supervisors/projects/:project_id/grading-history
func (h *Handler) GetSupervisorProjectGradingHistory(c *gin.Context) {
	projectID, ok := supervisorProjectIDParam(c)
	if !ok {
		return
	}
	stepID, ok := parseOptionalInt64Query(c, "step_id")
	if !ok {
		return
	}
	resp, err := h.adminClient.GetSupervisorProjectGradingHistory(adminPanelCtx(c), &adminv1.GetSupervisorProjectGradingHistoryRequest{
		SupervisorId: c.GetInt64("userId"),
		ProjectId:    projectID,
		StepId:       stepID,
		CallerRole:   c.GetString("role"),
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	writeProtoJSON(c, http.StatusOK, resp)
}

// GET /api/v1/supervisors/projects/:project_id/workflow-history
func (h *Handler) GetSupervisorProjectWorkflowHistory(c *gin.Context) {
	projectID, ok := supervisorProjectIDParam(c)
	if !ok {
		return
	}
	resp, err := h.adminClient.GetSupervisorProjectWorkflowHistory(adminPanelCtx(c), &adminv1.GetSupervisorProjectWorkflowHistoryRequest{
		SupervisorId: c.GetInt64("userId"),
		ProjectId:    projectID,
		CallerRole:   c.GetString("role"),
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	writeProtoJSON(c, http.StatusOK, resp)
}

// GET /api/v1/supervisors/projects/:project_id/files
func (h *Handler) ListSupervisorProjectFiles(c *gin.Context) {
	projectID, ok := supervisorProjectIDParam(c)
	if !ok {
		return
	}
	resp, err := h.adminClient.ListSupervisorProjectFiles(adminPanelCtx(c), &adminv1.ListSupervisorProjectFilesRequest{
		SupervisorId: c.GetInt64("userId"),
		ProjectId:    projectID,
		CallerRole:   c.GetString("role"),
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	writeProtoJSON(c, http.StatusOK, resp)
}

// GET /api/v1/supervisors/projects/:project_id/timeline
func (h *Handler) GetSupervisorProjectTimeline(c *gin.Context) {
	projectID, ok := supervisorProjectIDParam(c)
	if !ok {
		return
	}
	resp, err := h.adminClient.GetSupervisorProjectTimeline(adminPanelCtx(c), &adminv1.GetSupervisorProjectTimelineRequest{
		SupervisorId: c.GetInt64("userId"),
		ProjectId:    projectID,
		CallerRole:   c.GetString("role"),
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	writeProtoJSON(c, http.StatusOK, resp)
}

// POST /api/v1/supervisors/projects/:project_id/submissions/:submission_id/review
func (h *Handler) ReviewSupervisorProjectSubmission(c *gin.Context) {
	projectID, ok := supervisorProjectIDParam(c)
	if !ok {
		return
	}
	submissionID := strings.TrimSpace(c.Param("submission_id"))
	if submissionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid submission_id"})
		return
	}

	var body struct {
		Action   string `json:"action"`   // approve | reject | request_changes
		Decision string `json:"decision"` // approved | rejected | revision_requested (alias)
		Comment  string `json:"comment"`
		Grade    int32  `json:"grade"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	action := normalizeReviewAction(body.Action, body.Decision)
	if action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be approve, reject or request_changes"})
		return
	}

	resp, err := h.adminClient.ReviewSupervisorProjectSubmission(adminPanelCtx(c), &adminv1.ReviewSupervisorProjectSubmissionRequest{
		SupervisorId: c.GetInt64("userId"),
		ProjectId:    projectID,
		SubmissionId: submissionID,
		Action:       action,
		Comment:      body.Comment,
		Grade:        body.Grade,
		CallerRole:   c.GetString("role"),
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	writeProtoJSON(c, http.StatusOK, resp)
}

// normalizeReviewAction maps the {action|decision} aliases to the canonical action.
func normalizeReviewAction(action, decision string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "approve", "reject", "request_changes":
		return strings.ToLower(strings.TrimSpace(action))
	}
	switch strings.ToLower(strings.TrimSpace(decision)) {
	case "approved", "approve":
		return "approve"
	case "rejected", "reject":
		return "reject"
	case "revision_requested", "request_changes", "changes_requested":
		return "request_changes"
	}
	return ""
}

// POST /api/v1/supervisors/projects/:project_id/states/:state_id/review
//
// State-level review/grade is a workflow concern. We enforce supervisor ownership
// via the admin service, then delegate to the workflow service's SubmitReview,
// which itself validates that the user's role is allowed by the state's
// review_config.reviewer_roles (so commission/norm_control-only states are rejected).
func (h *Handler) SubmitSupervisorStateReview(c *gin.Context) {
	projectID, ok := supervisorProjectIDParam(c)
	if !ok {
		return
	}
	stateID, err := strconv.ParseInt(c.Param("state_id"), 10, 64)
	if err != nil || stateID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state_id"})
		return
	}

	if !h.requireSupervisorProjectAccess(c, projectID) {
		return
	}

	var body struct {
		Decision string `json:"decision"` // approved | rejected (admission)
		Score    *int32 `json:"score"`    // 0-100 (score)
		Comment  string `json:"comment"`
	}
	if bindErr := c.ShouldBindJSON(&body); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	userID := c.GetInt64("userId")
	roleSlug := c.GetString("role")

	req := &workflowv1.SubmitReviewRequest{
		ProjectId: projectID,
		StateId:   stateID,
		UserId:    userID,
		RoleSlug:  roleSlug,
		Decision:  body.Decision,
		Comment:   body.Comment,
	}
	if body.Score != nil {
		req.Score = *body.Score
		req.HasScore = true
	}

	reviewCtx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(userID, 10),
		"x-user-role", roleSlug,
	)

	resp, err := h.workflowClient.SubmitReview(reviewCtx, req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
