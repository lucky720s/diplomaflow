package tests_file

import (
	"context"

	"github.com/lucky720s/diplomaflow/internal/file"
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
	return args.Get(0).(*file.FileMetadata), args.Error(1)
}
