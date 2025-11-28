package service

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"
)

type StateProcessor interface {
	ProcessAction(ctx context.Context, currentData datatypes.JSON, action string, payload map[string]interface{}) (newData datatypes.JSON, isCompleted bool, err error)
}

type SupervisorSelectionProcessor struct{}

func (p *SupervisorSelectionProcessor) ProcessAction(ctx context.Context, currentData datatypes.JSON, action string, payload map[string]interface{}) (datatypes.JSON, bool, error) {
	if action != "SELECT_SUPERVISOR" {
		return nil, false, fmt.Errorf("unsupported action '%s' for SUPERVISOR_SELECTION", action)
	}
	supervisorIDVal, ok := payload["supervisor_id"]
	if !ok {
		return nil, false, fmt.Errorf("payload must contain 'supervisor_id'")
	}
	supervisorID, ok := supervisorIDVal.(float64)
	if !ok {
		return nil, false, fmt.Errorf("'supervisor_id' must be a number")
	}

	var data struct {
		SelectedSupervisorID int64 `json:"selected_supervisor_id"`
	}
	data.SelectedSupervisorID = int64(supervisorID)
	newData, _ := json.Marshal(data)
	return newData, true, nil
}

type DocumentUploadProcessor struct{}

func (p *DocumentUploadProcessor) ProcessAction(ctx context.Context, currentData datatypes.JSON, action string, payload map[string]interface{}) (datatypes.JSON, bool, error) {
	var data struct {
		FileID     string `json:"file_id,omitempty"`
		IsApproved *bool  `json:"is_approved,omitempty"`
	}
	_ = json.Unmarshal(currentData, &data)

	switch action {
	case "UPLOAD_FILE":
		fileIDVal, ok := payload["file_id"]
		if !ok {
			return nil, false, fmt.Errorf("payload must contain 'file_id'")
		}
		data.FileID = fmt.Sprintf("%v", fileIDVal)
		newData, _ := json.Marshal(data)
		return newData, false, nil
	case "REVIEW_DOCUMENT":
		isApprovedVal, ok := payload["is_approved"]
		if !ok {
			return nil, false, fmt.Errorf("payload must contain 'is_approved'")
		}
		isApproved, ok := isApprovedVal.(bool)
		if !ok {
			return nil, false, fmt.Errorf("'is_approved' must be a boolean")
		}
		data.IsApproved = &isApproved
		newData, _ := json.Marshal(data)
		return newData, true, nil
	default:
		return nil, false, fmt.Errorf("unsupported action '%s' for DOCUMENT_UPLOAD", action)
	}
}
