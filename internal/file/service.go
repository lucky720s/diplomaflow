package file

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"go.uber.org/zap"
)

type Service struct {
	storagePath string
	baseURL     string
	logger      *logger.Logger
}

func NewService(cfg *Config, log *logger.Logger) *Service {
	if err := os.MkdirAll(cfg.Storage.Path, 0755); err != nil {
		log.Fatal("failed to create storage directory", zap.Error(err))
	}

	return &Service{
		storagePath: cfg.Storage.Path,
		baseURL:     cfg.Storage.BaseURL,
		logger:      log,
	}
}

func (s *Service) CreateFile(ext string) (*os.File, string, error) {
	id := uuid.New().String()
	fileName := id
	if ext != "" {
		fileName = fmt.Sprintf("%s%s", id, ext)
	}

	fullPath := filepath.Join(s.storagePath, fileName)
	file, err := os.Create(fullPath)
	if err != nil {
		return nil, "", err
	}
	return file, fileName, nil
}

func (s *Service) GetFileURL(fileID string) string {
	return fmt.Sprintf("%s/files/%s", s.baseURL, fileID)
}

func (s *Service) FileExists(fileID string) bool {
	fullPath := filepath.Join(s.storagePath, fileID)
	_, err := os.Stat(fullPath)
	return !os.IsNotExist(err)
}
