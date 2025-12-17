package tests_form

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/lucky720s/diplomaflow/internal/form"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	formv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/form/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

// Тест СЕРВИСА
func TestService_GetFormSubmission(t *testing.T) {
	repo := new(MockRepository)
	log := logger.New("test")
	svc := form.NewService(repo, log)

	expectedSub := &form.FormSubmission{
		ID:        "uuid-123",
		ProjectID: 1,
	}

	repo.On("GetByID", mock.Anything, "uuid-123").Return(expectedSub, nil)

	res, err := svc.GetFormSubmission(context.Background(), "uuid-123")

	require.NoError(t, err)
	require.Equal(t, expectedSub, res)
}

// Тест ХЕНДЛЕРА
func TestHandler_GetFormSubmission(t *testing.T) {
	mockSvc := new(MockFormService)
	log := logger.New("test")
	handler := form.NewHandler(mockSvc, log)

	rawData := map[string]interface{}{"field": "test"}
	bytes, _ := json.Marshal(rawData)

	submission := &form.FormSubmission{
		ID:        "sub-1",
		ProjectID: 10,
		StepID:    5,
		UserID:    99,
		Data:      datatypes.JSON(bytes),
		CreatedAt: time.Now(),
	}

	mockSvc.On("GetFormSubmission", mock.Anything, "sub-1").
		Return(submission, nil)

	req := &formv1.GetFormSubmissionRequest{SubmissionId: "sub-1"}
	resp, err := handler.GetFormSubmission(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, "sub-1", resp.SubmissionId)
	require.Equal(t, "test", resp.Data.Fields["field"].GetStringValue())
}
