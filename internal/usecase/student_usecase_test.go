package usecase

import (
	"context"
	"database/sql"
	"errors" // <-- Добавляем стандартный пакет ошибок
	"testing"

	"github.com/lucky720s/diplomaflow/internal/domain"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Структура мока и заглушки остаются без изменений ---

type MockStudentRepository struct {
	mock.Mock
}

func (m *MockStudentRepository) GetWithDepartment(ctx context.Context, id string) (*domain.StudentWithDepartment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.StudentWithDepartment), args.Error(1)
}
func (m *MockStudentRepository) Create(ctx context.Context, student *domain.Student) error {
	args := m.Called(ctx, student)
	return args.Error(0)
}
func (m *MockStudentRepository) GetByID(ctx context.Context, id string) (*domain.Student, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Student), args.Error(1)
}
func (m *MockStudentRepository) List(ctx context.Context) ([]*domain.Student, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Student), args.Error(1)
}
func (m *MockStudentRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// --- ВАШ СТАРЫЙ ТЕСТ (без изменений) ---
func TestStudentUsecase_GetStudentWithDetails_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockStudentRepository)
	log := logger.New()
	uc := NewStudentUsecase(mockRepo, nil, log)

	expectedStudent := &domain.StudentWithDepartment{
		ID:       "test-id",
		FullName: "Тестовый Студент",
	}
	expectedStudent.Department.ID = "dep-test"
	expectedStudent.Department.Name = "Тестовая Кафедра"

	mockRepo.On("GetWithDepartment", mock.Anything, "test-id").Return(expectedStudent, nil)

	// Act
	resultStudent, err := uc.GetStudentWithDetails(context.Background(), "test-id")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resultStudent)
	assert.Equal(t, "test-id", resultStudent.ID)
	mockRepo.AssertExpectations(t)
}

// --- НОВЫЙ ТЕСТ №1: Студент не найден ---
func TestStudentUsecase_GetStudentWithDetails_NotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockStudentRepository)
	log := logger.New()
	uc := NewStudentUsecase(mockRepo, nil, log)

	// Настраиваем мок: "Когда будет вызван метод, нужно вернуть nil и ошибку sql.ErrNoRows"
	mockRepo.On("GetWithDepartment", mock.Anything, "not-found-id").Return(nil, sql.ErrNoRows)

	// Act
	resultStudent, err := uc.GetStudentWithDetails(context.Background(), "not-found-id")

	// Assert
	assert.Error(t, err)                  // Проверяем, что ошибка действительно есть
	assert.Nil(t, resultStudent)          // Проверяем, что студент не был возвращен
	assert.ErrorIs(t, err, sql.ErrNoRows) // Проверяем, что это именно ошибка "не найдено"
	mockRepo.AssertExpectations(t)
}

// --- НОВЫЙ ТЕСТ №2: Общая ошибка БД ---
func TestStudentUsecase_GetStudentWithDetails_RepositoryError(t *testing.T) {
	// Arrange
	mockRepo := new(MockStudentRepository)
	log := logger.New()
	uc := NewStudentUsecase(mockRepo, nil, log)

	// Создаем кастомную ошибку для теста
	expectedError := errors.New("something went wrong in database")

	// Настраиваем мок: "Когда будет вызван метод, нужно вернуть nil и нашу кастомную ошибку"
	mockRepo.On("GetWithDepartment", mock.Anything, "error-id").Return(nil, expectedError)

	// Act
	resultStudent, err := uc.GetStudentWithDetails(context.Background(), "error-id")

	// Assert
	assert.Error(t, err)                  // Проверяем, что ошибка есть
	assert.Nil(t, resultStudent)          // Проверяем, что студент не возвращен
	assert.ErrorIs(t, err, expectedError) // Проверяем, что это именно наша ошибка
	mockRepo.AssertExpectations(t)
}
