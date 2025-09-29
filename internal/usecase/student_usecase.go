package usecase

import (
	"context"
	"github.com/google/uuid" // Используем UUID для генерации ID
	"github.com/lucky720s/diplomaflow/internal/domain"
	"github.com/lucky720s/diplomaflow/pkg/errors"
	"github.com/lucky720s/diplomaflow/pkg/logger"
)

// StudentUsecase реализует логику работы со студентами
type StudentUsecase struct {
	studentRepo    domain.StudentRepository
	departmentRepo domain.DepartmentRepository
	log            *logger.Logger
}

// NewStudentUsecase создает новый экземпляр use case для студентов
func NewStudentUsecase(studentRepo domain.StudentRepository, departmentRepo domain.DepartmentRepository, log *logger.Logger) *StudentUsecase {
	return &StudentUsecase{
		studentRepo:    studentRepo,
		departmentRepo: departmentRepo,
		log:            log,
	}
}

// RegisterStudent регистрирует нового студента в системе
func (uc *StudentUsecase) RegisterStudent(ctx context.Context, fullName, departmentID string) (*domain.Student, error) {
	// ПРОВЕРКА СУЩЕСТВОВАНИЯ КАФЕДРЫ
	_, err := uc.departmentRepo.GetByID(ctx, departmentID)
	if err != nil {
		// Если кафедра не найдена, возвращаем ошибку, которую поймет хендлер
		return nil, errors.WrapErrorf(err, "department with id %s not found", departmentID)
	}

	student := &domain.Student{
		ID:           uuid.NewString(),
		FullName:     fullName,
		DepartmentID: departmentID,
	}

	if err := uc.studentRepo.Create(ctx, student); err != nil {
		return nil, errors.WrapErrorf(err, "uc.studentRepo.Create")
	}

	uc.log.Infof("Student registered: %s", student.ID)
	return student, nil
}

// GetStudentByID находит студента по ID
func (uc *StudentUsecase) GetStudentByID(ctx context.Context, id string) (*domain.Student, error) {
	student, err := uc.studentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.WrapErrorf(err, "uc.repo.GetByID")
	}
	return student, nil
}

// ListStudents возвращает список всех студентов
func (uc *StudentUsecase) ListStudents(ctx context.Context) ([]*domain.Student, error) {
	students, err := uc.studentRepo.List(ctx)
	if err != nil {
		return nil, errors.WrapErrorf(err, "uc.repo.List")
	}
	return students, nil
}

// DeleteStudent удаляет студента
func (uc *StudentUsecase) DeleteStudent(ctx context.Context, id string) error {
	err := uc.studentRepo.Delete(ctx, id)
	if err != nil {
		return errors.WrapErrorf(err, "uc.repo.Delete")
	}
	uc.log.Infof("Student deleted: %s", id)
	return nil
}
func (uc *StudentUsecase) GetStudentWithDetails(ctx context.Context, id string) (*domain.StudentWithDepartment, error) {
	student, err := uc.studentRepo.GetWithDepartment(ctx, id)
	if err != nil {
		return nil, errors.WrapErrorf(err, "uc.studentRepo.GetWithDepartment")
	}
	return student, nil
}
