package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
)

func (h *Handler) GetWorkflow(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}

	res, err := h.workflowClient.GetWorkflow(c.Request.Context(), &workflowv1.GetWorkflowRequest{
		Criteria: &workflowv1.GetWorkflowRequest_WorkflowId{WorkflowId: id},
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) ListWorkflows(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")

	res, err := h.workflowClient.ListWorkflows(c.Request.Context(), &workflowv1.ListWorkflowsRequest{
		DepartmentId: departmentID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) CreateState(c *gin.Context) {
	workflowIDStr := c.Param("id")
	workflowID, err := strconv.ParseInt(workflowIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}

	var req workflowv1.CreateStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.WorkflowId = workflowID

	res, err := h.workflowClient.CreateState(c.Request.Context(), &req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusCreated, res)
}

func (h *Handler) SetActiveWorkflow(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}

	res, err := h.workflowClient.SetActiveWorkflow(c.Request.Context(), &workflowv1.SetActiveWorkflowRequest{
		WorkflowId: id,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}
