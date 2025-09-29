package usecase

import (
	"context"
	"github.com/google/uuid"
	"github.com/lucky720s/diplomaflow/internal/domain"
	apperrors "github.com/lucky720s/diplomaflow/pkg/errors"
)

type StudentUsecase struct {
	studentRepo domain.StudentRepository
}

func NewStudentUsecase(studentRepo domain.StudentRepository) *StudentUsecase {
	return &StudentUsecase{studentRepo: studentRepo}
}

func (uc *StudentUsecase) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.StudentProfile, error) {
	profile, err := uc.studentRepo.GetProfile(ctx, userID)
	if err != nil {
		return nil, apperrors.WrapErrorf(err, "uc.studentRepo.GetProfile")
	}
	return profile, nil
}
