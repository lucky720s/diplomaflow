package usecase

import (
	"context"
	"github.com/google/uuid"
	"github.com/lucky720s/diplomaflow/internal/domain"
	"github.com/lucky720s/diplomaflow/pkg/errors"
	"github.com/lucky720s/diplomaflow/pkg/logger"
)

type DepartmentUsecase struct {
	repo domain.DepartmentRepository
	log  *logger.Logger
}

func NewDepartmentUsecase(repo domain.DepartmentRepository, log *logger.Logger) *DepartmentUsecase {
	return &DepartmentUsecase{
		repo: repo,
		log:  log,
	}
}

func (uc *DepartmentUsecase) CreateDepartment(ctx context.Context, name, universityID string) (*domain.Department, error) {
	dept := &domain.Department{
		ID:           uuid.NewString(),
		Name:         name,
		UniversityID: universityID,
	}

	if err := uc.repo.Create(ctx, dept); err != nil {
		return nil, errors.WrapErrorf(err, "uc.repo.Create")
	}

	uc.log.Infof("Department created: %s", dept.ID)
	return dept, nil
}

func (uc *DepartmentUsecase) GetDepartmentByID(ctx context.Context, id string) (*domain.Department, error) {
	dept, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.WrapErrorf(err, "uc.repo.GetByID")
	}
	return dept, nil
}
