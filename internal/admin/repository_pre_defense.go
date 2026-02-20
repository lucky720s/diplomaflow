package admin

import (
	"context"
	"time"
)

// PreDefenseFilter - фильтр для списка предзащит
type PreDefenseFilter struct {
	DepartmentID int64
	TeamID       int64
	SupervisorID int64
	Status       string
	DateFrom     *time.Time
	DateTo       *time.Time
	Limit        int
	Offset       int
}

// ScheduleFilter - фильтр для расписания
type ScheduleFilter struct {
	CommissionMemberID int64
	DepartmentID       int64
	Date               *time.Time
	DateFrom           *time.Time
	DateTo             *time.Time
	Location           string
}

// ==================== Pre-Defense Submissions ====================

func (r *repository) CreatePreDefenseSubmission(ctx context.Context, sub *PreDefenseSubmission) error {
	return r.db.WithContext(ctx).Create(sub).Error
}

func (r *repository) GetPreDefenseSubmission(ctx context.Context, id string) (*PreDefenseSubmission, error) {
	var sub PreDefenseSubmission
	if err := r.db.WithContext(ctx).First(&sub, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *repository) UpdatePreDefenseSubmission(ctx context.Context, sub *PreDefenseSubmission) error {
	sub.UpdatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Save(sub).Error
}

func (r *repository) ListPreDefenseSubmissions(ctx context.Context, filter PreDefenseFilter) ([]*PreDefenseSubmission, int64, error) {
	var subs []*PreDefenseSubmission
	var total int64

	query := r.db.WithContext(ctx).Model(&PreDefenseSubmission{})

	if filter.TeamID > 0 {
		query = query.Where("team_id = ?", filter.TeamID)
	}
	if filter.SupervisorID > 0 {
		query = query.Where("supervisor_id = ?", filter.SupervisorID)
	}
	if filter.Status != "" && filter.Status != "all" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.DepartmentID > 0 {
		query = query.Joins("JOIN projects p ON p.id = admin_pre_defense_submissions.project_id").
			Where("p.department_id = ?", filter.DepartmentID)
	}
	if filter.DateFrom != nil {
		query = query.Where("scheduled_date >= ?", filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("scheduled_date <= ?", filter.DateTo)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	err := query.
		Order("created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&subs).Error

	return subs, total, err
}

func (r *repository) ListScheduledPreDefenses(ctx context.Context, filter ScheduleFilter) ([]*PreDefenseSubmission, error) {
	var subs []*PreDefenseSubmission

	query := r.db.WithContext(ctx).Model(&PreDefenseSubmission{}).
		Where("status IN ?", []string{"scheduled", "in_progress"})

	if filter.CommissionMemberID > 0 {
		query = query.
			Joins("JOIN admin_pre_defense_commission c ON c.submission_id = admin_pre_defense_submissions.id").
			Where("c.user_id = ?", filter.CommissionMemberID)
	}

	if filter.DepartmentID > 0 {
		query = query.
			Joins("JOIN projects p ON p.id = admin_pre_defense_submissions.project_id").
			Where("p.department_id = ?", filter.DepartmentID)
	}

	if filter.Date != nil {
		query = query.Where("scheduled_date = ?", filter.Date)
	}
	if filter.DateFrom != nil {
		query = query.Where("scheduled_date >= ?", filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("scheduled_date <= ?", filter.DateTo)
	}
	if filter.Location != "" {
		query = query.Where("location = ?", filter.Location)
	}

	err := query.
		Order("scheduled_date ASC, scheduled_time ASC").
		Find(&subs).Error

	return subs, err
}

// ==================== Commission Members ====================

func (r *repository) AddCommissionMember(ctx context.Context, member *PreDefenseCommissionMember) error {
	member.CreatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Create(member).Error
}

func (r *repository) RemoveCommissionMember(ctx context.Context, submissionID string, userID int64) error {
	return r.db.WithContext(ctx).
		Where("submission_id = ? AND user_id = ?", submissionID, userID).
		Delete(&PreDefenseCommissionMember{}).Error
}

func (r *repository) GetCommissionMembers(ctx context.Context, submissionID string) ([]PreDefenseCommissionMember, error) {
	var members []PreDefenseCommissionMember
	err := r.db.WithContext(ctx).
		Where("submission_id = ?", submissionID).
		Order("role ASC, created_at ASC").
		Find(&members).Error
	return members, err
}

func (r *repository) UpdateCommissionMemberGrade(ctx context.Context, submissionID string, userID int64, grade int32, comment string) error {
	return r.db.WithContext(ctx).
		Model(&PreDefenseCommissionMember{}).
		Where("submission_id = ? AND user_id = ?", submissionID, userID).
		Updates(map[string]interface{}{
			"individual_grade": grade,
			"comment":          comment,
			"is_present":       true,
		}).Error
}

// ==================== Documents ====================

func (r *repository) AddPreDefenseDocument(ctx context.Context, doc *PreDefenseDocument) error {
	return r.db.WithContext(ctx).Create(doc).Error
}

func (r *repository) GetPreDefenseDocuments(ctx context.Context, submissionID string) ([]PreDefenseDocument, error) {
	var docs []PreDefenseDocument
	err := r.db.WithContext(ctx).
		Where("submission_id = ?", submissionID).
		Find(&docs).Error
	return docs, err
}

// ==================== History ====================

func (r *repository) AddPreDefenseHistory(ctx context.Context, entry *PreDefenseHistory) error {
	entry.CreatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *repository) GetPreDefenseHistory(ctx context.Context, submissionID string) ([]*PreDefenseHistory, error) {
	var entries []*PreDefenseHistory
	err := r.db.WithContext(ctx).
		Where("submission_id = ?", submissionID).
		Order("created_at DESC").
		Find(&entries).Error
	return entries, err
}
