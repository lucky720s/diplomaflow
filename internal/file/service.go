package file

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	filev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/file/v1"
	"go.uber.org/zap"
)

type Service struct {
	storagePath    string
	baseURL        string
	publicPath     string
	maxUploadBytes int64
	repo           Repository
	logger         *logger.Logger
}

func NewService(cfg *Config, repo Repository, log *logger.Logger) *Service {
	if err := os.MkdirAll(cfg.Storage.Path, 0755); err != nil {
		log.Fatal("failed to create storage directory", zap.Error(err))
	}
	return &Service{
		storagePath:    cfg.Storage.Path,
		baseURL:        cfg.Storage.BaseURL,
		publicPath:     publicBasePath(cfg.Storage.BaseURL),
		maxUploadBytes: cfg.MaxUploadBytes(),
		repo:           repo,
		logger:         log,
	}
}

// MaxUploadBytes exposes the configured upload size limit to the handler.
func (s *Service) MaxUploadBytes() int64 { return s.maxUploadBytes }

// publicBasePath reduces a possibly-absolute base URL to a relative path prefix
// (e.g. "http://host:8080/api/v1" -> "/api/v1"). Relative URLs keep the unified
// file contract correct behind an external reverse proxy. Defaults to "/api/v1".
func publicBasePath(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "/api/v1"
	}
	if u, err := url.Parse(baseURL); err == nil && u.Path != "" {
		return "/" + strings.Trim(u.Path, "/")
	}
	return "/" + strings.Trim(baseURL, "/")
}

// GetFileURL returns gateway URL (public, absolute) — kept for the legacy
// GetFileInfoResponse.download_url field.
func (s *Service) GetFileURL(id string) string {
	return fmt.Sprintf("%s/files/%s", strings.TrimRight(s.baseURL, "/"), id)
}

// BuildFileRef assembles the unified file contract for a stored file.
// Returns nil when meta is nil (no file / legacy id) so callers emit a clean
// empty file state instead of malformed URLs.
func (s *Service) BuildFileRef(meta *FileMetadata) *filev1.FileRef {
	if meta == nil {
		return nil
	}
	base := fmt.Sprintf("%s/files/%s", strings.TrimRight(s.publicPath, "/"), meta.ID)
	return &filev1.FileRef{
		FileId:      meta.ID,
		FileName:    meta.FileName,
		FileType:    metaMIME(meta),
		Size:        meta.Size,
		FileUrl:     base,
		DownloadUrl: base + "?download=1",
		PreviewUrl:  base + "?disposition=inline",
	}
}

// metaMIME returns the stored MIME type, falling back to the extension when the
// column is empty (e.g. rows created before the mime_type backfill).
func metaMIME(meta *FileMetadata) string {
	if meta.MimeType != "" {
		return meta.MimeType
	}
	if byExt := mime.TypeByExtension(strings.ToLower(filepath.Ext(meta.FileName))); byExt != "" {
		return byExt
	}
	return "application/octet-stream"
}

// officeMIMEByExt returns the canonical MIME type for Office formats, which
// http.DetectContentType cannot distinguish (it reports application/zip for
// OOXML and a generic type for the older binary formats).
func officeMIMEByExt(ext string) (string, bool) {
	switch strings.ToLower(ext) {
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document", true
	case ".doc":
		return "application/msword", true
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", true
	case ".xls":
		return "application/vnd.ms-excel", true
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation", true
	case ".ppt":
		return "application/vnd.ms-powerpoint", true
	}
	return "", false
}

// DetectMIME resolves a content type for a stored file: sniff the first 512
// bytes, then apply an explicit Office-extension override (sniff yields
// application/zip for OOXML), then fall back to the extension table.
func DetectMIME(path, fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))

	sniffed := ""
	if f, err := os.Open(path); err == nil {
		defer f.Close()
		buf := make([]byte, 512)
		if n, _ := f.Read(buf); n > 0 {
			sniffed = http.DetectContentType(buf[:n])
		}
	}

	// Office formats must be resolved by extension regardless of sniff result.
	if office, ok := officeMIMEByExt(ext); ok {
		return office
	}

	if sniffed != "" && sniffed != "application/octet-stream" {
		return sniffed
	}
	if byExt := mime.TypeByExtension(ext); byExt != "" {
		return byExt
	}
	return "application/octet-stream"
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
	userID, projectID int64, originalName, fileType, mimeType string, size int64) (*FileMetadata, error) {

	if err := os.Rename(tempPath, finalPath); err != nil {
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("rename temp to final: %w", err)
	}

	// Resolve MIME from the committed file if the caller did not supply one.
	if mimeType == "" {
		mimeType = DetectMIME(finalPath, originalName)
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
		MimeType:  mimeType,
		Size:      size,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.SaveMetadata(ctx, meta); err != nil {
		_ = os.Remove(finalPath)
		return nil, fmt.Errorf("save metadata: %w", err)
	}
	return meta, nil
}

// CanAccessFile is the centralized authorization check for a stable file id.
// Order: admin → owner/uploader → project relation (student/team member/
// supervisor) → norm_control role for documents sent to norm-control.
// Legacy ids (containing a dot) are never accessible to regular users here;
// internal callers bypass this check at the handler layer.
func (s *Service) CanAccessFile(ctx context.Context, userID int64, role, fileID string) (bool, error) {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return false, errors.New("file id is required")
	}
	if strings.Contains(fileID, ".") {
		return false, nil // legacy stored filename
	}
	meta, err := s.repo.GetMetadata(ctx, fileID)
	if err != nil {
		return false, err
	}
	return s.canAccessWithMeta(ctx, userID, role, fileID, meta)
}

// canAccessWithMeta applies the access rules against already-resolved metadata
// to avoid a redundant lookup when the caller already has it.
func (s *Service) canAccessWithMeta(ctx context.Context, userID int64, role, fileID string, meta *FileMetadata) (bool, error) {
	if meta == nil {
		return false, nil
	}
	if role == "admin" {
		return true, nil
	}
	if meta.UserID == userID && userID > 0 {
		return true, nil
	}
	if meta.ProjectID != nil && *meta.ProjectID > 0 {
		ok, err := s.repo.CanAccessViaProject(ctx, userID, *meta.ProjectID)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	if role == "norm_control" {
		ok, err := s.repo.IsFileOnNormControl(ctx, fileID)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
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
func (s *Service) DeleteFile(ctx context.Context, id string, callerUserID int64) error {
	meta, err := s.repo.GetMetadata(ctx, id)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}
	if callerUserID <= 0 {
		return fmt.Errorf("unauthorized")
	}
	if meta.UserID != callerUserID {
		return fmt.Errorf("permission denied")
	}

	ext := filepath.Ext(meta.FileName)
	filePath := filepath.Join(s.storagePath, meta.ID+ext)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		s.logger.Warn("failed to remove file from disk", zap.Error(err))
	}

	tmpPath := filePath + ".tmp"
	_ = os.Remove(tmpPath)
	if err := s.repo.DeleteMetadata(ctx, id); err != nil {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}

	return nil
}

// service.go — добавить:
func (s *Service) CleanupOrphanedTempFiles(maxAge time.Duration) int {
	entries, err := os.ReadDir(s.storagePath)
	if err != nil {
		s.logger.Error("failed to read storage dir", zap.Error(err))
		return 0
	}

	cleaned := 0
	cutoff := time.Now().Add(-maxAge)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tmp") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(s.storagePath, entry.Name())
			if err := os.Remove(path); err == nil {
				cleaned++
				s.logger.Info("Cleaned orphaned temp file", zap.String("file", entry.Name()))
			}
		}
	}
	return cleaned
}
