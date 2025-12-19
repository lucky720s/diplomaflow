package tests_workflow

import (
	"context"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/workflow"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_CreateTransition(t *testing.T) {
	repo := new(MockRepository)
	svc := workflow.NewService(repo)

	repo.On("CreateTransition", mock.Anything, mock.MatchedBy(func(tr *workflow.Transition) bool {
		return tr.FromStateID == 1 && tr.ToStateID == 2
	})).Return(nil)

	req := &workflowv1.CreateTransitionRequest{
		WorkflowId:  10,
		FromStateId: 1,
		ToStateId:   2,
		EventName:   "Submit",
	}

	res, err := svc.CreateTransition(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, int64(300), res.ID)
}
