package workflow

import (
	"context"

	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	"google.golang.org/grpc"
)

// ProjectClient is the subset of ProjectServiceClient that WorkflowService needs.
type ProjectClient interface {
	GetProjectRuntime(ctx context.Context, in *projectv1.GetProjectRuntimeRequest, opts ...grpc.CallOption) (*projectv1.GetProjectRuntimeResponse, error)
	CommitTransition(ctx context.Context, in *projectv1.CommitTransitionRequest, opts ...grpc.CallOption) (*projectv1.CommitTransitionResponse, error)
	ListProjectsRuntime(ctx context.Context, in *projectv1.ListProjectsRuntimeRequest, opts ...grpc.CallOption) (*projectv1.ListProjectsRuntimeResponse, error)
	ListProjectStateHistory(ctx context.Context, in *projectv1.ListProjectStateHistoryRequest, opts ...grpc.CallOption) (*projectv1.ListProjectStateHistoryResponse, error)
}
