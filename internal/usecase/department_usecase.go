package usecase

import (
	"context"
	"github.com/google/uuid"
	"github.com/lucky720s/diplomaflow/internal/domain"
	apperrors "github.com/lucky720s/diplomaflow/pkg/errors"
)

type DepartmentUsecase struct {
	repo domain.DepartmentRepository
}

func NewDepartmentUsecase(repo domain.DepartmentRepository) *DepartmentUsecase {
	return &DepartmentUsecase{repo: repo}
}

func (uc *DepartmentUsecase) Create(ctx context.Context, name, universityIDStr string) (*domain.Department, error) {
	universityID, err := uuid.Parse(universityIDStr)
	if err != nil {
		return nil, apperrors.WrapErrorf(err, "uuid.Parse")
	}

	dept := &domain.Department{
		ID:           uuid.New(),
		Name:         name,
		UniversityID: universityID,
	}

	if err := uc.repo.Create(ctx, dept); err != nil {
		return nil, apperrors.WrapErrorf(err, "uc.repo.Create")
	}

	return dept, nil
}

func (uc *DepartmentUsecase) GetByID(ctx context.Context, idStr string) (*domain.Department, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, apperrors.WrapErrorf(err, "uuid.Parse")
	}

	dept, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.WrapErrorf(err, "uc.repo.GetByID")
	}
	return dept, nil
}
