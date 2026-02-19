package file

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// GetFileURL returns gateway URL (public).
func (s *Service) GetFileURL(id string) string {
	return fmt.Sprintf("%s/files/%s", strings.TrimRight(s.baseURL, "/"), id)
}

// StartUpload creates temp file for upload and returns stable id + temp file handle.
// fileName used for extension extraction.
func (s *Service) StartUpload(fileName string) (id string, tempPath string, finalPath string, f *os.File, err error) {
	id = uuid.New().String()
	ext := filepath.Ext(fileName) // includes dot
	// If no extension, keep ext empty
	finalName := id + ext
	tempName := finalName + ".tmp"

	tempPath = filepath.Join(s.storagePath, tempName)
	finalPath = filepath.Join(s.storagePath, finalName)

	f, err = os.Create(tempPath)
	if err != nil {
		return "", "", "", nil, err
	}
	return id, tempPath, finalPath, f, nil
}

func (s *Service) CommitUpload(ctx context.Context, id, tempPath, finalPath string,
	userID, projectID int64, originalName, fileType string, size int64) error {

	if err := os.Rename(tempPath, finalPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("rename temp to final: %w", err)
	}

	// Конвертируем 0 → nil
	var pid *int64
	if projectID != 0 {
		pid = &projectID
	}

	meta := &FileMetadata{
		ID:        id,
		UserID:    userID,
		ProjectID: pid,
		FileName:  originalName,
		FileType:  fileType,
		Size:      size,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.SaveMetadata(ctx, meta); err != nil {
		_ = os.Remove(finalPath)
		return fmt.Errorf("save metadata: %w", err)
	}
	return nil
}

// ResolveFilePath supports:
// 1) stable id (uuid) -> metadata -> id+ext
// 2) legacy: id contains dot => treat as stored filename
func (s *Service) ResolveFilePath(ctx context.Context, id string) (path string, meta *FileMetadata, err error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", nil, errors.New("id is required")
	}

	// Legacy: looks like "uuid.ext"
	if strings.Contains(id, ".") {
		p := filepath.Join(s.storagePath, filepath.Base(id))
		if _, statErr := os.Stat(p); statErr != nil {
			return "", nil, statErr
		}
		return p, nil, nil
	}

	// Stable id: resolve by metadata
	m, err := s.repo.GetMetadata(ctx, id)
	if err != nil {
		return "", nil, err
	}
	ext := filepath.Ext(m.FileName)
	p := filepath.Join(s.storagePath, m.ID+ext)
	if _, statErr := os.Stat(p); statErr != nil {
		return "", nil, statErr
	}
	return p, m, nil
}
