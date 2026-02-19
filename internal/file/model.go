package file

import "time"

type FileMetadata struct {
	ID        string `gorm:"primaryKey"`
	UserID    int64
	ProjectID *int64 `gorm:"column:project_id"`
	FileName  string
	FileType  string
	Size      int64
	CreatedAt time.Time
}
