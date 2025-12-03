package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
)

func (h *Handler) CreateProject(c *gin.Context) {
	// Получаем ID студента из токена (безопасно)
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

	// Принудительно ставим ID студента из токена
	req.StudentId = studentID

	res, err := h.projectClient.CreateProject(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) GetProject(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	res, err := h.projectClient.GetProject(context.Background(), &projectv1.GetProjectRequest{ProjectId: id})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ListProjects(c *gin.Context) {
	// Пример: листинг всех проектов (можно добавить фильтры)
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (h *Handler) GetStudentProjects(c *gin.Context) {
	studentID, _ := strconv.ParseInt(c.Param("student_id"), 10, 64)

	res, err := h.projectClient.GetStudentProjects(context.Background(), &projectv1.GetStudentProjectsRequest{StudentId: studentID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
