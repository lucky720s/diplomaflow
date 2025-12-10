package university

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
func (s *Service) GetUniversity(ctx context.Context, id int64) (*University, error) {
	return s.repo.GetUniversity(ctx, id)
}

func (s *Service) UpdateUniversity(ctx context.Context, id int64, name, shortName string, updateMask *fieldmaskpb.FieldMask) (*University, error) {
	uni, err := s.repo.GetUniversity(ctx, id)
	if err != nil {
		return nil, err
	}

	if updateMask == nil || len(updateMask.Paths) == 0 {
		uni.Name = name
		uni.ShortName = shortName
	} else {
		for _, path := range updateMask.Paths {
			switch path {
			case "name":
				uni.Name = name
			case "short_name":
				uni.ShortName = shortName
			}
		}
	}

	if err := s.repo.UpdateUniversity(ctx, uni); err != nil {
		return nil, err
	}
	return uni, nil
}

func (s *Service) DeleteUniversity(ctx context.Context, id int64) error {
	return s.repo.DeleteUniversity(ctx, id)
}

func (s *Service) GetDepartment(ctx context.Context, id int64) (*Department, error) {
	return s.repo.GetDepartment(ctx, id)
}

func (s *Service) UpdateDepartment(ctx context.Context, id int64, name string, updateMask *fieldmaskpb.FieldMask) (*Department, error) {
	dep, err := s.repo.GetDepartment(ctx, id)
	if err != nil {
		return nil, err
	}

	if updateMask == nil || len(updateMask.Paths) == 0 {
		dep.Name = name
	} else {
		for _, path := range updateMask.Paths {
			if path == "name" {
				dep.Name = name
			}
		}
	}

	if err := s.repo.UpdateDepartment(ctx, dep); err != nil {
		return nil, err
	}
	return dep, nil
}

func (s *Service) DeleteDepartment(ctx context.Context, id int64) error {
	return s.repo.DeleteDepartment(ctx, id)
}
