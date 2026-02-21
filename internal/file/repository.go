package file

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	SaveMetadata(ctx context.Context, meta *FileMetadata) error
	GetMetadata(ctx context.Context, id string) (*FileMetadata, error)
	DeleteMetadata(ctx context.Context, id string) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) SaveMetadata(ctx context.Context, meta *FileMetadata) error {
	if meta.ProjectID != nil && *meta.ProjectID == 0 {
		meta.ProjectID = nil
	}
	return r.db.WithContext(ctx).Create(meta).Error
}

func (r *repository) GetMetadata(ctx context.Context, id string) (*FileMetadata, error) {
	var meta FileMetadata
	if err := r.db.WithContext(ctx).First(&meta, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &meta, nil
}
func (r *repository) DeleteMetadata(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&FileMetadata{}).Error
}
