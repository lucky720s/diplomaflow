package university

import (
	"context"
	"errors"

	universityv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/university/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type Handler struct {
	universityv1.UnimplementedUniversityServiceServer
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}
func toProtoUniversity(u *University) *universityv1.University {
	return &universityv1.University{
		Id:        u.ID,
		Name:      u.Name,
		ShortName: u.ShortName,
	}
}
func toProtoDepartment(u *Department) *universityv1.Department {
	return &universityv1.Department{
		Id:           u.ID,
		Name:         u.Name,
		UniversityId: u.UniversityID,
	}
}
func (h *Handler) CreateUniversity(ctx context.Context, req *universityv1.CreateUniversityRequest) (*universityv1.CreateUniversityResponse, error) {
	university, err := h.repo.CreateUniversity(ctx, req.GetName(), req.GetShortName())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error in creating university: %v", err)
	}
	return &universityv1.CreateUniversityResponse{University: toProtoUniversity(university)}, nil
}
func (h *Handler) GetUniversity(ctx context.Context, req *universityv1.GetUniversityRequest) (*universityv1.GetUniversityResponse, error) {
	university, err := h.repo.GetUniversityByID(ctx, req.GetUniversityId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "university not found")
		}
		return nil, status.Errorf(codes.Internal, "error in getting university: %v", err)
	}
	return &universityv1.GetUniversityResponse{University: toProtoUniversity(university)}, nil
}

func (h *Handler) ListUniversities(ctx context.Context, req *universityv1.ListUniversitiesRequest) (*universityv1.ListUniversitiesResponse, error) {
	universities, err := h.repo.ListUniversities(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error in listing universities: %v", err)
	}
	resUniversities := make([]*universityv1.University, len(universities))
	for i, university := range universities {
		resUniversities[i] = toProtoUniversity(university)
	}
	return &universityv1.ListUniversitiesResponse{Universities: resUniversities}, nil
}

func (h *Handler) UpdateUniversity(ctx context.Context, req *universityv1.UpdateUniversityRequest) (*universityv1.UpdateUniversityResponse, error) {
	updatedUniversity, err := h.repo.UpdateUniversity(ctx, req.GetUniversity(), req.GetUpdateMask())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "university not found")
		}
		return nil, status.Errorf(codes.Internal, "error in updating university: %v", err)
	}
	return &universityv1.UpdateUniversityResponse{University: toProtoUniversity(updatedUniversity)}, nil
}
func (h *Handler) DeleteUniversity(ctx context.Context, req *universityv1.DeleteUniversityRequest) (*universityv1.DeleteUniversityResponse, error) {
	if err := h.repo.DeleteUniversity(ctx, req.GetUniversityId()); err != nil {
		return nil, status.Errorf(codes.Internal, "error in deleting university: %v", err)
	}
	return &universityv1.DeleteUniversityResponse{Success: true}, nil
}

func (h *Handler) CreateDepartment(ctx context.Context, req *universityv1.CreateDepartmentRequest) (*universityv1.CreateDepartmentResponse, error) {
	department, err := h.repo.CreateDepartment(ctx, req.GetName(), req.GetUniversityId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error in creating department: %v", err)
	}
	return &universityv1.CreateDepartmentResponse{Department: toProtoDepartment(department)}, nil
}
func (h *Handler) GetDepartment(ctx context.Context, req *universityv1.GetDepartmentRequest) (*universityv1.GetDepartmentResponse, error) {
	department, err := h.repo.GetDepartmentByID(ctx, req.GetDepartmentId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "department not found")
		}
		return nil, status.Errorf(codes.Internal, "error in getting department: %v", err)
	}
	return &universityv1.GetDepartmentResponse{Department: toProtoDepartment(department)}, nil
}

func (h *Handler) ListDepartments(ctx context.Context, req *universityv1.ListDepartmentsRequest) (*universityv1.ListDepartmentsResponse, error) {
	departments, err := h.repo.ListDepartments(ctx, req.GetUniversityId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error in listing departments: %v", err)
	}
	resDepartments := make([]*universityv1.Department, len(departments))
	for _, d := range departments {
		resDepartments = append(resDepartments, toProtoDepartment(d))
	}
	return &universityv1.ListDepartmentsResponse{Departments: resDepartments}, nil
}

func (h *Handler) UpdateDepartment(ctx context.Context, req *universityv1.UpdateDepartmentRequest) (*universityv1.UpdateDepartmentResponse, error) {
	updatedDepartment, err := h.repo.UpdateDepartment(ctx, req.GetDepartment(), req.GetUpdateMask())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "department not found")
		}
		return nil, status.Errorf(codes.Internal, "error in updating department: %v", err)
	}
	return &universityv1.UpdateDepartmentResponse{Department: toProtoDepartment(updatedDepartment)}, nil
}
func (h *Handler) DeleteDepartment(ctx context.Context, req *universityv1.DeleteDepartmentRequest) (*universityv1.DeleteDepartmentResponse, error) {
	if err := h.repo.DeleteDepartment(ctx, req.GetDepartmentId()); err != nil {
		return nil, status.Errorf(codes.Internal, "error in deleting department: %v", err)
	}
	return &universityv1.DeleteDepartmentResponse{Success: true}, nil
}
