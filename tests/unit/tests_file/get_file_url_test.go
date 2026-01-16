package tests_file

import (
	"testing"

	filesvc "github.com/lucky720s/diplomaflow/internal/file"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestService_GetFileURL(t *testing.T) {
	cfg := &filesvc.Config{}
	cfg.Storage.Path = t.TempDir()
	cfg.Storage.BaseURL = "http://test/api/v1"

	svc := filesvc.NewService(cfg, new(MockRepository), logger.New("test"))

	url := svc.GetFileURL("abc123")
	require.Equal(t, "http://test/api/v1/files/abc123", url)
}
