package project

import (
	"context"
	"fmt"
)

type SupervisorSelectionHandler struct{}

func (h *SupervisorSelectionHandler) Handle(ctx context.Context, currentData map[string]interface{}, payload map[string]interface{}, config map[string]interface{}) (map[string]interface{}, error) {
	val, ok := payload["supervisor_id"]
	if !ok {
		return nil, fmt.Errorf("missing 'supervisor_id'")
	}

	var supervisorID int64
	switch v := val.(type) {
	case float64:
		supervisorID = int64(v)
	case int64:
		supervisorID = v
	default:
		return nil, fmt.Errorf("invalid type for 'supervisor_id'")
	}

	currentData["selected_supervisor_id"] = supervisorID
	currentData["supervisor_selected_at"] = "now"

	return currentData, nil
}

type DocumentUploadHandler struct{}

func (h *DocumentUploadHandler) Handle(ctx context.Context, currentData map[string]interface{}, payload map[string]interface{}, config map[string]interface{}) (map[string]interface{}, error) {
	fileID, ok := payload["file_id"]
	if !ok {
		return nil, fmt.Errorf("missing 'file_id'")
	}
	currentData["uploaded_file_id"] = fmt.Sprintf("%v", fileID)

	return currentData, nil
}

type ApprovalHandler struct{}

func (h *ApprovalHandler) Handle(ctx context.Context, currentData map[string]interface{}, payload map[string]interface{}, config map[string]interface{}) (map[string]interface{}, error) {
	approvedVal, ok := payload["is_approved"]
	if !ok {
		return nil, fmt.Errorf("missing 'is_approved'")
	}

	isApproved, ok := approvedVal.(bool)
	if !ok {
		return nil, fmt.Errorf("'is_approved' must be boolean")
	}

	currentData["is_approved"] = isApproved
	if !isApproved {
		currentData["rejection_reason"] = payload["reason"]
	}

	return currentData, nil
}
