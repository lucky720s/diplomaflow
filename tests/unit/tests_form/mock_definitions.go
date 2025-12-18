package tests_form

import (
	"context"

	"github.com/lucky720s/diplomaflow/internal/form"
	"github.com/stretchr/testify/mock"
)

// --- Mock для тестирования Service ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, submission *form.FormSubmission) error {
	args := m.Called(ctx, submission)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id string) (*form.FormSubmission, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	result, ok := args.Get(0).(*form.FormSubmission)
	if !ok {
		panic("unexpected type for arg 0 in GetByID")
	}
	return result, args.Error(1)
}

func (m *MockRepository) ListByProject(ctx context.Context, projectID int64) ([]*form.FormSubmission, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	result, ok := args.Get(0).([]*form.FormSubmission)
	if !ok {
		panic("unexpected type for arg 0 in ListByProject")
	}
	return result, args.Error(1)
}

// --- Mock для тестирования Handler ---

type MockFormService struct {
	mock.Mock
}

func (m *MockFormService) SubmitForm(ctx context.Context, projectID, stepID, userID int64, data map[string]interface{}) (string, error) {
	args := m.Called(ctx, projectID, stepID, userID, data)
	return args.String(0), args.Error(1)
}

func (m *MockFormService) GetFormSubmission(ctx context.Context, id string) (*form.FormSubmission, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	result, ok := args.Get(0).(*form.FormSubmission)
	if !ok {
		panic("unexpected type for arg 0 in GetFormSubmission")
	}
	return result, args.Error(1)
}

func (m *MockFormService) ListProjectForms(ctx context.Context, projectID int64) ([]*form.FormSubmission, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	result, ok := args.Get(0).([]*form.FormSubmission)
	if !ok {
		panic("unexpected type for arg 0 in ListProjectForms")
	}
	return result, args.Error(1)
}
