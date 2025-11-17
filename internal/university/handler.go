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
func (h *Handler) CreateUniversity(ctx context.Context, req *universityv1.CreateUniversityRequest) (*universityv1.CreateUniversityResponse, error) {
	university, err := h.repo.CreateUniversity(ctx, req.GetName(), req.GetShortName())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &universityv1.CreateUniversityResponse{
		University: &universityv1.University{
			Id:        university.ID,
			Name:      university.Name,
			ShortName: university.ShortName,
		},
	}, nil
}

func (h *Handler) GetUniversity(ctx context.Context, req *universityv1.GetUniversityRequest) (*universityv1.GetUniversityResponse, error) {
	var university *University
	var err error
	if shortName := req.GetShortName(); shortName != "" {
		university, err = h.repo.GetUniversityByShortName(ctx, shortName)
	} else if id := req.GetId(); id != 0 {
		university, err = h.repo.GetUniversityByID(ctx, id)
	} else {
		return nil, status.Error(codes.NotFound, "university not found")
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &universityv1.GetUniversityResponse{
		University: &universityv1.University{
			Id:        university.ID,
			Name:      university.Name,
			ShortName: university.ShortName,
		}}, nil
}

func (h *Handler) CreateDepartment(ctx context.Context, req *universityv1.CreateDepartmentRequest) (*universityv1.CreateDepartmentResponse, error) {
	department, err := h.repo.CreateDepartment(ctx, req.GetName(), req.GetUniversityId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &universityv1.CreateDepartmentResponse{
		Department: &universityv1.Department{
			Id:           department.ID,
			Name:         department.Name,
			UniversityId: department.UniversityID,
		}}, nil
}
func (h *Handler) GetDepartment(ctx context.Context, req *universityv1.GetDepartmentRequest) (*universityv1.GetDepartmentResponse, error) {
	department, err := h.repo.GetDepartmentByID(ctx, req.GetDepartmentId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &universityv1.GetDepartmentResponse{
		Department: &universityv1.Department{
			Id:           department.ID,
			Name:         department.Name,
			UniversityId: department.UniversityID,
		}}, nil
}
