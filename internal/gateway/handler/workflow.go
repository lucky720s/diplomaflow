package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
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
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
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
func (h *Handler) GetWorkflowFull(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}

	res, err := h.workflowClient.GetWorkflowFull(c.Request.Context(), &workflowv1.GetWorkflowFullRequest{
		WorkflowId: id,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
func (h *Handler) GetAvailableTransitions(c *gin.Context) {
	projectID, _ := strconv.ParseInt(c.Query("project_id"), 10, 64)
	stateID, _ := strconv.ParseInt(c.Query("state_id"), 10, 64)
	ctx := workflowCtx(c)

	res, err := h.workflowClient.GetAvailableTransitions(ctx, &workflowv1.GetAvailableTransitionsRequest{
		ProjectId:      projectID,
		CurrentStateId: stateID,
		UserId:         c.GetInt64("userId"),
		UserRole:       c.GetString("role"),
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetStepConfiguration(c *gin.Context) {
	stateID, _ := strconv.ParseInt(c.Param("state_id"), 10, 64)
	projectID, _ := strconv.ParseInt(c.Query("project_id"), 10, 64)

	res, err := h.workflowClient.GetStepConfiguration(c.Request.Context(), &workflowv1.GetStepConfigurationRequest{
		StateId:   stateID,
		ProjectId: projectID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
func (h *Handler) ListStates(c *gin.Context) {
	workflowID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	res, err := h.workflowClient.ListStates(c.Request.Context(), &workflowv1.ListStatesRequest{
		WorkflowId: workflowID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
func (h *Handler) ListTransitions(c *gin.Context) {
	workflowID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	res, err := h.workflowClient.ListTransitions(c.Request.Context(), &workflowv1.ListTransitionsRequest{
		WorkflowId: workflowID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) CloneWorkflow(c *gin.Context) {
	workflowID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}

	var req struct {
		TargetDepartmentID int64  `json:"target_department_id"`
		NewName            string `json:"new_name"`
		CloneAsTemplate    bool   `json:"clone_as_template"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	res, err := h.workflowClient.CloneWorkflow(c.Request.Context(), &workflowv1.CloneWorkflowRequest{
		SourceWorkflowId:   workflowID,
		TargetDepartmentId: req.TargetDepartmentID,
		NewName:            req.NewName,
		CloneAsTemplate:    req.CloneAsTemplate,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) CreateNewVersion(c *gin.Context) {
	workflowID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}

	var req struct {
		Changelog string `json:"changelog"`
	}
	_ = c.ShouldBindJSON(&req)

	res, err := h.workflowClient.CreateNewVersion(c.Request.Context(), &workflowv1.CreateNewVersionRequest{
		WorkflowId: workflowID,
		Changelog:  req.Changelog,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) ValidateWorkflow(c *gin.Context) {
	workflowID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}

	res, err := h.workflowClient.ValidateWorkflow(c.Request.Context(), &workflowv1.ValidateWorkflowRequest{
		WorkflowId: workflowID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) CreateTransition(c *gin.Context) {
	workflowID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow id"})
		return
	}

	var req struct {
		EventName   string `json:"event_name" binding:"required"`
		DisplayName string `json:"display_name"`
		FromStateID int64  `json:"from_state_id" binding:"required"`
		ToStateID   int64  `json:"to_state_id" binding:"required"`
		ButtonLabel string `json:"button_label"`
		ButtonColor string `json:"button_color"`
		ConfirmText string `json:"confirm_text"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	res, err := h.workflowClient.CreateTransition(c.Request.Context(), &workflowv1.CreateTransitionRequest{
		WorkflowId:  workflowID,
		EventName:   req.EventName,
		DisplayName: req.DisplayName,
		FromStateId: req.FromStateID,
		ToStateId:   req.ToStateID,
		ButtonLabel: req.ButtonLabel,
		ButtonColor: req.ButtonColor,
		ConfirmText: req.ConfirmText,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) UpdateTransition(c *gin.Context) {
	transitionID, err := strconv.ParseInt(c.Param("tid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transition id"})
		return
	}

	var req struct {
		DisplayName string `json:"display_name"`
		ButtonLabel string `json:"button_label"`
		ButtonColor string `json:"button_color"`
		ConfirmText string `json:"confirm_text"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	res, err := h.workflowClient.UpdateTransition(c.Request.Context(), &workflowv1.UpdateTransitionRequest{
		Id:          transitionID,
		DisplayName: req.DisplayName,
		ButtonLabel: req.ButtonLabel,
		ButtonColor: req.ButtonColor,
		ConfirmText: req.ConfirmText,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) DeleteTransition(c *gin.Context) {
	transitionID, err := strconv.ParseInt(c.Param("tid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transition id"})
		return
	}

	_, err = h.workflowClient.DeleteTransition(c.Request.Context(), &workflowv1.DeleteTransitionRequest{
		TransitionId: transitionID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) CreateStateAction(c *gin.Context) {
	stateID, err := strconv.ParseInt(c.Param("sid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state id"})
		return
	}

	var req struct {
		Name       string                 `json:"name" binding:"required"`
		Type       string                 `json:"type" binding:"required"`
		Trigger    string                 `json:"trigger" binding:"required"`
		OrderIndex int32                  `json:"order_index"`
		Config     map[string]interface{} `json:"config"`
		IsEnabled  bool                   `json:"is_enabled"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	actionType := workflowv1.ActionType(workflowv1.ActionType_value[req.Type])
	actionTrigger := workflowv1.ActionTrigger(workflowv1.ActionTrigger_value[req.Trigger])

	var configStruct *structpb.Struct
	if req.Config != nil {
		configStruct, _ = structpb.NewStruct(req.Config)
	}

	res, err := h.workflowClient.CreateStateAction(c.Request.Context(), &workflowv1.CreateStateActionRequest{
		StateId:    stateID,
		Name:       req.Name,
		Type:       actionType,
		Trigger:    actionTrigger,
		OrderIndex: req.OrderIndex,
		Config:     configStruct,
		IsEnabled:  req.IsEnabled,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}
func workflowCtx(c *gin.Context) context.Context {
	userID := c.GetInt64("userId")
	role := c.GetString("role")
	universityID := c.GetInt64("universityId")
	departmentID := c.GetInt64("departmentId")

	return metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(userID, 10),
		"x-user-role", role,
		"x-university-id", strconv.FormatInt(universityID, 10),
		"x-department-id", strconv.FormatInt(departmentID, 10),
	)
}
