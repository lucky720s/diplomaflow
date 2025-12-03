package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/metadata"
)

func (h *Handler) CreateProject(c *gin.Context) {
	studentID := c.GetInt64("userId")
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

	res, err := h.projectClient.CreateProject(context.Background(), &req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) GetProject(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	res, err := h.projectClient.GetProject(context.Background(), &projectv1.GetProjectRequest{ProjectId: id})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ListProjects(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (h *Handler) GetStudentProjects(c *gin.Context) {
	studentID, _ := strconv.ParseInt(c.Param("student_id"), 10, 64)

	res, err := h.projectClient.GetStudentProjects(context.Background(), &projectv1.GetStudentProjectsRequest{StudentId: studentID})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetProjectDetails демонстрирует параллельные запросы
func (h *Handler) GetProjectDetails(c *gin.Context) {
	idStr := c.Param("id")
	projectID, _ := strconv.ParseInt(idStr, 10, 64)
	traceID := c.GetString("trace_id")

	// Прокидываем TraceID
	ctx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-trace-id", traceID)
	g, ctx := errgroup.WithContext(ctx)

	var projectResp *projectv1.GetProjectResponse

	// Переменная для данных пользователя (параллельный запрос)
	// В реальном коде здесь был бы вызов h.authClient.GetUser(...)
	var currentUserInfo map[string]interface{}

	// 1. Получаем проект
	g.Go(func() error {
		var err error
		projectResp, err = h.projectClient.GetProject(ctx, &projectv1.GetProjectRequest{ProjectId: projectID})
		return err
	})

	// 2. Получаем информацию о текущем пользователе (параллельно)
	userID := c.GetInt64("userId")
	g.Go(func() error {
		// Имитация полезной работы или реальный вызов Auth Service
		// userResp, err := h.authClient.GetUser(ctx, &authv1.GetUserRequest{Id: userID})
		// if err != nil { return err }

		// Пока просто заполним мапу, чтобы переменная использовалась
		currentUserInfo = map[string]interface{}{
			"id":     userID,
			"role":   c.GetString("role"),
			"status": "active", // Пример данных, полученных параллельно
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"project": projectResp,
		"viewer":  currentUserInfo, // Теперь переменная используется
	})
}
