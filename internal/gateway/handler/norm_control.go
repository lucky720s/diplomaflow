package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	"google.golang.org/grpc/metadata"
)

func normCtx(c *gin.Context) context.Context {
	return metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(c.GetInt64("userId"), 10),
		"x-user-role", c.GetString("role"),
		"x-university-id", strconv.FormatInt(c.GetInt64("universityId"), 10),
		"x-department-id", strconv.FormatInt(c.GetInt64("departmentId"), 10),
		"x-trace-id", c.GetString("trace_id"),
	)
}

func (h *Handler) NormListPending(c *gin.Context) {
	page := int32(1)
	pageSize := int32(20)

	if v := c.Query("page"); v != "" {
		if x, err := strconv.ParseInt(v, 10, 32); err == nil && x > 0 {
			page = int32(x)
		}
	}
	if v := c.Query("page_size"); v != "" {
		if x, err := strconv.ParseInt(v, 10, 32); err == nil && x > 0 {
			pageSize = int32(x)
		}
	}
	if v := c.Query("pageSize"); v != "" {
		if x, err := strconv.ParseInt(v, 10, 32); err == nil && x > 0 {
			pageSize = int32(x)
		}
	}
	dept := c.GetInt64("departmentId")
	if c.GetString("role") == "admin" {
		if q := parseInt64(c.Query("departmentId")); q > 0 {
			dept = q
		}
	}
	req := &adminv1.ListPendingDocumentsRequest{
		Status:       c.Query("status"),
		TeamId:       parseInt64(c.Query("teamId")),
		CheckerId:    parseInt64(c.Query("checkerId")),
		DepartmentId: dept,
		Page:         page,
		PageSize:     pageSize,
	}

	resp, err := h.normControlClient.ListPendingDocuments(normCtx(c), req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) NormGetDocument(c *gin.Context) {
	id := c.Param("id")
	resp, err := h.normControlClient.GetDocumentForReview(normCtx(c), &adminv1.GetDocumentRequest{Id: id})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) NormStartReview(c *gin.Context) {
	id := c.Param("id")
	resp, err := h.normControlClient.StartReview(normCtx(c), &adminv1.StartReviewRequest{
		Id:        id,
		CheckerId: c.GetInt64("userId"), // will be ignored in admin_service anyway
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) NormAddIssue(c *gin.Context) {
	docID := c.Param("id")

	var body struct {
		Category    string `json:"category"`
		Severity    string `json:"severity"`
		Description string `json:"description"`
		PageNumber  int32  `json:"page_number"`
		LineNumber  int32  `json:"line_number"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	resp, err := h.normControlClient.AddIssue(normCtx(c), &adminv1.AddIssueRequest{
		DocumentId:  docID,
		Category:    body.Category,
		Severity:    body.Severity,
		Description: body.Description,
		PageNumber:  body.PageNumber,
		LineNumber:  body.LineNumber,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) NormUpdateIssue(c *gin.Context) {
	issueID := c.Param("id")

	var body struct {
		Category    string `json:"category"`
		Severity    string `json:"severity"`
		Description string `json:"description"`
		PageNumber  int32  `json:"page_number"`
		LineNumber  int32  `json:"line_number"`
		IsResolved  bool   `json:"is_resolved"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	resp, err := h.normControlClient.UpdateIssue(normCtx(c), &adminv1.UpdateIssueRequest{
		Id:          issueID,
		Category:    body.Category,
		Severity:    body.Severity,
		Description: body.Description,
		PageNumber:  body.PageNumber,
		LineNumber:  body.LineNumber,
		IsResolved:  body.IsResolved,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) NormDeleteIssue(c *gin.Context) {
	issueID := c.Param("id")
	resp, err := h.normControlClient.DeleteIssue(normCtx(c), &adminv1.DeleteIssueRequest{Id: issueID})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) NormApprove(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Comment string `json:"comment"`
	}
	_ = c.ShouldBindJSON(&body)

	resp, err := h.normControlClient.ApproveDocument(normCtx(c), &adminv1.ApproveDocumentRequest{
		Id:        id,
		CheckerId: c.GetInt64("userId"),
		Comment:   body.Comment,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) NormReturn(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Comment string `json:"comment"`
	}
	_ = c.ShouldBindJSON(&body)

	resp, err := h.normControlClient.ReturnForRevision(normCtx(c), &adminv1.ReturnForRevisionRequest{
		Id:        id,
		CheckerId: c.GetInt64("userId"),
		Comment:   body.Comment,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) NormHistory(c *gin.Context) {
	pid := parseInt64(c.Param("project_id"))
	resp, err := h.normControlClient.GetCheckHistory(normCtx(c), &adminv1.GetCheckHistoryRequest{ProjectId: pid})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	// фронту обычно удобнее {history:[...]}
	c.JSON(http.StatusOK, gin.H{"history": resp.History})
}

func (h *Handler) NormStatistics(c *gin.Context) {
	deptID := c.GetInt64("departmentId")
	if c.GetString("role") == "admin" {
		if q := parseInt64(c.Query("departmentId")); q > 0 {
			deptID = q
		}
	}

	resp, err := h.normControlClient.GetErrorStatistics(normCtx(c), &adminv1.GetStatisticsRequest{DepartmentId: deptID})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) NormListChecklists(c *gin.Context) {
	resp, err := h.normControlClient.ListChecklists(normCtx(c), &adminv1.ListChecklistsRequest{})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"checklists": resp.Checklists})
}

func (h *Handler) NormCreateChecklist(c *gin.Context) {
	var body struct {
		Name     string `json:"name"`
		Category string `json:"category"`
		Items    any    `json:"items"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	// Items as raw JSON string
	itemsBytes, _ := json.Marshal(body.Items)

	resp, err := h.normControlClient.CreateChecklist(normCtx(c), &adminv1.CreateChecklistRequest{
		Name:      body.Name,
		Category:  body.Category,
		ItemsJson: string(itemsBytes),
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
