// file: internal/usecase/student_usecase.go
package usecase

import (
	"context"
	"github.com/lucky720s/diplomaflow/pkg/errors"

	"github.com/lucky720s/diplomaflow/internal/domain"
	"github.com/lucky720s/diplomaflow/pkg/logger"
)

// StudentUsecase реализует логику работы со студентами
type StudentUsecase struct {
	repo domain.StudentRepository
	log  *logger.Logger
}

// NewStudentUsecase создает новый экземпляр use case для студентов
func NewStudentUsecase(repo domain.StudentRepository, log *logger.Logger) *StudentUsecase {
	return &StudentUsecase{
		repo: repo,
		log:  log,
	}
}

// RegisterStudent регистрирует нового студента в системе
func (uc *StudentUsecase) RegisterStudent(ctx context.Context, fullName, departmentID string) (*domain.Student, error) {
	// Здесь может быть бизнес-логика:
	// - Проверка, что кафедра (departmentID) существует
	// - Генерация ID для нового студента
	// - Валидация fullName и т.д.

	student := &domain.Student{
		ID:           "generated-uuid", // В реальности использовать библиотеку вроде uuid
		FullName:     fullName,
		DepartmentID: departmentID,
	}

	if err := uc.repo.Create(ctx, student); err != nil {
		// Оборачиваем ошибку для трассировки, как вы и планировали
		return nil, errors.WrapErrorf(err, "uc.repo.Create")
	}

	uc.log.Infof("Student registered: %s", student.ID)

	return student, nil
}
