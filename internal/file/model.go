package file

import "time"

type FileMetadata struct {
	ID        string `gorm:"primaryKey"`
	UserID    int64
	ProjectID *int64 `gorm:"column:project_id"`
	FileName  string
	FileType  string // legacy: stores the extension (e.g. ".pdf")
	MimeType  string `gorm:"column:mime_type"` // resolved content type (e.g. "application/pdf")
	Size      int64
	CreatedAt time.Time
}
