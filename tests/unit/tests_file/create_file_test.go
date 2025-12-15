package tests_file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/file"
	"github.com/stretchr/testify/require"
)

func TestService_CreateFile(t *testing.T) {
	tmpDir := t.TempDir()

	svc := file.NewTestService(tmpDir, nil, nil)

	file, name, err := svc.CreateFile(".txt")

	if file != nil {
		file.Close()
	}

	require.NoError(t, err)
	require.NotNil(t, file)
	require.NotEmpty(t, name)

	_, err = os.Stat(filepath.Join(tmpDir, name))
	require.NoError(t, err)
}
