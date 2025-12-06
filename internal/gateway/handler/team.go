package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
)

func (h *Handler) CreateTeam(c *gin.Context) {
	var req teamv1.CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.teamClient.CreateTeam(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) GetTeam(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	res, err := h.teamClient.GetTeam(context.Background(), &teamv1.GetTeamRequest{TeamId: id})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetAvailableStudents(c *gin.Context) {
	uniID := c.GetInt64("universityId")

	res, err := h.teamClient.GetAvailableStudents(context.Background(), &teamv1.GetAvailableStudentsRequest{UniversityId: uniID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
func (h *Handler) AssignProjectToTeam(c *gin.Context) {
	teamIDStr := c.Param("id")
	teamID, err := strconv.ParseInt(teamIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	var req struct {
		ProjectID int64 `json:"project_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = h.teamClient.AssignProject(context.Background(), &teamv1.AssignProjectRequest{
		TeamId:    teamID,
		ProjectId: req.ProjectID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
