package project

import (
	"context"

	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
)

type WorkflowClient interface {
	GetActiveWorkflowByDepartment(ctx context.Context, in *workflowv1.GetActiveWorkflowByDepartmentRequest, opts ...interface{}) (*workflowv1.Workflow, error)
	GetAvailableTransitions(ctx context.Context, in *workflowv1.GetAvailableTransitionsRequest, opts ...interface{}) (*workflowv1.GetAvailableTransitionsResponse, error)
	ExecuteTransition(ctx context.Context, in *workflowv1.ExecuteTransitionRequest, opts ...interface{}) (*workflowv1.ExecuteTransitionResponse, error)
}
