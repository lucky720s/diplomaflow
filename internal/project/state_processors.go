package project

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
		return nil, false, fmt.Errorf("unsupported action '%s'", action)
	}

	val, ok := payload["supervisor_id"]
	if !ok {
		return nil, false, fmt.Errorf("missing 'supervisor_id'")
	}

	var supervisorID int64
	switch v := val.(type) {
	case float64:
		supervisorID = int64(v)
	case int64:
		supervisorID = v
	default:
		return nil, false, fmt.Errorf("invalid type for 'supervisor_id'")
	}

	data := map[string]interface{}{
		"selected_supervisor_id": supervisorID,
	}

	newDataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, false, err
	}
	return datatypes.JSON(newDataBytes), true, nil
}

type DocumentUploadProcessor struct{}

func (p *DocumentUploadProcessor) ProcessAction(ctx context.Context, currentData datatypes.JSON, action string, payload map[string]interface{}) (datatypes.JSON, bool, error) {
	data := make(map[string]interface{})
	if len(currentData) > 0 {
		if err := json.Unmarshal(currentData, &data); err != nil {
			return nil, false, fmt.Errorf("corrupted state data: %w", err)
		}
	}

	switch action {
	case "UPLOAD_FILE":
		fileID, ok := payload["file_id"]
		if !ok {
			return nil, false, fmt.Errorf("missing 'file_id'")
		}
		data["file_id"] = fmt.Sprintf("%v", fileID)

		newData, err := json.Marshal(data)
		return datatypes.JSON(newData), false, err

	case "REVIEW_DOCUMENT":
		approvedVal, ok := payload["is_approved"]
		if !ok {
			return nil, false, fmt.Errorf("missing 'is_approved'")
		}
		isApproved, ok := approvedVal.(bool)
		if !ok {
			return nil, false, fmt.Errorf("'is_approved' must be boolean")
		}
		data["is_approved"] = isApproved

		newData, err := json.Marshal(data)
		return datatypes.JSON(newData), isApproved, err

	default:
		return nil, false, fmt.Errorf("unknown action '%s'", action)
	}
}
