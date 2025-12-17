package tests_form

import (
	"context"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/form"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	formv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/form/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

// Тест СЕРВИСА для SubmitForm
func TestService_SubmitForm(t *testing.T) {
	repo := new(MockRepository)
	log := logger.New("test")
	svc := form.NewService(repo, log)

	testData := map[string]interface{}{"field1": "value1"}

	repo.On("Create", mock.Anything, mock.MatchedBy(func(s *form.FormSubmission) bool {
		return s.ProjectID == 1 && s.UserID == 10 && s.StepID == 2
	})).Return(nil)

	id, err := svc.SubmitForm(context.Background(), 1, 2, 10, testData)

	require.NoError(t, err)
	require.NotEmpty(t, id)
	repo.AssertExpectations(t)
}

// Тест ХЕНДЛЕРА для SubmitForm
func TestHandler_SubmitForm(t *testing.T) {
	mockSvc := new(MockFormService)
	log := logger.New("test")
	handler := form.NewHandler(mockSvc, log)

	inputData := map[string]interface{}{"key": "val"}
	pbStruct, _ := structpb.NewStruct(inputData)

	mockSvc.On("SubmitForm", mock.Anything, int64(1), int64(2), int64(10), inputData).
		Return("new-uuid", nil)

	req := &formv1.SubmitFormRequest{
		ProjectId: 1,
		StepId:    2,
		UserId:    10,
		Data:      pbStruct,
	}

	resp, err := handler.SubmitForm(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, "new-uuid", resp.SubmissionId)
	require.True(t, resp.Success)
}
