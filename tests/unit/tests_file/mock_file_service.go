package tests_file

import (
	"context"

	file "github.com/lucky720s/diplomaflow/internal/file"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) SaveMetadata(ctx context.Context, meta *file.FileMetadata) error {
	args := m.Called(ctx, meta)
	return args.Error(0)
}

func (m *MockRepository) GetMetadata(ctx context.Context, id string) (*file.FileMetadata, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	res, ok := args.Get(0).(*file.FileMetadata)
	if !ok {
		panic("args.Get(0) is not *file.FileMetadata")
	}
	return res, args.Error(1)
}
func (m *MockRepository) DeleteMetadata(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) CanAccessViaProject(ctx context.Context, userID, projectID int64) (bool, error) {
	args := m.Called(ctx, userID, projectID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) IsFileOnNormControl(ctx context.Context, fileID string) (bool, error) {
	args := m.Called(ctx, fileID)
	return args.Bool(0), args.Error(1)
}
func (m *MockRepository) IsFileOnAntiplagiat(ctx context.Context, fileID string) (bool, error) {
	return false, nil
}

func (m *MockRepository) IsFileOnCommission(ctx context.Context, fileID string) (bool, error) {
	return false, nil
}
