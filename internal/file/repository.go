package file

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	SaveMetadata(ctx context.Context, meta *FileMetadata) error
	GetMetadata(ctx context.Context, id string) (*FileMetadata, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	_ = db.AutoMigrate(&FileMetadata{})
	return &repository{db: db}
}

func (r *repository) SaveMetadata(ctx context.Context, meta *FileMetadata) error {
	return r.db.WithContext(ctx).Create(meta).Error
}

func (r *repository) GetMetadata(ctx context.Context, id string) (*FileMetadata, error) {
	var meta FileMetadata
	if err := r.db.WithContext(ctx).First(&meta, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &meta, nil
}
