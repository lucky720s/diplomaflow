package admin

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	// Grades
	CreateGrade(ctx context.Context, grade *Grade) error
	GetGrade(ctx context.Context, projectID, stepID int64) (*Grade, error)
	GetGradesByProject(ctx context.Context, projectID int64) ([]*Grade, error)
	UpdateGrade(ctx context.Context, grade *Grade) error
	CreateGradeHistory(ctx context.Context, history *GradeHistory) error
	GetGradeHistory(ctx context.Context, projectID, stepID int64) ([]*GradeHistory, error)

	// Submissions
	CreateSubmission(ctx context.Context, sub *Submission) error
	GetSubmission(ctx context.Context, id string) (*Submission, error)
	ListSubmissions(ctx context.Context, filter SubmissionFilter) ([]*Submission, int64, error)
	UpdateSubmission(ctx context.Context, sub *Submission) error
	CreateSubmissionReview(ctx context.Context, review *SubmissionReview) error
	GetSubmissionReviews(ctx context.Context, submissionID string) ([]*SubmissionReview, error)

	// Supervisor Assignments
	AssignSupervisor(ctx context.Context, assignment *SupervisorAssignment) error
	GetSupervisorAssignment(ctx context.Context, teamID int64) (*SupervisorAssignment, error)
	UpdateSupervisorAssignment(ctx context.Context, assignment *SupervisorAssignment) error
	GetTeamsBySupervisor(ctx context.Context, supervisorID int64) ([]int64, error)
	CountTeamsBySupervisor(ctx context.Context, supervisorID int64) (int64, error)

	// Activities
	LogActivity(ctx context.Context, activity *AdminActivity) error
	GetRecentActivities(ctx context.Context, departmentID int64, limit int) ([]*AdminActivity, error)

	// Dashboard Stats (cross-table queries)
	GetDashboardStats(ctx context.Context, departmentID int64) (*DashboardStatsData, error)
	GetStepProgressStats(ctx context.Context, departmentID int64, workflowID int64) ([]*StepProgressData, error)
	GetPendingReviewsCount(ctx context.Context, departmentID int64) (int64, error)
}

// Filter structs
type SubmissionFilter struct {
	DepartmentID int64
	StepID       int64
	TeamID       int64
	Status       string
	Limit        int
	Offset       int
}

// Stats data structs
type DashboardStatsData struct {
	TotalStudents     int32
	TotalTeams        int32
	TotalProjects     int32
	CompletedProjects int32
	PendingReviews    int32
	ActiveSupervisors int32
}

type StepProgressData struct {
	StepID         int64
	StepName       string
	StepType       string
	TotalTeams     int32
	CompletedTeams int32
	PendingTeams   int32
	RejectedTeams  int32
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	// Auto-migrate admin tables
	_ = db.AutoMigrate(
		&Grade{},
		&GradeHistory{},
		&Submission{},
		&SubmissionReview{},
		&SupervisorAssignment{},
		&AdminActivity{},
	)
	return &repository{db: db}
}

// ==================== Grades ====================

func (r *repository) CreateGrade(ctx context.Context, grade *Grade) error {
	grade.LetterGrade = CalculateLetterGrade(grade.Grade)
	grade.CreatedAt = time.Now()
	grade.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(grade).Error
}

func (r *repository) GetGrade(ctx context.Context, projectID, stepID int64) (*Grade, error) {
	var grade Grade
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND step_id = ?", projectID, stepID).
		First(&grade).Error
	if err != nil {
		return nil, err
	}
	return &grade, nil
}

func (r *repository) GetGradesByProject(ctx context.Context, projectID int64) ([]*Grade, error) {
	var grades []*Grade
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("step_id ASC").
		Find(&grades).Error
	return grades, err
}

func (r *repository) UpdateGrade(ctx context.Context, grade *Grade) error {
	grade.LetterGrade = CalculateLetterGrade(grade.Grade)
	grade.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(grade).Error
}

func (r *repository) CreateGradeHistory(ctx context.Context, history *GradeHistory) error {
	history.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(history).Error
}

func (r *repository) GetGradeHistory(ctx context.Context, projectID, stepID int64) ([]*GradeHistory, error) {
	var history []*GradeHistory
	query := r.db.WithContext(ctx).Where("project_id = ?", projectID)
	if stepID > 0 {
		query = query.Where("step_id = ?", stepID)
	}
	err := query.Order("created_at DESC").Find(&history).Error
	return history, err
}

// ==================== Submissions ====================

func (r *repository) CreateSubmission(ctx context.Context, sub *Submission) error {
	sub.CreatedAt = time.Now()
	sub.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(sub).Error
}

func (r *repository) GetSubmission(ctx context.Context, id string) (*Submission, error) {
	var sub Submission
	err := r.db.WithContext(ctx).First(&sub, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *repository) ListSubmissions(ctx context.Context, filter SubmissionFilter) ([]*Submission, int64, error) {
	var submissions []*Submission
	var total int64

	query := r.db.WithContext(ctx).Model(&Submission{})

	if filter.StepID > 0 {
		query = query.Where("step_id = ?", filter.StepID)
	}
	if filter.TeamID > 0 {
		query = query.Where("team_id = ?", filter.TeamID)
	}
	if filter.Status != "" && filter.Status != "all" {
		query = query.Where("status = ?", filter.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Order("created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&submissions).Error

	return submissions, total, err
}

func (r *repository) UpdateSubmission(ctx context.Context, sub *Submission) error {
	sub.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(sub).Error
}

func (r *repository) CreateSubmissionReview(ctx context.Context, review *SubmissionReview) error {
	review.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(review).Error
}

func (r *repository) GetSubmissionReviews(ctx context.Context, submissionID string) ([]*SubmissionReview, error) {
	var reviews []*SubmissionReview
	err := r.db.WithContext(ctx).
		Where("submission_id = ?", submissionID).
		Order("created_at DESC").
		Find(&reviews).Error
	return reviews, err
}

// ==================== Supervisor Assignments ====================

func (r *repository) AssignSupervisor(ctx context.Context, assignment *SupervisorAssignment) error {
	assignment.CreatedAt = time.Now()
	assignment.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(assignment).Error
}

func (r *repository) GetSupervisorAssignment(ctx context.Context, teamID int64) (*SupervisorAssignment, error) {
	var assignment SupervisorAssignment
	err := r.db.WithContext(ctx).Where("team_id = ?", teamID).First(&assignment).Error
	if err != nil {
		return nil, err
	}
	return &assignment, nil
}

func (r *repository) UpdateSupervisorAssignment(ctx context.Context, assignment *SupervisorAssignment) error {
	assignment.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(assignment).Error
}

func (r *repository) GetTeamsBySupervisor(ctx context.Context, supervisorID int64) ([]int64, error) {
	var teamIDs []int64
	err := r.db.WithContext(ctx).
		Model(&SupervisorAssignment{}).
		Where("supervisor_id = ?", supervisorID).
		Pluck("team_id", &teamIDs).Error
	return teamIDs, err
}

func (r *repository) CountTeamsBySupervisor(ctx context.Context, supervisorID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&SupervisorAssignment{}).
		Where("supervisor_id = ?", supervisorID).
		Count(&count).Error
	return count, err
}

// ==================== Activities ====================

func (r *repository) LogActivity(ctx context.Context, activity *AdminActivity) error {
	activity.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(activity).Error
}

func (r *repository) GetRecentActivities(ctx context.Context, departmentID int64, limit int) ([]*AdminActivity, error) {
	var activities []*AdminActivity
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Find(&activities).Error
	return activities, err
}

// ==================== Dashboard Stats ====================

func (r *repository) GetDashboardStats(ctx context.Context, departmentID int64) (*DashboardStatsData, error) {
	stats := &DashboardStatsData{}

	// Используем временные переменные int64 для Count()
	var totalStudents, totalTeams, totalProjects, completedProjects, pendingReviews, activeSupervisors int64

	// Count students
	r.db.WithContext(ctx).
		Table("users").
		Where("role = ? AND department_id = ?", "student", departmentID).
		Count(&totalStudents)
	stats.TotalStudents = int32(totalStudents)

	// Count teams
	r.db.WithContext(ctx).
		Table("teams").
		Joins("JOIN projects ON teams.project_id = projects.id").
		Where("projects.department_id = ?", departmentID).
		Count(&totalTeams)
	stats.TotalTeams = int32(totalTeams)

	// Count projects
	r.db.WithContext(ctx).
		Table("projects").
		Where("department_id = ?", departmentID).
		Count(&totalProjects)
	stats.TotalProjects = int32(totalProjects)

	// Count completed projects
	r.db.WithContext(ctx).
		Table("projects").
		Where("department_id = ? AND status = ?", departmentID, "completed").
		Count(&completedProjects)
	stats.CompletedProjects = int32(completedProjects)

	// Count pending reviews
	r.db.WithContext(ctx).
		Table("admin_submissions").
		Where("status = ?", SubmissionStatusPending).
		Count(&pendingReviews)
	stats.PendingReviews = int32(pendingReviews)

	// Count active supervisors
	r.db.WithContext(ctx).
		Table("admin_supervisor_assignments").
		Distinct("supervisor_id").
		Count(&activeSupervisors)
	stats.ActiveSupervisors = int32(activeSupervisors)

	return stats, nil
}

func (r *repository) GetStepProgressStats(ctx context.Context, departmentID int64, workflowID int64) ([]*StepProgressData, error) {
	var stats []*StepProgressData

	// This is a simplified query - in production, you'd want more complex joins
	query := `
		SELECT 
			s.id as step_id,
			s.name as step_name,
			s.type as step_type,
			COUNT(DISTINCT p.id) as total_teams,
			COUNT(DISTINCT CASE WHEN sub.status = 'approved' THEN p.id END) as completed_teams,
			COUNT(DISTINCT CASE WHEN sub.status = 'pending' THEN p.id END) as pending_teams,
			COUNT(DISTINCT CASE WHEN sub.status = 'rejected' THEN p.id END) as rejected_teams
		FROM states s
		LEFT JOIN projects p ON p.workflow_id = s.workflow_id AND p.department_id = ?
		LEFT JOIN admin_submissions sub ON sub.project_id = p.id AND sub.step_id = s.id
		WHERE s.workflow_id = ?
		GROUP BY s.id, s.name, s.type
		ORDER BY s.id
	`

	r.db.WithContext(ctx).Raw(query, departmentID, workflowID).Scan(&stats)

	return stats, nil
}

func (r *repository) GetPendingReviewsCount(ctx context.Context, departmentID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&Submission{}).
		Where("status = ?", SubmissionStatusPending).
		Count(&count).Error
	return count, err
}
