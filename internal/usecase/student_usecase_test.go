package usecase

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/domain"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Мок для StudentRepository ---
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

// --- НОВЫЙ МОК для DepartmentRepository ---
type MockDepartmentRepository struct {
	mock.Mock
}

func (m *MockDepartmentRepository) Create(ctx context.Context, department *domain.Department) error {
	args := m.Called(ctx, department)
	return args.Error(0)
}
func (m *MockDepartmentRepository) GetByID(ctx context.Context, id string) (*domain.Department, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Department), args.Error(1)
}

// --- Тесты для GetStudentWithDetails ---

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

func TestStudentUsecase_GetStudentWithDetails_NotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockStudentRepository)
	log := logger.New()
	uc := NewStudentUsecase(mockRepo, nil, log)

	mockRepo.On("GetWithDepartment", mock.Anything, "not-found-id").Return(nil, sql.ErrNoRows)

	// Act
	resultStudent, err := uc.GetStudentWithDetails(context.Background(), "not-found-id")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, resultStudent)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	mockRepo.AssertExpectations(t)
}

func TestStudentUsecase_GetStudentWithDetails_RepositoryError(t *testing.T) {
	// Arrange
	mockRepo := new(MockStudentRepository)
	log := logger.New()
	uc := NewStudentUsecase(mockRepo, nil, log)

	expectedError := errors.New("something went wrong in database")

	mockRepo.On("GetWithDepartment", mock.Anything, "error-id").Return(nil, expectedError)

	// Act
	resultStudent, err := uc.GetStudentWithDetails(context.Background(), "error-id")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, resultStudent)
	assert.ErrorIs(t, err, expectedError)
	mockRepo.AssertExpectations(t)
}

// --- Тесты для RegisterStudent ---

func TestStudentUsecase_RegisterStudent_Success(t *testing.T) {
	// Arrange
	mockStudentRepo := new(MockStudentRepository)
	mockDepartmentRepo := new(MockDepartmentRepository)
	log := logger.New()
	uc := NewStudentUsecase(mockStudentRepo, mockDepartmentRepo, log)

	departmentID := "existing-dept-id"

	mockDepartmentRepo.On("GetByID", mock.Anything, departmentID).Return(&domain.Department{ID: departmentID}, nil)
	mockStudentRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Student")).Return(nil)

	// Act
	createdStudent, err := uc.RegisterStudent(context.Background(), "Новый Студент", departmentID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, createdStudent)
	assert.Equal(t, "Новый Студент", createdStudent.FullName)
	assert.Equal(t, departmentID, createdStudent.DepartmentID)
	mockDepartmentRepo.AssertExpectations(t)
	mockStudentRepo.AssertExpectations(t)
}

func TestStudentUsecase_RegisterStudent_DepartmentNotFound(t *testing.T) {
	// Arrange
	mockStudentRepo := new(MockStudentRepository)
	mockDepartmentRepo := new(MockDepartmentRepository)
	log := logger.New()
	uc := NewStudentUsecase(mockStudentRepo, mockDepartmentRepo, log)

	departmentID := "non-existent-dept-id"

	mockDepartmentRepo.On("GetByID", mock.Anything, departmentID).Return(nil, sql.ErrNoRows)

	// Act
	createdStudent, err := uc.RegisterStudent(context.Background(), "Новый Студент", departmentID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, createdStudent)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	mockDepartmentRepo.AssertExpectations(t)
	mockStudentRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestStudentUsecase_RegisterStudent_CreateStudentFails(t *testing.T) {
	// Arrange
	mockStudentRepo := new(MockStudentRepository)
	mockDepartmentRepo := new(MockDepartmentRepository)
	log := logger.New()
	uc := NewStudentUsecase(mockStudentRepo, mockDepartmentRepo, log)

	departmentID := "existing-dept-id"
	dbError := errors.New("database write error")

	mockDepartmentRepo.On("GetByID", mock.Anything, departmentID).Return(&domain.Department{ID: departmentID}, nil)
	mockStudentRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Student")).Return(dbError)

	// Act
	createdStudent, err := uc.RegisterStudent(context.Background(), "Новый Студент", departmentID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, createdStudent)
	assert.ErrorIs(t, err, dbError)
	mockDepartmentRepo.AssertExpectations(t)
	mockStudentRepo.AssertExpectations(t)
}
