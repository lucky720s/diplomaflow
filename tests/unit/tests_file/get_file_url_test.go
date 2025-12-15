package tests_file

import (
	"testing"

	"github.com/lucky720s/diplomaflow/internal/file"
	"github.com/stretchr/testify/require"
)

func TestService_GetFileURL(t *testing.T) {
	svc := file.NewTestService(t.TempDir(), nil, nil)

	url := svc.GetFileURL("abc123")

	require.Equal(t, "http://test/files/abc123", url)
}
