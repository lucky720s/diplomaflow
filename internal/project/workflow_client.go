package project

import (
	"context"

	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"google.golang.org/grpc"
)

type WorkflowClient interface {
	GetActiveWorkflowByDepartment(ctx context.Context, in *workflowv1.GetActiveWorkflowByDepartmentRequest, opts ...grpc.CallOption) (*workflowv1.Workflow, error)
	GetAvailableTransitions(ctx context.Context, in *workflowv1.GetAvailableTransitionsRequest, opts ...grpc.CallOption) (*workflowv1.GetAvailableTransitionsResponse, error)
	ExecuteTransition(ctx context.Context, in *workflowv1.ExecuteTransitionRequest, opts ...grpc.CallOption) (*workflowv1.ExecuteTransitionResponse, error)
}
