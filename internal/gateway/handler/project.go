package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/structpb"
)

// NOTE: CreateProject endpoint лучше удалить из gateway (см. ниже патч роутинга).
// Оставляю метод, чтобы сборка не падала, если где-то ещё на него ссылаются.
func (h *Handler) CreateProject(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{"error": "project creation is not available via gateway; use supervisor approve/claim flow"})
}

func (h *Handler) GetProject(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	res, err := h.projectClient.GetProject(outgoingCtx(c), &projectv1.GetProjectRequest{ProjectId: id})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetStudentProjects(c *gin.Context) {
	studentID, err := strconv.ParseInt(c.Param("student_id"), 10, 64)
	if err != nil || studentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	res, err := h.projectClient.GetStudentProjects(outgoingCtx(c), &projectv1.GetStudentProjectsRequest{
		StudentId: studentID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetProjectDetails(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || projectID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	g, ctx := errgroup.WithContext(outgoingCtx(c))

	var projectResp *projectv1.GetProjectResponse
	var currentUserInfo map[string]interface{}

	g.Go(func() error {
		var e error
		projectResp, e = h.projectClient.GetProject(ctx, &projectv1.GetProjectRequest{ProjectId: projectID})
		return e
	})

	userID := c.GetInt64("userId")
	g.Go(func() error {
		currentUserInfo = map[string]interface{}{
			"id":     userID,
			"role":   c.GetString("role"),
			"status": "active",
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"project": projectResp,
		"viewer":  currentUserInfo,
	})
}

func (h *Handler) ListProjects(c *gin.Context) {
	studentID := c.GetInt64("userId")
	role := c.GetString("role")

	var req projectv1.GetStudentProjectsRequest
	if role == "student" {
		req.StudentId = studentID
	}

	res, err := h.projectClient.GetStudentProjects(outgoingCtx(c), &req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) PerformProjectAction(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || projectID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	var req struct {
		ActionName string                 `json:"action_name" binding:"required"`
		Payload    map[string]interface{} `json:"payload"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	payloadStruct, _ := structpb.NewStruct(req.Payload)

	res, err := h.projectClient.PerformAction(outgoingCtx(c), &projectv1.PerformActionRequest{
		ProjectId:  projectID,
		ActionName: req.ActionName,
		Payload:    payloadStruct,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}
