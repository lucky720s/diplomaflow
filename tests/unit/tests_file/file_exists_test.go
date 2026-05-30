package tests_file

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	filesvc "github.com/lucky720s/diplomaflow/internal/file"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"github.com/stretchr/testify/require"
)

type memRepo struct {
	m map[string]*filesvc.FileMetadata
}

func newMemRepo() *memRepo {
	return &memRepo{m: make(map[string]*filesvc.FileMetadata)}
}

func (r *memRepo) SaveMetadata(ctx context.Context, meta *filesvc.FileMetadata) error {
	if meta == nil || meta.ID == "" {
		return errors.New("meta/id is required")
	}
	r.m[meta.ID] = meta
	return nil
}

func (r *memRepo) GetMetadata(ctx context.Context, id string) (*filesvc.FileMetadata, error) {
	v, ok := r.m[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	return v, nil
}

func TestService_UploadFlow_AndResolvePath(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &filesvc.Config{}
	cfg.Storage.Path = tmpDir
	cfg.Storage.BaseURL = "http://localhost:8080/api/v1"

	svc := filesvc.NewService(cfg, newMemRepo(), logger.New("test"))

	id, tempPath, finalPath, f, err := svc.StartUpload("test.txt")
	require.NoError(t, err)
	require.NotEmpty(t, id)

	_, err = f.Write([]byte("data"))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	// size=4 bytes, fileType=".txt", mimeType resolved by service
	meta, err := svc.CommitUpload(context.Background(), id, tempPath, finalPath, 1, 2, "test.txt", ".txt", "", 4)
	require.NoError(t, err)
	require.NotNil(t, meta)

	// файл должен существовать
	_, err = os.Stat(finalPath)
	require.NoError(t, err)

	// ResolveFilePath по stable id должен вернуть путь к (id + ext)
	resolvedPath, meta, err := svc.ResolveFilePath(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.Equal(t, id, meta.ID)

	require.Equal(t, filepath.Clean(finalPath), filepath.Clean(resolvedPath))
}
func (r *memRepo) DeleteMetadata(ctx context.Context, id string) error {
	if _, ok := r.m[id]; !ok {
		return os.ErrNotExist
	}
	delete(r.m, id)
	return nil
}

func (r *memRepo) CanAccessViaProject(ctx context.Context, userID, projectID int64) (bool, error) {
	return false, nil
}

func (r *memRepo) IsFileOnNormControl(ctx context.Context, fileID string) (bool, error) {
	return false, nil
}
