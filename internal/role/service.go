package role

import (
	"context"

	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateRole(ctx context.Context, name string, departmentID int64) (*Role, error) {
	role := &Role{
		Name:         name,
		DepartmentID: departmentID,
	}
	if err := s.repo.Create(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *Service) DeleteRole(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) GetRole(ctx context.Context, id int64) (*Role, error) {
	return s.repo.GetByID(ctx, id)
}
func (s *Service) UpdateRole(ctx context.Context, id int64, name string, departmentID int64, updateMask *fieldmaskpb.FieldMask) (*Role, error) {
	role, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if updateMask == nil || len(updateMask.Paths) == 0 {
		role.Name = name
		role.DepartmentID = departmentID
	} else {
		for _, path := range updateMask.Paths {
			switch path {
			case "name":
				role.Name = name
			case "department_id":
				role.DepartmentID = departmentID
			}
		}
	}

	if err := s.repo.Update(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *Service) ListRoles(ctx context.Context, departmentID int64) ([]*Role, error) {
	return s.repo.List(ctx, departmentID)
}
