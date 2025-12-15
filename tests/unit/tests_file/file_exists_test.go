package tests_file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/file"
	"github.com/stretchr/testify/require"
)

func TestService_FileExists(t *testing.T) {
	tmpDir := t.TempDir()

	filePath := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(filePath, []byte("data"), 0644)
	require.NoError(t, err)

	svc := file.NewTestService(tmpDir, nil, nil)

	require.True(t, svc.FileExists("test.txt"))
	require.False(t, svc.FileExists("missing.txt"))
}
