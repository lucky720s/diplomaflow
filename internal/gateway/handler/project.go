package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

func (h *Handler) CreateProject(c *gin.Context) {
	studentID := c.GetInt64("userId")
	universityID := c.GetInt64("universityId")
	departmentID := c.GetInt64("departmentId")
	if studentID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}

	var req projectv1.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.StudentId = studentID
	req.UniversityId = universityID
	req.DepartmentId = departmentID
	ctx := metadata.AppendToOutgoingContext(c.Request.Context(),
		"x-university-id", strconv.FormatInt(universityID, 10),
		"x-department-id", strconv.FormatInt(departmentID, 10),
	)
	res, err := h.projectClient.CreateProject(ctx, &req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) GetProject(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	res, err := h.projectClient.GetProject(c.Request.Context(), &projectv1.GetProjectRequest{ProjectId: id})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetStudentProjects(c *gin.Context) {
	studentID, _ := strconv.ParseInt(c.Param("student_id"), 10, 64)

	res, err := h.projectClient.GetStudentProjects(c.Request.Context(), &projectv1.GetStudentProjectsRequest{StudentId: studentID})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetProjectDetails(c *gin.Context) {
	idStr := c.Param("id")
	projectID, _ := strconv.ParseInt(idStr, 10, 64)
	traceID := c.GetString("trace_id")

	ctx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-trace-id", traceID)
	g, ctx := errgroup.WithContext(ctx)

	var projectResp *projectv1.GetProjectResponse
	var currentUserInfo map[string]interface{}

	g.Go(func() error {
		var err error
		projectResp, err = h.projectClient.GetProject(ctx, &projectv1.GetProjectRequest{ProjectId: projectID})
		return err
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

	res, err := h.projectClient.GetStudentProjects(c.Request.Context(), &req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) PerformProjectAction(c *gin.Context) {
	projectID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		ActionName string                 `json:"action_name" binding:"required"`
		Payload    map[string]interface{} `json:"payload"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payloadStruct, _ := structpb.NewStruct(req.Payload)

	userID := c.GetInt64("userId")
	role := c.GetString("role")

	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(userID, 10),
		"x-user-role", role,
	)

	res, err := h.projectClient.PerformAction(ctx, &projectv1.PerformActionRequest{
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
