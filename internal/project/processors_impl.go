package project

import (
	"context"
	"fmt"
	"time"
)

type TeamFormedHandler struct{}

func (h *TeamFormedHandler) Handle(ctx context.Context, currentData map[string]interface{}, payload map[string]interface{}, config map[string]interface{}) (map[string]interface{}, error) {
	teamID, ok := payload["team_id"]
	if !ok {
		return nil, fmt.Errorf("missing 'team_id'")
	}
	if tc, ok := config["team_config"].(map[string]interface{}); ok {
		var minSize int
		if val, ok := tc["min_size"].(float64); ok {
			minSize = int(val)
		}
		memberCount := 1
		if memberCount < minSize {
			return nil, fmt.Errorf("team must have at least %d members", minSize)
		}
	}

	currentData["team_id"] = teamID
	currentData["team_formed_at"] = time.Now().Format(time.RFC3339)
	return currentData, nil
}

type SelectSupervisorHandler struct{}

func (h *SelectSupervisorHandler) Handle(ctx context.Context, currentData map[string]interface{}, payload map[string]interface{}, config map[string]interface{}) (map[string]interface{}, error) {
	supervisorID, ok := payload["supervisor_id"]
	if !ok {
		return nil, fmt.Errorf("missing 'supervisor_id'")
	}

	topic, _ := payload["topic"].(string)

	currentData["supervisor_id"] = supervisorID
	currentData["topic"] = topic
	currentData["supervisor_selected_at"] = time.Now().Format(time.RFC3339)

	return currentData, nil
}

type TopicApprovedHandler struct{}

func (h *TopicApprovedHandler) Handle(ctx context.Context, currentData map[string]interface{}, payload map[string]interface{}, config map[string]interface{}) (map[string]interface{}, error) {
	approverID, ok := payload["approver_id"]
	if !ok {
		return nil, fmt.Errorf("missing 'approver_id'")
	}

	currentData["topic_approved"] = true
	currentData["topic_approved_by"] = approverID
	currentData["topic_approved_at"] = time.Now().Format(time.RFC3339)

	return currentData, nil
}

type UploadTaskHandler struct{}

func (h *UploadTaskHandler) Handle(ctx context.Context, currentData map[string]interface{}, payload map[string]interface{}, config map[string]interface{}) (map[string]interface{}, error) {
	fileID, ok := payload["file_id"]
	if !ok {
		return nil, fmt.Errorf("missing 'file_id'")
	}

	if fr, ok := config["file_requirements"].(map[string]interface{}); ok {
		var maxSize int64
		if val, ok := fr["max_size_bytes"].(float64); ok {
			maxSize = int64(val)
		}
		_ = maxSize
	}

	currentData["task_file_id"] = fileID
	currentData["task_uploaded_at"] = time.Now().Format(time.RFC3339)

	return currentData, nil
}

type TaskUploadedHandler struct{}

func (h *TaskUploadedHandler) Handle(ctx context.Context, currentData map[string]interface{}, payload map[string]interface{}, config map[string]interface{}) (map[string]interface{}, error) {
	currentData["task_confirmed"] = true
	return currentData, nil
}

type ApproveHandler struct{}

func (h *ApproveHandler) Handle(ctx context.Context, currentData map[string]interface{}, payload map[string]interface{}, config map[string]interface{}) (map[string]interface{}, error) {
	approverID, ok := payload["approver_id"]
	if !ok {
		return nil, fmt.Errorf("missing 'approver_id'")
	}

	comment, _ := payload["comment"].(string)

	currentData["final_approved"] = true
	currentData["approved_by"] = approverID
	currentData["approval_comment"] = comment
	currentData["approved_at"] = time.Now().Format(time.RFC3339)

	return currentData, nil
}

type RejectHandler struct{}

func (h *RejectHandler) Handle(ctx context.Context, currentData map[string]interface{}, payload map[string]interface{}, config map[string]interface{}) (map[string]interface{}, error) {
	reason, ok := payload["reason"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'reason'")
	}

	currentData["rejected"] = true
	currentData["rejection_reason"] = reason
	currentData["rejected_at"] = time.Now().Format(time.RFC3339)

	return currentData, nil
}
