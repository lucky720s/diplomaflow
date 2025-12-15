package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"go.uber.org/zap"
)

type Service struct {
	storagePath string
	baseURL     string
	repo        Repository
	logger      *logger.Logger
}

func NewService(cfg *Config, repo Repository, log *logger.Logger) *Service {
	if err := os.MkdirAll(cfg.Storage.Path, 0755); err != nil {
		log.Fatal("failed to create storage directory", zap.Error(err))
	}
	return &Service{
		storagePath: cfg.Storage.Path,
		baseURL:     cfg.Storage.BaseURL,
		repo:        repo,
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
func (s *Service) SaveFile(ctx context.Context, userID, projectID int64, fileName, fileType string, size int64) (string, *os.File, error) {
	id := uuid.New().String()
	ext := filepath.Ext(fileName)
	fullName := id + ext

	fullPath := filepath.Join(s.storagePath, fullName)
	file, err := os.Create(fullPath)
	if err != nil {
		return "", nil, err
	}

	meta := &FileMetadata{
		ID:        id,
		UserID:    userID,
		ProjectID: projectID,
		FileName:  fileName,
		FileType:  fileType,
		Size:      size,
		CreatedAt: time.Now(),
	}
	if err := s.repo.SaveMetadata(ctx, meta); err != nil {
		file.Close()
		os.Remove(fullPath)
		return "", nil, err
	}

	return fullName, file, nil
}
func (s *Service) GetMetadata(ctx context.Context, id string) (*FileMetadata, error) {
	return s.repo.GetMetadata(ctx, id)
}

func NewTestService(
	storagePath string,
	repo Repository,
	log *logger.Logger,
) *Service {
	return &Service{
		storagePath: storagePath,
		baseURL:     "http://test",
		repo:        repo,
		logger:      log,
	}
}
