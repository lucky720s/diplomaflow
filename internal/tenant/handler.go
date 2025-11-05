package tenant

import (
	"context"
	tenant_pb "github.com/lucky720s/diplomaflow/pkg/protobuf/tenant"
)

type Handler struct {
	tenant_pb.UnimplementedTenantServiceServer
	repo *TenantRepository
}

func NewHandler() *Handler {
	return &Handler{repo: NewTenantRepository()}
}

func (h *Handler) GetWorkflowTemplateID(ctx context.Context, req *tenant_pb.GetWorkflowTemplateIDRequest) (*tenant_pb.GetWorkflowTemplateIDResponse, error) {
	templateID, err := h.repo.GetTemplateID(ctx, req.GetDepartmentId())
	if err != nil {
		return nil, err
	}
	return &tenant_pb.GetWorkflowTemplateIDResponse{TemplateId: templateID}, nil
}
