package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
)

// ---- Workflow Template CRUD ----

func (h *Handler) AdminCreateWorkflow(c *gin.Context) {
	var req workflowv1.CreateWorkflowRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}
	if req.DepartmentId == 0 {
		req.DepartmentId = c.GetInt64("departmentId")
	}
	res, err := h.workflowClient.CreateWorkflow(workflowCtx(c), &req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) AdminGetWorkflow(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}
	res, err := h.workflowClient.GetWorkflow(workflowCtx(c), &workflowv1.GetWorkflowRequest{
		Criteria: &workflowv1.GetWorkflowRequest_WorkflowId{WorkflowId: id},
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AdminGetWorkflowFull(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}
	res, err := h.workflowClient.GetWorkflowFull(workflowCtx(c), &workflowv1.GetWorkflowFullRequest{
		WorkflowId: id,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AdminListWorkflows(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")
	if qd := c.Query("department_id"); qd != "" {
		departmentID = parseInt64(qd)
	}
	res, err := h.workflowClient.ListWorkflows(workflowCtx(c), &workflowv1.ListWorkflowsRequest{
		DepartmentId: departmentID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AdminUpdateWorkflow(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if bindErr := c.ShouldBindJSON(&body); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	res, err := h.workflowClient.UpdateWorkflow(workflowCtx(c), &workflowv1.UpdateWorkflowRequest{
		Id:          id,
		Name:        body.Name,
		Description: body.Description,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AdminDeleteWorkflow(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}
	_, err = h.workflowClient.DeleteWorkflow(workflowCtx(c), &workflowv1.DeleteWorkflowRequest{
		WorkflowId: id,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) AdminCloneWorkflow(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}

	var body struct {
		TargetDepartmentID int64  `json:"target_department_id"`
		NewName            string `json:"new_name"`
		CloneAsTemplate    bool   `json:"clone_as_template"`
	}
	if bindErr := c.ShouldBindJSON(&body); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	res, err := h.workflowClient.CloneWorkflow(workflowCtx(c), &workflowv1.CloneWorkflowRequest{
		SourceWorkflowId:   id,
		TargetDepartmentId: body.TargetDepartmentID,
		NewName:            body.NewName,
		CloneAsTemplate:    body.CloneAsTemplate,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) AdminActivateWorkflow(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}
	res, err := h.workflowClient.SetActiveWorkflow(workflowCtx(c), &workflowv1.SetActiveWorkflowRequest{
		WorkflowId: id,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AdminCreateNewVersion(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}

	var body struct {
		Changelog string `json:"changelog"`
	}
	_ = c.ShouldBindJSON(&body)

	res, err := h.workflowClient.CreateNewVersion(workflowCtx(c), &workflowv1.CreateNewVersionRequest{
		WorkflowId: id,
		Changelog:  body.Changelog,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) AdminValidateWorkflow(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}
	res, err := h.workflowClient.ValidateWorkflow(workflowCtx(c), &workflowv1.ValidateWorkflowRequest{
		WorkflowId: id,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AdminGetActiveWorkflowByDepartment(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")
	if qd := c.Query("department_id"); qd != "" {
		departmentID = parseInt64(qd)
	}
	res, err := h.workflowClient.GetActiveWorkflowByDepartment(workflowCtx(c), &workflowv1.GetActiveWorkflowByDepartmentRequest{
		DepartmentId: departmentID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// ---- Workflow States ----

func (h *Handler) AdminListStates(c *gin.Context) {
	workflowID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}
	res, err := h.workflowClient.ListStates(workflowCtx(c), &workflowv1.ListStatesRequest{
		WorkflowId: workflowID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AdminCreateState(c *gin.Context) {
	workflowID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}
	var req workflowv1.CreateStateRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}
	req.WorkflowId = workflowID
	res, err := h.workflowClient.CreateState(workflowCtx(c), &req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) AdminUpdateState(c *gin.Context) {
	stateID, err := strconv.ParseInt(c.Param("sid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state id"})
		return
	}
	var req workflowv1.UpdateStateRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}
	req.Id = stateID
	res, err := h.workflowClient.UpdateState(workflowCtx(c), &req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AdminDeleteState(c *gin.Context) {
	stateID, err := strconv.ParseInt(c.Param("sid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state id"})
		return
	}
	_, err = h.workflowClient.DeleteState(workflowCtx(c), &workflowv1.DeleteStateRequest{
		StateId: stateID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) AdminReorderStates(c *gin.Context) {
	workflowID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}

	var body struct {
		StateIDs []int64 `json:"state_ids" binding:"required"`
	}
	if bindErr := c.ShouldBindJSON(&body); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	res, err := h.workflowClient.ReorderStates(workflowCtx(c), &workflowv1.ReorderStatesRequest{
		WorkflowId: workflowID,
		StateIds:   body.StateIDs,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// ---- Workflow Transitions ----

func (h *Handler) AdminListTransitions(c *gin.Context) {
	workflowID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}
	res, err := h.workflowClient.ListTransitions(workflowCtx(c), &workflowv1.ListTransitionsRequest{
		WorkflowId: workflowID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AdminCreateTransition(c *gin.Context) {
	workflowID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}

	var body struct {
		EventName   string `json:"event_name" binding:"required"`
		DisplayName string `json:"display_name"`
		FromStateID int64  `json:"from_state_id" binding:"required"`
		ToStateID   int64  `json:"to_state_id" binding:"required"`
		ButtonLabel string `json:"button_label"`
		ButtonColor string `json:"button_color"`
		ConfirmText string `json:"confirm_text"`
	}
	if bindErr := c.ShouldBindJSON(&body); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	res, err := h.workflowClient.CreateTransition(workflowCtx(c), &workflowv1.CreateTransitionRequest{
		WorkflowId:  workflowID,
		EventName:   body.EventName,
		DisplayName: body.DisplayName,
		FromStateId: body.FromStateID,
		ToStateId:   body.ToStateID,
		ButtonLabel: body.ButtonLabel,
		ButtonColor: body.ButtonColor,
		ConfirmText: body.ConfirmText,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) AdminUpdateTransition(c *gin.Context) {
	transitionID, err := strconv.ParseInt(c.Param("tid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transition id"})
		return
	}

	var body struct {
		DisplayName string `json:"display_name"`
		ButtonLabel string `json:"button_label"`
		ButtonColor string `json:"button_color"`
		ConfirmText string `json:"confirm_text"`
	}
	if bindErr := c.ShouldBindJSON(&body); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	res, err := h.workflowClient.UpdateTransition(workflowCtx(c), &workflowv1.UpdateTransitionRequest{
		Id:          transitionID,
		DisplayName: body.DisplayName,
		ButtonLabel: body.ButtonLabel,
		ButtonColor: body.ButtonColor,
		ConfirmText: body.ConfirmText,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AdminDeleteTransition(c *gin.Context) {
	transitionID, err := strconv.ParseInt(c.Param("tid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transition id"})
		return
	}
	_, err = h.workflowClient.DeleteTransition(workflowCtx(c), &workflowv1.DeleteTransitionRequest{
		TransitionId: transitionID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// ---- Project Workflow Runtime ----

func (h *Handler) AdminGetProjectWorkflowState(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}
	res, err := h.workflowClient.GetProjectWorkflowState(workflowCtx(c), &workflowv1.GetProjectWorkflowStateRequest{
		ProjectId: projectID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AdminAdvanceProjectWorkflow(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	var body struct {
		TransitionID int64  `json:"transition_id"`
		ToStateID    int64  `json:"to_state_id"`
		Comment      string `json:"comment"`
	}
	if bindErr := c.ShouldBindJSON(&body); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	res, err := h.workflowClient.AdvanceProjectWorkflow(workflowCtx(c), &workflowv1.AdvanceProjectWorkflowRequest{
		ProjectId:    projectID,
		TransitionId: body.TransitionID,
		ToStateId:    body.ToStateID,
		Comment:      body.Comment,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AdminRollbackProjectWorkflow(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	var body struct {
		ToStateID int64  `json:"to_state_id"`
		Comment   string `json:"comment"`
	}
	_ = c.ShouldBindJSON(&body)

	res, err := h.workflowClient.RollbackProjectWorkflow(workflowCtx(c), &workflowv1.RollbackProjectWorkflowRequest{
		ProjectId: projectID,
		ToStateId: body.ToStateID,
		Comment:   body.Comment,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AdminSetProjectWorkflowState(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	var body struct {
		ToStateID int64  `json:"to_state_id" binding:"required"`
		Comment   string `json:"comment"`
	}
	if bindErr := c.ShouldBindJSON(&body); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	res, err := h.workflowClient.SetProjectWorkflowState(workflowCtx(c), &workflowv1.SetProjectWorkflowStateRequest{
		ProjectId: projectID,
		ToStateId: body.ToStateID,
		Comment:   body.Comment,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AdminListProjectWorkflowHistory(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}
	limit := parseInt64(c.DefaultQuery("limit", "50"))
	res, err := h.workflowClient.ListProjectWorkflowHistory(workflowCtx(c), &workflowv1.ListProjectWorkflowHistoryRequest{
		ProjectId: projectID,
		Limit:     int32(limit),
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// ---- Monitoring ----

func (h *Handler) AdminListWorkflowProjects(c *gin.Context) {
	req := &workflowv1.ListWorkflowProjectsRequest{
		DepartmentId: c.GetInt64("departmentId"),
		WorkflowId:   parseInt64(c.Query("workflow_id")),
		StateId:      parseInt64(c.Query("state_id")),
		Status:       c.Query("status"),
		Search:       c.Query("search"),
		Page:         int32(parseInt64(c.DefaultQuery("page", "1"))),
		PageSize:     int32(parseInt64(c.DefaultQuery("page_size", "20"))),
	}
	if qd := c.Query("department_id"); qd != "" {
		req.DepartmentId = parseInt64(qd)
	}
	res, err := h.workflowClient.ListWorkflowProjects(workflowCtx(c), req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AdminGetWorkflowDashboardStats(c *gin.Context) {
	req := &workflowv1.GetWorkflowDashboardStatsRequest{
		DepartmentId: c.GetInt64("departmentId"),
		WorkflowId:   parseInt64(c.Query("workflow_id")),
	}
	if qd := c.Query("department_id"); qd != "" {
		req.DepartmentId = parseInt64(qd)
	}
	res, err := h.workflowClient.GetWorkflowDashboardStats(workflowCtx(c), req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
