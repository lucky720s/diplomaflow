package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"google.golang.org/grpc/metadata"
)

// SubmitReview — преподаватель/комиссия выставляет оценку или допуск по стэйту.
// POST /projects/:id/states/:state_id/review
func (h *Handler) SubmitReview(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || projectID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}
	stateID, err := strconv.ParseInt(c.Param("state_id"), 10, 64)
	if err != nil || stateID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state id"})
		return
	}

	var body struct {
		Decision string `json:"decision"` // "approved" | "rejected" — для admission
		Score    *int32 `json:"score"`    // 0-100 — для score
		Comment  string `json:"comment"`
	}
	if bindErr := c.ShouldBindJSON(&body); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	userID := c.GetInt64("userId")
	roleSlug := c.GetString("role")

	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(userID, 10),
		"x-user-role", roleSlug,
	)

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

	resp, err := h.workflowClient.SubmitReview(ctx, req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetStateReviews — возвращает все голоса и сводку по стэйту проекта.
// GET /projects/:id/states/:state_id/reviews
func (h *Handler) GetStateReviews(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || projectID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}
	stateID, err := strconv.ParseInt(c.Param("state_id"), 10, 64)
	if err != nil || stateID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state id"})
		return
	}

	userID := c.GetInt64("userId")
	roleSlug := c.GetString("role")

	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(userID, 10),
		"x-user-role", roleSlug,
	)

	resp, err := h.workflowClient.GetStateReviews(ctx, &workflowv1.GetStateReviewsRequest{
		ProjectId: projectID,
		StateId:   stateID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
