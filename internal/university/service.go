package university

import (
	"context"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateUniversity(ctx context.Context, name, shortName string) (*University, error) {
	uni := &University{
		Name:      name,
		ShortName: shortName,
	}
	if err := s.repo.CreateUniversity(ctx, uni); err != nil {
		return nil, err
	}
	return uni, nil
}

func (s *Service) ListUniversities(ctx context.Context) ([]*University, error) {
	return s.repo.ListUniversities(ctx)
}

func (s *Service) CreateDepartment(ctx context.Context, name string, uniID int64) (*Department, error) {
	_, err := s.repo.GetUniversity(ctx, uniID)
	if err != nil {
		return nil, err
	}

	dep := &Department{
		Name:         name,
		UniversityID: uniID,
	}
	if err := s.repo.CreateDepartment(ctx, dep); err != nil {
		return nil, err
	}
	return dep, nil
}

func (s *Service) ListDepartments(ctx context.Context, uniID int64) ([]*Department, error) {
	return s.repo.ListDepartments(ctx, uniID)
}
