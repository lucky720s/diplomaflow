package tests_file

import (
	"context"
	"testing"

	domain "github.com/lucky720s/diplomaflow/internal/file"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_SaveFile_RepoError(t *testing.T) {
	tmpDir := t.TempDir()
	repo := new(MockRepository)
	svc := domain.NewTestService(tmpDir, repo, nil)

	repo.
		On("SaveMetadata", mock.Anything, mock.Anything).
		Return(errors.New("db error"))

	_, file, err := svc.SaveFile(
		context.Background(),
		1, 2, "test.txt", "text/plain", 10,
	)

	require.Error(t, err)
	require.Nil(t, file)

	repo.AssertExpectations(t)
}
