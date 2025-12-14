package tests_file

import (
	"context"
	"testing"

	domain "github.com/lucky720s/diplomaflow/internal/file"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_SaveFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	repo := new(MockRepository)

	repo.
		On("SaveMetadata", mock.Anything, mock.AnythingOfType("*file.FileMetadata")).
		Return(nil)

	svc := domain.NewTestService(tmpDir, repo, nil)

	name, file, err := svc.SaveFile(
		context.Background(),
		1, 2, "test.txt", "text/plain", 10,
	)

	if file != nil {
		file.Close()
	}

	require.NoError(t, err)
	require.NotNil(t, file)
	require.NotEmpty(t, name)

	repo.AssertExpectations(t)

}
