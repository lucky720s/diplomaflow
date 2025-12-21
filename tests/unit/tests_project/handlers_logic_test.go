package tests_project

import (
	"context"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/project"
	"github.com/stretchr/testify/require"
)

func TestHandlers_TeamFormed(t *testing.T) {
	h := &project.TeamFormedHandler{}

	// Конфиг требует минимум 2 человека
	config := map[string]interface{}{
		"team_config": map[string]interface{}{"min_size": 2.0},
	}
	// Мы пока не передаем размер команды в payload, но проверим логику
	// В вашем коде: if memberCount < minSize return error. (MemberCount hardcoded to 1)

	payload := map[string]interface{}{"team_id": "team_1"}
	data := make(map[string]interface{})

	// Ожидаем ошибку, так как memberCount=1, а minSize=2
	_, err := h.Handle(context.Background(), data, payload, config)
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least 2 members")
}

func TestHandlers_SelectSupervisor(t *testing.T) {
	h := &project.SelectSupervisorHandler{}

	payload := map[string]interface{}{"supervisor_id": "sup_1", "topic": "AI"}
	data := make(map[string]interface{})

	newData, err := h.Handle(context.Background(), data, payload, nil)

	require.NoError(t, err)
	require.Equal(t, "sup_1", newData["supervisor_id"])
	require.Equal(t, "AI", newData["topic"])
	require.NotEmpty(t, newData["supervisor_selected_at"])
}

func TestHandlers_Approve(t *testing.T) {
	h := &project.ApproveHandler{}

	payload := map[string]interface{}{"approver_id": "admin_1", "comment": "Good job"}
	data := make(map[string]interface{})

	newData, err := h.Handle(context.Background(), data, payload, nil)

	require.NoError(t, err)
	require.Equal(t, true, newData["final_approved"])
	require.Equal(t, "Good job", newData["approval_comment"])
}
