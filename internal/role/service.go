package role

import "context"

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
