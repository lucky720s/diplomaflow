package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
)

func parseInt32Default(s string, def int32) int32 {
	if s == "" {
		return def
	}
	if x, err := strconv.ParseInt(s, 10, 32); err == nil && x > 0 {
		return int32(x)
	}
	return def
}

func (h *Handler) AntiplagListPending(c *gin.Context) {
	page := parseInt32Default(c.Query("page"), 1)
	pageSize := parseInt32Default(c.Query("page_size"), 20)
	if v := c.Query("pageSize"); v != "" {
		pageSize = parseInt32Default(v, pageSize)
	}

	dept := c.GetInt64("departmentId")
	if c.GetString("role") == "admin" {
		if q := parseInt64(c.Query("departmentId")); q > 0 {
			dept = q
		}
	}

	req := &adminv1.ListAntiplagPendingRequest{
		Status:       c.Query("status"),
		TeamId:       parseInt64(c.Query("teamId")),
		CheckerId:    parseInt64(c.Query("checkerId")),
		DepartmentId: dept,
		Page:         page,
		PageSize:     pageSize,
	}

	resp, err := h.antiplagClient.ListAntiplagPending(normCtx(c), req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AntiplagGetDocument(c *gin.Context) {
	id := c.Param("id")
	resp, err := h.antiplagClient.GetAntiplagDocument(normCtx(c), &adminv1.GetAntiplagDocumentRequest{Id: id})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AntiplagStartReview(c *gin.Context) {
	id := c.Param("id")
	resp, err := h.antiplagClient.StartAntiplagReview(normCtx(c), &adminv1.StartAntiplagReviewRequest{
		Id:        id,
		CheckerId: c.GetInt64("userId"),
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AntiplagSetScores(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		PlagiarismScore *int32 `json:"plagiarism_score"`
		AIScore         *int32 `json:"ai_score"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	req := &adminv1.SetAntiplagScoresRequest{Id: id}
	if body.PlagiarismScore != nil {
		req.HasPlagiarism = true
		req.PlagiarismScore = *body.PlagiarismScore
	}
	if body.AIScore != nil {
		req.HasAi = true
		req.AiScore = *body.AIScore
	}

	resp, err := h.antiplagClient.SetAntiplagScores(normCtx(c), req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AntiplagAddComment(c *gin.Context) {
	docID := c.Param("id")
	var body struct {
		Text       string `json:"text"`
		PageNumber int32  `json:"page_number"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	resp, err := h.antiplagClient.AddAntiplagComment(normCtx(c), &adminv1.AddAntiplagCommentRequest{
		DocumentId: docID,
		Text:       body.Text,
		PageNumber: body.PageNumber,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AntiplagUpdateComment(c *gin.Context) {
	commentID := c.Param("id")
	var body struct {
		Text       string `json:"text"`
		PageNumber int32  `json:"page_number"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	resp, err := h.antiplagClient.UpdateAntiplagComment(normCtx(c), &adminv1.UpdateAntiplagCommentRequest{
		Id:         commentID,
		Text:       body.Text,
		PageNumber: body.PageNumber,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AntiplagDeleteComment(c *gin.Context) {
	commentID := c.Param("id")
	resp, err := h.antiplagClient.DeleteAntiplagComment(normCtx(c), &adminv1.DeleteAntiplagCommentRequest{Id: commentID})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AntiplagApprove(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Comment string `json:"comment"`
	}
	_ = c.ShouldBindJSON(&body)

	resp, err := h.antiplagClient.ApproveAntiplagDocument(normCtx(c), &adminv1.ApproveAntiplagRequest{
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

func (h *Handler) AntiplagReturn(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Comment string `json:"comment"`
	}
	_ = c.ShouldBindJSON(&body)

	resp, err := h.antiplagClient.ReturnAntiplagForRevision(normCtx(c), &adminv1.ReturnAntiplagRequest{
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

func (h *Handler) AntiplagHistory(c *gin.Context) {
	pid := parseInt64(c.Param("project_id"))
	resp, err := h.antiplagClient.GetAntiplagHistory(normCtx(c), &adminv1.GetAntiplagHistoryRequest{ProjectId: pid})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"history": resp.History})
}
