package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
)

// ==================== Dashboard ====================

func (h *Handler) GetAdminDashboard(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")
	if departmentID == 0 {
		// Попробуем из query
		if did := c.Query("department_id"); did != "" {
			departmentID, _ = strconv.ParseInt(did, 10, 64)
		}
	}

	resp, err := h.adminClient.GetDashboard(c.Request.Context(), &adminv1.GetDashboardRequest{
		DepartmentId: departmentID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetDepartmentStats(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")
	resp, err := h.adminClient.GetDepartmentStats(c.Request.Context(), &adminv1.GetDepartmentStatsRequest{
		DepartmentId: departmentID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ==================== Students ====================

func (h *Handler) AdminListStudents(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")
	universityID := c.GetInt64("universityId")

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

	resp, err := h.adminClient.ListStudents(c.Request.Context(), &adminv1.ListStudentsRequest{
		DepartmentId:    departmentID,
		UniversityId:    universityID,
		Search:          c.Query("search"),
		OnlyWithoutTeam: c.Query("without_team") == "true",
		Page:            page,
		PageSize:        pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AdminGetStudent(c *gin.Context) {
	studentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	resp, err := h.adminClient.GetStudent(c.Request.Context(), &adminv1.GetStudentRequest{
		StudentId: studentID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ==================== Teams ====================

func (h *Handler) AdminListTeams(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")

	page := int32(1)
	pageSize := int32(20)

	if p := c.Query("page"); p != "" {
		if v, _ := strconv.ParseInt(p, 10, 32); v > 0 {
			page = int32(v)
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, _ := strconv.ParseInt(ps, 10, 32); v > 0 {
			pageSize = int32(v)
		}
	}

	resp, err := h.adminClient.ListAllTeams(c.Request.Context(), &adminv1.ListAllTeamsRequest{
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

func (h *Handler) AdminGetTeamDetails(c *gin.Context) {
	teamID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	resp, err := h.adminClient.GetTeamDetails(c.Request.Context(), &adminv1.GetTeamDetailsRequest{
		TeamId: teamID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AdminUpdateTeam(c *gin.Context) {
	teamID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	var req struct {
		Name         string  `json:"name"`
		SupervisorID int64   `json:"supervisor_id"`
		MemberIDs    []int64 `json:"member_ids"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	resp, err := h.adminClient.UpdateTeamAdmin(c.Request.Context(), &adminv1.UpdateTeamAdminRequest{
		TeamId:       teamID,
		Name:         req.Name,
		SupervisorId: req.SupervisorID,
		MemberIds:    req.MemberIDs,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AdminDeleteTeam(c *gin.Context) {
	teamID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	_, err = h.adminClient.DeleteTeamAdmin(c.Request.Context(), &adminv1.DeleteTeamAdminRequest{
		TeamId: teamID,
		Reason: req.Reason,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ==================== Supervisors ====================

func (h *Handler) ListSupervisors(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")
	universityID := c.GetInt64("universityId")

	page := int32(1)
	pageSize := int32(20)

	resp, err := h.adminClient.ListSupervisors(c.Request.Context(), &adminv1.ListSupervisorsRequest{
		DepartmentId: departmentID,
		UniversityId: universityID,
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AssignSupervisor(c *gin.Context) {
	var req struct {
		TeamID       int64 `json:"team_id" binding:"required"`
		SupervisorID int64 `json:"supervisor_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.adminClient.AssignSupervisor(c.Request.Context(), &adminv1.AssignSupervisorRequest{
		TeamId:       req.TeamID,
		SupervisorId: req.SupervisorID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ==================== Topic Registration (PROJECT-FIRST) ====================

// POST /api/v1/projects/:id/topic-registration
func (h *Handler) SubmitTopicRegistration(c *gin.Context) {
	userID := c.GetInt64("userId")

	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || projectID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	var req struct {
		ProposedTopic    string `json:"proposed_topic" binding:"required,min=5"`
		TopicDescription string `json:"topic_description"`
		SupervisorID     int64  `json:"supervisor_id" binding:"required"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	resp, err := h.adminClient.SubmitTopicRegistration(c.Request.Context(), &adminv1.SubmitTopicRegistrationRequest{
		ProjectId:        projectID,
		TeamId:           0, // optional; admin_service resolves via project runtime
		ProposedTopic:    req.ProposedTopic,
		TopicDescription: req.TopicDescription,
		SupervisorId:     req.SupervisorID,
		SubmittedBy:      userID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) ListTopicRegistrations(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")
	status := c.Query("status")

	page := int32(1)
	pageSize := int32(20)
	if p := c.Query("page"); p != "" {
		if v, _ := strconv.ParseInt(p, 10, 32); v > 0 {
			page = int32(v)
		}
	}

	resp, err := h.adminClient.ListTopicRegistrations(c.Request.Context(), &adminv1.ListTopicRegistrationsRequest{
		DepartmentId: departmentID,
		Status:       status,
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetTopicRegistration(c *gin.Context) {
	registrationID := c.Param("id")

	resp, err := h.adminClient.GetTopicRegistration(c.Request.Context(), &adminv1.GetTopicRegistrationRequest{
		RegistrationId: registrationID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ReviewTopicRegistration(c *gin.Context) {
	registrationID := c.Param("id")
	reviewerID := c.GetInt64("userId")

	var req struct {
		Action          string `json:"action" binding:"required"` // approve, reject, request_changes
		Comment         string `json:"comment"`
		RejectionReason string `json:"rejection_reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.adminClient.ReviewTopicRegistration(c.Request.Context(), &adminv1.ReviewTopicRegistrationRequest{
		RegistrationId:  registrationID,
		ReviewerId:      reviewerID,
		Action:          req.Action,
		Comment:         req.Comment,
		RejectionReason: req.RejectionReason,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ==================== Submissions ====================

func (h *Handler) ListSubmissions(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")

	var stepID, teamID int64
	if s := c.Query("step_id"); s != "" {
		stepID, _ = strconv.ParseInt(s, 10, 64)
	}
	if t := c.Query("team_id"); t != "" {
		teamID, _ = strconv.ParseInt(t, 10, 64)
	}

	page := int32(1)
	pageSize := int32(20)
	if p := c.Query("page"); p != "" {
		if v, _ := strconv.ParseInt(p, 10, 32); v > 0 {
			page = int32(v)
		}
	}

	resp, err := h.adminClient.ListSubmissions(c.Request.Context(), &adminv1.ListSubmissionsRequest{
		DepartmentId: departmentID,
		StepId:       stepID,
		TeamId:       teamID,
		Status:       c.Query("status"),
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetSubmission(c *gin.Context) {
	submissionID := c.Param("id")

	resp, err := h.adminClient.GetSubmission(c.Request.Context(), &adminv1.GetSubmissionRequest{
		SubmissionId: submissionID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ReviewSubmission(c *gin.Context) {
	submissionID := c.Param("id")
	reviewerID := c.GetInt64("userId")

	var req struct {
		Action  string `json:"action" binding:"required"` // approve, reject, request_changes
		Comment string `json:"comment"`
		Grade   int32  `json:"grade"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.adminClient.ReviewSubmission(c.Request.Context(), &adminv1.ReviewSubmissionRequest{
		SubmissionId: submissionID,
		ReviewerId:   reviewerID,
		Action:       req.Action,
		Comment:      req.Comment,
		Grade:        req.Grade,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ==================== Grading ====================

func (h *Handler) GetProjectGrades(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	resp, err := h.adminClient.GetProjectGrades(c.Request.Context(), &adminv1.GetProjectGradesRequest{
		ProjectId: projectID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) SetStepGrade(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	graderID := c.GetInt64("userId")

	var req struct {
		StepID  int64  `json:"step_id" binding:"required"`
		Grade   int32  `json:"grade" binding:"required,min=0,max=100"`
		Comment string `json:"comment"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	resp, err := h.adminClient.SetStepGrade(c.Request.Context(), &adminv1.SetStepGradeRequest{
		ProjectId: projectID,
		StepId:    req.StepID,
		Grade:     req.Grade,
		Comment:   req.Comment,
		GraderId:  graderID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ==================== Workflow Progress ====================

func (h *Handler) GetWorkflowProgress(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")

	var workflowID int64
	if w := c.Query("workflow_id"); w != "" {
		workflowID, _ = strconv.ParseInt(w, 10, 64)
	}

	resp, err := h.adminClient.GetWorkflowProgress(c.Request.Context(), &adminv1.GetWorkflowProgressRequest{
		DepartmentId: departmentID,
		WorkflowId:   workflowID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListPendingReviews(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")

	page := int32(1)
	pageSize := int32(20)
	if p := c.Query("page"); p != "" {
		if v, _ := strconv.ParseInt(p, 10, 32); v > 0 {
			page = int32(v)
		}
	}

	resp, err := h.adminClient.ListPendingReviews(c.Request.Context(), &adminv1.ListPendingReviewsRequest{
		DepartmentId: departmentID,
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ==================== Supervisor Request Routes ====================

// POST /api/v1/projects/:id/supervisor-request
func (h *Handler) CreateSupervisorRequest(c *gin.Context) {
	userID := c.GetInt64("userId")

	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || projectID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
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

	resp, err := h.adminClient.CreateSupervisorRequest(c.Request.Context(), &adminv1.CreateSupervisorRequestReq{
		ProjectId:     projectID,
		TeamId:        0, // optional; admin_service resolves via project runtime
		SupervisorId:  req.SupervisorID,
		RequestedBy:   userID,
		Message:       req.Message,
		ProposedTopic: req.ProposedTopic,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// ListSupervisorRequests - список всех запросов (для админки)
func (h *Handler) ListAllSupervisorRequests(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")

	var supervisorID, teamID int64
	if s := c.Query("supervisor_id"); s != "" {
		supervisorID, _ = strconv.ParseInt(s, 10, 64)
	}
	if t := c.Query("team_id"); t != "" {
		teamID, _ = strconv.ParseInt(t, 10, 64)
	}

	page := int32(1)
	pageSize := int32(20)
	if p := c.Query("page"); p != "" {
		if v, _ := strconv.ParseInt(p, 10, 32); v > 0 {
			page = int32(v)
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, _ := strconv.ParseInt(ps, 10, 32); v > 0 {
			pageSize = int32(v)
		}
	}

	resp, err := h.adminClient.ListSupervisorRequests(c.Request.Context(), &adminv1.ListSupervisorRequestsReq{
		DepartmentId: departmentID,
		SupervisorId: supervisorID,
		TeamId:       teamID,
		Status:       c.Query("status"),
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetSupervisorRequest - получение детального запроса
func (h *Handler) GetSupervisorRequestDetails(c *gin.Context) {
	requestID := c.Param("id")
	if requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request_id is required"})
		return
	}

	resp, err := h.adminClient.GetSupervisorRequest(c.Request.Context(), &adminv1.GetSupervisorRequestReq{
		RequestId: requestID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// RespondToSupervisorRequest - ответ супервайзера на запрос
func (h *Handler) RespondToSupervisorRequest(c *gin.Context) {
	requestID := c.Param("id")
	supervisorID := c.GetInt64("userId")

	var req struct {
		Action       string `json:"action" binding:"required"` // approve, reject
		RejectReason string `json:"reject_reason"`
		Comment      string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.adminClient.RespondToSupervisorRequest(c.Request.Context(), &adminv1.RespondToSupervisorRequestReq{
		RequestId:    requestID,
		SupervisorId: supervisorID,
		Action:       req.Action,
		RejectReason: req.RejectReason,
		Comment:      req.Comment,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ListMySupervisorRequests - входящие запросы для супервайзера
func (h *Handler) ListMySupervisorRequests(c *gin.Context) {
	supervisorID := c.GetInt64("userId")

	page := int32(1)
	pageSize := int32(20)
	if p := c.Query("page"); p != "" {
		if v, _ := strconv.ParseInt(p, 10, 32); v > 0 {
			page = int32(v)
		}
	}

	resp, err := h.adminClient.ListMySupervisorRequests(c.Request.Context(), &adminv1.ListMySupervisorRequestsReq{
		SupervisorId: supervisorID,
		Status:       c.Query("status"),
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	// optional: safety fallback (как у вас на фронте)
	pending := resp.PendingCount
	if pending == 0 && len(resp.Requests) > 0 {
		cnt := 0
		for _, r := range resp.Requests {
			if r != nil && r.Status == "pending" {
				cnt++
			}
		}
		pending = int32(cnt)
	}

	c.JSON(http.StatusOK, gin.H{
		"requests":       resp.Requests,
		"total_count":    resp.TotalCount,
		"pending_count":  pending,
		"active_teams":   resp.ActiveTeams,
		"total_students": resp.TotalStudents,
		"teams_report":   resp.TeamsReport,
	})
}

// DELETE /api/v1/projects/:id/supervisor-requests/:request_id
func (h *Handler) CancelSupervisorRequest(c *gin.Context) {
	userID := c.GetInt64("userId")

	requestID := c.Param("request_id")
	if requestID == "" {
		// fallback (если вдруг кто-то вызвал старый путь)
		requestID = c.Param("id")
	}
	if requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request_id is required"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	resp, err := h.adminClient.CancelSupervisorRequest(c.Request.Context(), &adminv1.CancelSupervisorRequestReq{
		RequestId:   requestID,
		CancelledBy: userID,
		Reason:      req.Reason,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
