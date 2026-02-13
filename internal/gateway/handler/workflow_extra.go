package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
)

func (h *Handler) GetTeamConfiguration(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")

	var workflowID int64
	if v := c.Query("workflow_id"); v != "" {
		workflowID, _ = strconv.ParseInt(v, 10, 64)
	}

	var stateID int64
	if v := c.Query("state_id"); v != "" {
		stateID, _ = strconv.ParseInt(v, 10, 64)
	}

	// Allow override department_id (optional)
	if v := c.Query("department_id"); v != "" {
		if did, err := strconv.ParseInt(v, 10, 64); err == nil && did > 0 {
			departmentID = did
		}
	}

	resp, err := h.workflowClient.GetTeamConfiguration(c.Request.Context(), &workflowv1.GetTeamConfigurationRequest{
		DepartmentId: departmentID,
		WorkflowId:   workflowID,
		StateId:      stateID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetDepartmentWorkflowConfig(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")
	academicYear := c.Query("academic_year")

	// Allow override department_id (optional)
	if v := c.Query("department_id"); v != "" {
		if did, err := strconv.ParseInt(v, 10, 64); err == nil && did > 0 {
			departmentID = did
		}
	}

	resp, err := h.workflowClient.GetDepartmentWorkflowConfig(c.Request.Context(), &workflowv1.GetDepartmentWorkflowConfigRequest{
		DepartmentId: departmentID,
		AcademicYear: academicYear,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
