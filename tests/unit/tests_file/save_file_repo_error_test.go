package tests_file

import (
	"context"
	"os"
	"testing"

	filesvc "github.com/lucky720s/diplomaflow/internal/file"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_CommitUpload_MetadataError_RollsBackFile(t *testing.T) {
	tmpDir := t.TempDir()

	repo := new(MockRepository)
	repo.
		On("SaveMetadata", mock.Anything, mock.AnythingOfType("*file.FileMetadata")).
		Return(errors.New("db error"))

	cfg := &filesvc.Config{}
	cfg.Storage.Path = tmpDir
	cfg.Storage.BaseURL = "http://localhost:8080/api/v1"

	svc := filesvc.NewService(cfg, repo, logger.New("test"))

	id, tempPath, finalPath, f, err := svc.StartUpload("test.txt")
	require.NoError(t, err)

	_, err = f.Write([]byte("data"))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	_, err = svc.CommitUpload(context.Background(), id, tempPath, finalPath, 1, 2, "test.txt", ".txt", "", 4)
	require.Error(t, err)

	// должен удалить финальный файл при провале метадаты
	_, statErr := os.Stat(finalPath)
	require.Error(t, statErr)

	repo.AssertExpectations(t)
}
