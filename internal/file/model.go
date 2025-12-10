package file

import "time"

type FileMetadata struct {
	ID        string `gorm:"primaryKey"`
	UserID    int64  `gorm:"index"`
	ProjectID int64  `gorm:"index"`
	FileName  string
	FileType  string
	Size      int64
	CreatedAt time.Time
}
