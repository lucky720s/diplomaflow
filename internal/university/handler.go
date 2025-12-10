package university

import (
	"context"

	universityv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/university/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	universityv1.UnimplementedUniversityServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateUniversity(ctx context.Context, req *universityv1.CreateUniversityRequest) (*universityv1.CreateUniversityResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	uni, err := h.service.CreateUniversity(ctx, req.Name, req.ShortName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create: %v", err)
	}
	return &universityv1.CreateUniversityResponse{
		University: &universityv1.University{Id: uni.ID, Name: uni.Name, ShortName: uni.ShortName},
	}, nil
}

func (h *Handler) ListUniversities(ctx context.Context, req *universityv1.ListUniversitiesRequest) (*universityv1.ListUniversitiesResponse, error) {
	unis, err := h.service.ListUniversities(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list: %v", err)
	}
	var pbUnis []*universityv1.University
	for _, u := range unis {
		pbUnis = append(pbUnis, &universityv1.University{Id: u.ID, Name: u.Name, ShortName: u.ShortName})
	}
	return &universityv1.ListUniversitiesResponse{Universities: pbUnis}, nil
}

func (h *Handler) CreateDepartment(ctx context.Context, req *universityv1.CreateDepartmentRequest) (*universityv1.CreateDepartmentResponse, error) {
	dep, err := h.service.CreateDepartment(ctx, req.Name, req.UniversityId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create department: %v", err)
	}
	return &universityv1.CreateDepartmentResponse{
		Department: &universityv1.Department{Id: dep.ID, Name: dep.Name, UniversityId: dep.UniversityID},
	}, nil
}
func (h *Handler) ListDepartments(ctx context.Context, req *universityv1.ListDepartmentsRequest) (*universityv1.ListDepartmentsResponse, error) {
	if req.UniversityId == 0 {
		return nil, status.Error(codes.InvalidArgument, "university_id is required")
	}

	deps, err := h.service.ListDepartments(ctx, req.UniversityId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list departments: %v", err)
	}

	var pbDeps []*universityv1.Department
	for _, d := range deps {
		pbDeps = append(pbDeps, &universityv1.Department{
			Id:           d.ID,
			Name:         d.Name,
			UniversityId: d.UniversityID,
		})
	}

	return &universityv1.ListDepartmentsResponse{
		Departments: pbDeps,
	}, nil
}
func (h *Handler) GetUniversity(ctx context.Context, req *universityv1.GetUniversityRequest) (*universityv1.GetUniversityResponse, error) {
	uni, err := h.service.GetUniversity(ctx, req.UniversityId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "university not found")
	}
	return &universityv1.GetUniversityResponse{
		University: &universityv1.University{Id: uni.ID, Name: uni.Name, ShortName: uni.ShortName},
	}, nil
}

func (h *Handler) UpdateUniversity(ctx context.Context, req *universityv1.UpdateUniversityRequest) (*universityv1.UpdateUniversityResponse, error) {
	if req.University == nil {
		return nil, status.Error(codes.InvalidArgument, "university is required")
	}
	uni, err := h.service.UpdateUniversity(ctx, req.University.Id, req.University.Name, req.University.ShortName, req.UpdateMask)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update: %v", err)
	}
	return &universityv1.UpdateUniversityResponse{
		University: &universityv1.University{Id: uni.ID, Name: uni.Name, ShortName: uni.ShortName},
	}, nil
}

func (h *Handler) DeleteUniversity(ctx context.Context, req *universityv1.DeleteUniversityRequest) (*universityv1.DeleteUniversityResponse, error) {
	if err := h.service.DeleteUniversity(ctx, req.UniversityId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete: %v", err)
	}
	return &universityv1.DeleteUniversityResponse{Success: true}, nil
}

func (h *Handler) GetDepartment(ctx context.Context, req *universityv1.GetDepartmentRequest) (*universityv1.GetDepartmentResponse, error) {
	dep, err := h.service.GetDepartment(ctx, req.DepartmentId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "department not found")
	}
	return &universityv1.GetDepartmentResponse{
		Department: &universityv1.Department{Id: dep.ID, Name: dep.Name, UniversityId: dep.UniversityID},
	}, nil
}

func (h *Handler) UpdateDepartment(ctx context.Context, req *universityv1.UpdateDepartmentRequest) (*universityv1.UpdateDepartmentResponse, error) {
	if req.Department == nil {
		return nil, status.Error(codes.InvalidArgument, "department is required")
	}
	dep, err := h.service.UpdateDepartment(ctx, req.Department.Id, req.Department.Name, req.UpdateMask)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update: %v", err)
	}
	return &universityv1.UpdateDepartmentResponse{
		Department: &universityv1.Department{Id: dep.ID, Name: dep.Name, UniversityId: dep.UniversityID},
	}, nil
}

func (h *Handler) DeleteDepartment(ctx context.Context, req *universityv1.DeleteDepartmentRequest) (*universityv1.DeleteDepartmentResponse, error) {
	if err := h.service.DeleteDepartment(ctx, req.DepartmentId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete: %v", err)
	}
	return &universityv1.DeleteDepartmentResponse{Success: true}, nil
}
