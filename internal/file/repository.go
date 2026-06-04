package file

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	SaveMetadata(ctx context.Context, meta *FileMetadata) error
	GetMetadata(ctx context.Context, id string) (*FileMetadata, error)
	DeleteMetadata(ctx context.Context, id string) error

	// Access-control lookups (denormalized reads over the shared DB).
	// CanAccessViaProject reports whether the user is the project's student,
	// a member of the project's team, or the team's assigned supervisor.
	CanAccessViaProject(ctx context.Context, userID, projectID int64) (bool, error)
	// IsFileOnNormControl reports whether the file is referenced by a
	// norm-control check (as primary file or in the file_ids set) or by an
	// admin submission's files list — i.e. it was sent for norm-control review.
	IsFileOnNormControl(ctx context.Context, fileID string) (bool, error)
	IsFileOnAntiplagiat(ctx context.Context, fileID string) (bool, error)
	IsFileOnCommission(ctx context.Context, fileID string) (bool, error)
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

func (r *repository) CanAccessViaProject(ctx context.Context, userID, projectID int64) (bool, error) {
	if userID <= 0 || projectID <= 0 {
		return false, nil
	}
	const q = `
		SELECT EXISTS (
			-- project owner (student)
			SELECT 1 FROM projects p WHERE p.id = ? AND p.student_id = ?
			UNION ALL
			-- member of the project's team
			SELECT 1 FROM projects p
			  JOIN team_members tm ON tm.team_id = p.team_id
			 WHERE p.id = ? AND tm.user_id = ?
			UNION ALL
			-- supervisor assigned to the project's team
			SELECT 1 FROM projects p
			  JOIN admin_supervisor_assignments a ON a.team_id = p.team_id
			 WHERE p.id = ? AND a.supervisor_id = ?
		)`
	var ok bool
	if err := r.db.WithContext(ctx).Raw(q,
		projectID, userID,
		projectID, userID,
		projectID, userID,
	).Scan(&ok).Error; err != nil {
		return false, err
	}
	return ok, nil
}

func (r *repository) IsFileOnNormControl(ctx context.Context, fileID string) (bool, error) {
	if fileID == "" {
		return false, nil
	}
	const q = `
		SELECT EXISTS (
			SELECT 1 FROM norm_control_checks c
			 WHERE c.primary_file_id = ?
			    OR c.file_ids @> to_jsonb(?::text)
			UNION ALL
			SELECT 1 FROM admin_submissions s
			 WHERE s.files @> to_jsonb(?::text)
		)`
	var ok bool
	if err := r.db.WithContext(ctx).Raw(q, fileID, fileID, fileID).Scan(&ok).Error; err != nil {
		return false, err
	}
	return ok, nil
}
func (r *repository) IsFileOnAntiplagiat(ctx context.Context, fileID string) (bool, error) {
	if fileID == "" {
		return false, nil
	}

	const q = `
		SELECT EXISTS (
			SELECT 1 FROM antiplag_checks c
			 WHERE c.primary_file_id = ?
			    OR c.file_ids @> to_jsonb(?::text)
			UNION ALL
			SELECT 1 FROM admin_submissions s
			 WHERE s.files @> to_jsonb(?::text)
		)`

	var ok bool
	if err := r.db.WithContext(ctx).Raw(q, fileID, fileID, fileID).Scan(&ok).Error; err != nil {
		return false, err
	}
	return ok, nil
}
func (r *repository) IsFileOnCommission(ctx context.Context, fileID string) (bool, error) {
	if fileID == "" {
		return false, nil
	}

	const q = `
		SELECT EXISTS (
			SELECT 1 FROM admin_pre_defense_documents d
			 WHERE d.id = ?
			UNION ALL
			SELECT 1 FROM admin_submissions s
			 WHERE s.files @> to_jsonb(?::text)
		)`

	var ok bool
	if err := r.db.WithContext(ctx).Raw(q, fileID, fileID).Scan(&ok).Error; err != nil {
		return false, err
	}
	return ok, nil
}
