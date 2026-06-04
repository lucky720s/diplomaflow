package admin

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NormCheckFilter struct {
	DepartmentID int64
	Status       string
	TeamID       int64
	CheckerID    int64
	Limit        int
	Offset       int
}

func (r *repository) EnsureNormCheckForSubmission(ctx context.Context, submissionID string) (*NormControlCheck, error) {
	// If exists -> return
	existing, err := r.GetNormCheck(ctx, submissionID)
	if err == nil && existing != nil {
		return existing, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// Load submission
	sub, err := r.GetSubmission(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	// Create only for NORM_CONTROL step submissions
	// (state name is not stored on submission, but step_id is; we still create check row and rely on service validation)
	// Determine version: max + 1 for same project+step
	var maxVer int32
	_ = r.db.WithContext(ctx).
		Model(&NormControlCheck{}).
		Select("COALESCE(MAX(document_version),0)").
		Where("project_id = ? AND step_id = ?", sub.ProjectID, sub.StepID).
		Scan(&maxVer).Error

	version := maxVer + 1

	// Parse files (stored as JSON array of strings)
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

	check := &NormControlCheck{
		SubmissionID:    sub.ID,
		ProjectID:       sub.ProjectID,
		TeamID:          teamIDPtr,
		StepID:          stepIDPtr,
		PrimaryFileID:   primary,
		FileIDs:         fileIDsJSON,
		DocumentVersion: version,
		Status:          NormStatusSubmitted,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Upsert by submission_id (in case of race)
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "submission_id"}},
		DoNothing: true,
	}).Create(check).Error; err != nil {
		return nil, err
	}

	return r.GetNormCheck(ctx, submissionID)
}
func (r *repository) ListNormChecks(ctx context.Context, filter NormCheckFilter) ([]*NormControlCheck, int64, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	args := make([]interface{}, 0)
	where := "WHERE 1=1"

	if filter.DepartmentID > 0 {
		where += " AND p.department_id = ?"
		args = append(args, filter.DepartmentID)
	}

	if filter.TeamID > 0 {
		where += " AND nc.team_id = ?"
		args = append(args, filter.TeamID)
	}

	if filter.CheckerID > 0 {
		where += " AND nc.checker_id = ?"
		args = append(args, filter.CheckerID)
	}

	if filter.Status != "" {
		where += " AND nc.status = ?"
		args = append(args, filter.Status)
	} else {
		// Главное исправление:
		// pending endpoint по умолчанию возвращает только активные проверки.
		where += " AND nc.status IN ('submitted', 'in_review')"
	}

	countSQL := `
		SELECT COUNT(*) FROM (
			SELECT DISTINCT ON (nc.project_id, nc.step_id)
				nc.submission_id
			FROM norm_control_checks nc
			JOIN projects p ON p.id = nc.project_id
			` + where + `
			ORDER BY nc.project_id, nc.step_id, nc.created_at DESC
		) x
	`

	var total int64
	if err := r.db.WithContext(ctx).Raw(countSQL, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	listSQL := `
		SELECT *
		FROM (
			SELECT DISTINCT ON (nc.project_id, nc.step_id)
				nc.*
			FROM norm_control_checks nc
			JOIN projects p ON p.id = nc.project_id
			` + where + `
			ORDER BY nc.project_id, nc.step_id, nc.created_at DESC
		) x
		ORDER BY x.created_at DESC
		LIMIT ? OFFSET ?
	`

	listArgs := append(args, limit, offset)

	var checks []*NormControlCheck
	if err := r.db.WithContext(ctx).Raw(listSQL, listArgs...).Scan(&checks).Error; err != nil {
		return nil, 0, err
	}

	return checks, total, nil
}
func (r *repository) GetNormCheck(ctx context.Context, submissionID string) (*NormControlCheck, error) {
	var c NormControlCheck
	err := r.db.WithContext(ctx).First(&c, "submission_id = ?", submissionID).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *repository) UpdateNormCheck(ctx context.Context, check *NormControlCheck) error {
	check.UpdatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Save(check).Error
}

func (r *repository) ListNormIssues(ctx context.Context, submissionID string) ([]*NormControlIssue, error) {
	var issues []*NormControlIssue
	err := r.db.WithContext(ctx).
		Where("submission_id = ?", submissionID).
		Order("created_at ASC").
		Find(&issues).Error
	return issues, err
}

func (r *repository) GetNormIssue(ctx context.Context, id string) (*NormControlIssue, error) {
	var issue NormControlIssue
	err := r.db.WithContext(ctx).First(&issue, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &issue, nil
}

func (r *repository) CreateNormIssue(ctx context.Context, issue *NormControlIssue) error {
	if issue.ID == "" {
		issue.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	issue.CreatedAt = now
	issue.UpdatedAt = now
	return r.db.WithContext(ctx).Create(issue).Error
}

func (r *repository) UpdateNormIssue(ctx context.Context, issue *NormControlIssue) error {
	issue.UpdatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Save(issue).Error
}

func (r *repository) DeleteNormIssue(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&NormControlIssue{}).Error
}

func (r *repository) CountUnresolvedCriticalIssues(ctx context.Context, submissionID string) (int64, error) {
	var cnt int64
	err := r.db.WithContext(ctx).
		Model(&NormControlIssue{}).
		Where("submission_id = ? AND severity = ? AND is_resolved = false", submissionID, "critical").
		Count(&cnt).Error
	return cnt, err
}

func (r *repository) AddNormHistory(ctx context.Context, h *NormControlHistory) error {
	h.CreatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Create(h).Error
}

func (r *repository) ListNormHistory(ctx context.Context, projectID int64) ([]*NormControlHistory, error) {
	var hist []*NormControlHistory
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("created_at DESC").
		Find(&hist).Error
	return hist, err
}

func (r *repository) ListNormChecklists(ctx context.Context) ([]*NormControlChecklist, error) {
	var list []*NormControlChecklist
	err := r.db.WithContext(ctx).
		Where("is_active = true").
		Order("created_at DESC").
		Find(&list).Error
	return list, err
}

func (r *repository) CreateNormChecklist(ctx context.Context, c *NormControlChecklist) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *repository) StatsNormIssuesByCategory(ctx context.Context, departmentID int64) (map[string]int64, error) {
	type row struct {
		Category string `gorm:"column:category"`
		Cnt      int64  `gorm:"column:cnt"`
	}

	var rows []row
	err := r.db.WithContext(ctx).
		Table("norm_control_issues i").
		Select("i.category as category, COUNT(*) as cnt").
		Joins("JOIN norm_control_checks c ON c.submission_id = i.submission_id").
		Joins("JOIN projects p ON p.id = c.project_id").
		Where("p.department_id = ?", departmentID).
		Group("i.category").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		out[r.Category] = r.Cnt
	}
	return out, nil
}
