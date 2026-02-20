package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *Handler) SubmitPreDefenseGW(c *gin.Context) {
	userID := c.GetInt64("userId")

	var req struct {
		TeamID      int64    `json:"team_id" binding:"required"`
		ProjectID   int64    `json:"project_id" binding:"required"`
		Message     string   `json:"message"`
		DocumentIDs []string `json:"document_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.adminClient.SubmitPreDefense(adminPanelCtx(c), &adminv1.SubmitPreDefenseRequest{
		TeamId:      req.TeamID,
		ProjectId:   req.ProjectID,
		SubmittedBy: userID,
		Message:     req.Message,
		DocumentIds: req.DocumentIDs,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) ListPreDefenseSubmissionsGW(c *gin.Context) {
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

	var supervisorID, teamID int64
	if s := c.Query("supervisor_id"); s != "" {
		supervisorID, _ = strconv.ParseInt(s, 10, 64)
	}
	if t := c.Query("team_id"); t != "" {
		teamID, _ = strconv.ParseInt(t, 10, 64)
	}

	resp, err := h.adminClient.ListPreDefenseSubmissions(adminPanelCtx(c), &adminv1.ListPreDefenseSubmissionsRequest{
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

func (h *Handler) GetPreDefenseSubmissionGW(c *gin.Context) {
	submissionID := c.Param("id")
	if submissionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "submission_id is required"})
		return
	}

	resp, err := h.adminClient.GetPreDefenseSubmission(adminPanelCtx(c), &adminv1.GetPreDefenseSubmissionRequest{
		SubmissionId: submissionID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) SchedulePreDefenseGW(c *gin.Context) {
	submissionID := c.Param("id")
	userID := c.GetInt64("userId")

	var req struct {
		ScheduledDate       string  `json:"scheduled_date" binding:"required"` // "2025-06-15"
		ScheduledTime       string  `json:"scheduled_time" binding:"required"` // "14:00"
		Location            string  `json:"location"`
		MeetingLink         string  `json:"meeting_link"`
		DurationMinutes     int32   `json:"duration_minutes"`
		CommissionMemberIDs []int64 `json:"commission_member_ids"`
		Comment             string  `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse date
	parsedDate, parseErr := parseDate(req.ScheduledDate)
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scheduled_date format, use YYYY-MM-DD"})
		return
	}

	resp, err := h.adminClient.SchedulePreDefense(adminPanelCtx(c), &adminv1.SchedulePreDefenseRequest{
		SubmissionId:        submissionID,
		ScheduledDate:       timestamppb.New(parsedDate),
		ScheduledTime:       req.ScheduledTime,
		Location:            req.Location,
		MeetingLink:         req.MeetingLink,
		DurationMinutes:     req.DurationMinutes,
		CommissionMemberIds: req.CommissionMemberIDs,
		ScheduledBy:         userID,
		Comment:             req.Comment,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GradePreDefenseGW(c *gin.Context) {
	submissionID := c.Param("id")
	userID := c.GetInt64("userId")

	var req struct {
		Grade        int32  `json:"grade" binding:"required,min=0,max=100"`
		Comment      string `json:"comment"`
		MemberGrades []struct {
			MemberID int64  `json:"member_id"`
			Grade    int32  `json:"grade"`
			Comment  string `json:"comment"`
		} `json:"member_grades"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var pbMemberGrades []*adminv1.CommissionMemberGrade
	for _, mg := range req.MemberGrades {
		pbMemberGrades = append(pbMemberGrades, &adminv1.CommissionMemberGrade{
			MemberId: mg.MemberID,
			Grade:    mg.Grade,
			Comment:  mg.Comment,
		})
	}

	resp, err := h.adminClient.GradePreDefense(adminPanelCtx(c), &adminv1.GradePreDefenseRequest{
		SubmissionId: submissionID,
		GradedBy:     userID,
		Grade:        req.Grade,
		Comment:      req.Comment,
		MemberGrades: pbMemberGrades,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) CompletePreDefenseGW(c *gin.Context) {
	submissionID := c.Param("id")
	userID := c.GetInt64("userId")

	var req struct {
		Result            string   `json:"result" binding:"required"` // passed, failed, conditional
		ResultComment     string   `json:"result_comment"`
		Recommendations   []string `json:"recommendations"`
		AllowResubmission bool     `json:"allow_resubmission"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.adminClient.CompletePreDefense(adminPanelCtx(c), &adminv1.CompletePreDefenseRequest{
		SubmissionId:      submissionID,
		CompletedBy:       userID,
		Result:            req.Result,
		ResultComment:     req.ResultComment,
		Recommendations:   req.Recommendations,
		AllowResubmission: req.AllowResubmission,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ReschedulePreDefenseGW(c *gin.Context) {
	submissionID := c.Param("id")
	userID := c.GetInt64("userId")

	var req struct {
		NewDate     string `json:"new_date" binding:"required"`
		NewTime     string `json:"new_time"`
		NewLocation string `json:"new_location"`
		Reason      string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	parsedDate, parseErr := parseDate(req.NewDate)
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid new_date format, use YYYY-MM-DD"})
		return
	}

	resp, err := h.adminClient.ReschedulePreDefense(adminPanelCtx(c), &adminv1.ReschedulePreDefenseRequest{
		SubmissionId:  submissionID,
		RescheduledBy: userID,
		NewDate:       timestamppb.New(parsedDate),
		NewTime:       req.NewTime,
		NewLocation:   req.NewLocation,
		Reason:        req.Reason,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListScheduledPreDefensesGW(c *gin.Context) {
	var commissionMemberID, departmentID int64
	if s := c.Query("commission_member_id"); s != "" {
		commissionMemberID, _ = strconv.ParseInt(s, 10, 64)
	}
	if s := c.Query("department_id"); s != "" {
		departmentID, _ = strconv.ParseInt(s, 10, 64)
	}
	if departmentID == 0 {
		departmentID = c.GetInt64("departmentId")
	}

	req := &adminv1.ListScheduledPreDefensesRequest{
		DepartmentId:       departmentID,
		CommissionMemberId: commissionMemberID,
	}

	// Parse optional date filters
	if df := c.Query("date_from"); df != "" {
		if parsed, err := parseDate(df); err == nil {
			req.DateFrom = timestamppb.New(parsed)
		}
	}
	if dt := c.Query("date_to"); dt != "" {
		if parsed, err := parseDate(dt); err == nil {
			req.DateTo = timestamppb.New(parsed)
		}
	}

	resp, err := h.adminClient.ListScheduledPreDefenses(adminPanelCtx(c), req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AddPreDefenseCommissionMemberGW(c *gin.Context) {
	submissionID := c.Param("id")
	userID := c.GetInt64("userId")

	var req struct {
		MemberUserID int64  `json:"user_id" binding:"required"`
		Role         string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.adminClient.AddPreDefenseCommissionMember(adminPanelCtx(c), &adminv1.AddPreDefenseCommissionMemberRequest{
		SubmissionId: submissionID,
		UserId:       req.MemberUserID,
		Role:         req.Role,
		AddedBy:      userID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) RemovePreDefenseCommissionMemberGW(c *gin.Context) {
	submissionID := c.Param("id")
	memberUserID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	userID := c.GetInt64("userId")

	resp, grpcErr := h.adminClient.RemovePreDefenseCommissionMember(adminPanelCtx(c), &adminv1.RemovePreDefenseCommissionMemberRequest{
		SubmissionId: submissionID,
		UserId:       memberUserID,
		RemovedBy:    userID,
		Reason:       c.Query("reason"),
	})
	if grpcErr != nil {
		MapGRPCError(c, grpcErr)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// parseDate parses "YYYY-MM-DD" string to time.Time
func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}
