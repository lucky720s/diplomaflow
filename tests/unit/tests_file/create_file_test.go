package tests_file

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	filesvc "github.com/lucky720s/diplomaflow/internal/file"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestUploadFlow_StartCommitResolve(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &filesvc.Config{}
	cfg.Storage.Path = tmpDir
	cfg.Storage.BaseURL = "http://localhost:8080/api/v1"

	svc := filesvc.NewService(cfg, newMemRepo(), logger.New("test"))

	id, tempPath, finalPath, f, err := svc.StartUpload("test.pdf")
	require.NoError(t, err)
	require.NotEmpty(t, id)

	_, err = f.Write([]byte("hello"))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	meta, err := svc.CommitUpload(context.Background(), id, tempPath, finalPath, 123, 456, "test.pdf", ".pdf", "", 5)
	require.NoError(t, err)
	require.NotNil(t, meta)

	_, err = os.Stat(finalPath)
	require.NoError(t, err)

	p, meta, err := svc.ResolveFilePath(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.Equal(t, id, meta.ID)

	require.Equal(t, filepath.Clean(finalPath), filepath.Clean(p))
}
func (m *memRepo) IsFileOnNormControl(ctx context.Context, fileID string) (bool, error) {
	return false, nil
}

func (m *memRepo) IsFileOnAntiplagiat(ctx context.Context, fileID string) (bool, error) {
	return false, nil
}

func (m *memRepo) IsFileOnCommission(ctx context.Context, fileID string) (bool, error) {
	return false, nil
}
