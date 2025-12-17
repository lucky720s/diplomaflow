package tests_form

import (
	"context"
	"testing"
	"time"

	"github.com/lucky720s/diplomaflow/internal/form"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	formv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/form/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Тест СЕРВИСА
func TestService_ListProjectForms(t *testing.T) {
	repo := new(MockRepository)
	log := logger.New("test")
	svc := form.NewService(repo, log)

	list := []*form.FormSubmission{
		{ID: "1", CreatedAt: time.Now()},
		{ID: "2", CreatedAt: time.Now()},
	}

	repo.On("ListByProject", mock.Anything, int64(100)).Return(list, nil)

	res, err := svc.ListProjectForms(context.Background(), 100)

	require.NoError(t, err)
	require.Len(t, res, 2)
}

// Тест ХЕНДЛЕРА
func TestHandler_ListProjectForms(t *testing.T) {
	mockSvc := new(MockFormService)
	log := logger.New("test")
	handler := form.NewHandler(mockSvc, log)

	list := []*form.FormSubmission{
		{ID: "1", StepID: 1, CreatedAt: time.Now()},
		{ID: "2", StepID: 2, CreatedAt: time.Now()},
	}

	mockSvc.On("ListProjectForms", mock.Anything, int64(100)).
		Return(list, nil)

	req := &formv1.ListProjectFormsRequest{ProjectId: 100}
	resp, err := handler.ListProjectForms(context.Background(), req)

	require.NoError(t, err)
	require.Len(t, resp.Forms, 2)
	require.Equal(t, "1", resp.Forms[0].SubmissionId)
}
