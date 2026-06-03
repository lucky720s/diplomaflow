package admin

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AntiplagCheckFilter struct {
	DepartmentID int64
	Status       string
	TeamID       int64
	CheckerID    int64
	Limit        int
	Offset       int
}

func (r *repository) EnsureAntiplagCheckForSubmission(ctx context.Context, submissionID string) (*AntiplagCheck, error) {
	existing, err := r.GetAntiplagCheck(ctx, submissionID)
	if err == nil && existing != nil {
		return existing, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	sub, err := r.GetSubmission(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	// version = max + 1 для того же project+step
	var maxVer int32
	_ = r.db.WithContext(ctx).
		Model(&AntiplagCheck{}).
		Select("COALESCE(MAX(document_version),0)").
		Where("project_id = ? AND step_id = ?", sub.ProjectID, sub.StepID).
		Scan(&maxVer).Error
	version := maxVer + 1

	var fileIDs []string
	_ = json.Unmarshal(sub.Files, &fileIDs)
	var primary *string
	if len(fileIDs) > 0 {
		primary = &fileIDs[0]
	}
	fileIDsJSON, _ := json.Marshal(fileIDs)

	now := time.Now().UTC()

	var teamIDPtr *int64
	if sub.TeamID > 0 {
		t := sub.TeamID
		teamIDPtr = &t
	}
	stepID := sub.StepID
	stepIDPtr := &stepID

	check := &AntiplagCheck{
		SubmissionID:    sub.ID,
		ProjectID:       sub.ProjectID,
		TeamID:          teamIDPtr,
		StepID:          stepIDPtr,
		PrimaryFileID:   primary,
		FileIDs:         fileIDsJSON,
		DocumentVersion: version,
		Status:          AntiplagStatusSubmitted,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "submission_id"}},
		DoNothing: true,
	}).Create(check).Error; err != nil {
		return nil, err
	}

	return r.GetAntiplagCheck(ctx, submissionID)
}

func (r *repository) ListAntiplagChecks(ctx context.Context, filter AntiplagCheckFilter) ([]*AntiplagCheck, int64, error) {
	var list []*AntiplagCheck
	var total int64

	q := r.db.WithContext(ctx).Model(&AntiplagCheck{}).
		Joins("JOIN projects p ON p.id = antiplag_checks.project_id")

	if filter.DepartmentID > 0 {
		q = q.Where("p.department_id = ?", filter.DepartmentID)
	}
	if filter.Status != "" {
		q = q.Where("antiplag_checks.status = ?", filter.Status)
	}
	if filter.TeamID > 0 {
		q = q.Where("antiplag_checks.team_id = ?", filter.TeamID)
	}
	if filter.CheckerID > 0 {
		q = q.Where("antiplag_checks.checker_id = ?", filter.CheckerID)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	err := q.Order("antiplag_checks.created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&list).Error

	return list, total, err
}

func (r *repository) GetAntiplagCheck(ctx context.Context, submissionID string) (*AntiplagCheck, error) {
	var c AntiplagCheck
	err := r.db.WithContext(ctx).First(&c, "submission_id = ?", submissionID).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *repository) UpdateAntiplagCheck(ctx context.Context, check *AntiplagCheck) error {
	check.UpdatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Save(check).Error
}

func (r *repository) ListAntiplagComments(ctx context.Context, submissionID string) ([]*AntiplagComment, error) {
	var comments []*AntiplagComment
	err := r.db.WithContext(ctx).
		Where("submission_id = ?", submissionID).
		Order("created_at ASC").
		Find(&comments).Error
	return comments, err
}

func (r *repository) GetAntiplagComment(ctx context.Context, id string) (*AntiplagComment, error) {
	var c AntiplagComment
	err := r.db.WithContext(ctx).First(&c, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *repository) CreateAntiplagComment(ctx context.Context, c *AntiplagComment) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *repository) UpdateAntiplagComment(ctx context.Context, c *AntiplagComment) error {
	c.UpdatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *repository) DeleteAntiplagComment(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&AntiplagComment{}).Error
}

func (r *repository) AddAntiplagHistory(ctx context.Context, h *AntiplagHistory) error {
	h.CreatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Create(h).Error
}

func (r *repository) ListAntiplagHistory(ctx context.Context, projectID int64) ([]*AntiplagHistory, error) {
	var hist []*AntiplagHistory
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("created_at DESC").
		Find(&hist).Error
	return hist, err
}
