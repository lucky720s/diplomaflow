package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

	// Topic Registrations
	CreateTopicRegistration(ctx context.Context, reg *TopicRegistration) error
	GetTopicRegistration(ctx context.Context, id string) (*TopicRegistration, error)
	GetTopicRegistrationByTeam(ctx context.Context, teamID int64) (*TopicRegistration, error)
	ListTopicRegistrations(ctx context.Context, filter TopicRegistrationFilter) ([]*TopicRegistration, int64, error)
	UpdateTopicRegistration(ctx context.Context, reg *TopicRegistration) error
	CreateTopicRegistrationReview(ctx context.Context, review *TopicRegistrationReview) error
	GetTopicRegistrationReviews(ctx context.Context, registrationID string) ([]*TopicRegistrationReview, error)
	CountPendingTopicRegistrations(ctx context.Context, departmentID int64) (int64, error)

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

	// Dashboard Stats
	GetDashboardStats(ctx context.Context, departmentID int64) (*DashboardStatsData, error)
	GetStepProgressStats(ctx context.Context, departmentID int64, workflowID int64) ([]*StepProgressData, error)
	GetPendingReviewsCount(ctx context.Context, departmentID int64) (int64, error)

	// Students
	GetStudentByID(ctx context.Context, studentID int64) (*StudentFullInfo, error)
	GetStudentGrades(ctx context.Context, studentID int64) ([]*Grade, error)
	GetStudentSubmissions(ctx context.Context, studentID int64) ([]*Submission, error)

	// Teams (Admin)
	GetTeamFullDetails(ctx context.Context, teamID int64) (*TeamFullDetails, error)
	UpdateTeamByAdmin(ctx context.Context, teamID int64, updates *TeamAdminUpdateData) error
	DeleteTeamByAdmin(ctx context.Context, teamID int64, reason string, deletedBy int64) error
	AddTeamMember(ctx context.Context, teamID, userID int64, role string) error
	RemoveTeamMember(ctx context.Context, teamID, userID int64) error

	// Grading History (expanded)
	GetGradingHistoryFull(ctx context.Context, projectID, stepID int64) ([]*GradeHistoryFull, error)

	// Supervisor Request
	CreateSupervisorRequest(ctx context.Context, req *SupervisorRequest) error
	GetSupervisorRequest(ctx context.Context, id string) (*SupervisorRequest, error)
	GetSupervisorRequestWithDetails(ctx context.Context, id string) (*SupervisorRequestWithDetails, error)
	ListSupervisorRequests(ctx context.Context, filter SupervisorRequestFilter) ([]*SupervisorRequestWithDetails, int64, error)
	ListSupervisorRequestsByTeam(ctx context.Context, teamID int64) ([]*SupervisorRequest, error)
	ListSupervisorRequestsBySupervisor(ctx context.Context, supervisorID int64, status string) ([]*SupervisorRequestWithDetails, int64, error)
	UpdateSupervisorRequest(ctx context.Context, req *SupervisorRequest) error
	CreateSupervisorRequestHistory(ctx context.Context, history *SupervisorRequestHistory) error
	GetSupervisorRequestHistory(ctx context.Context, requestID string) ([]*SupervisorRequestHistory, error)
	CountPendingSupervisorRequests(ctx context.Context, supervisorID int64) (int64, error)
	HasPendingSupervisorRequest(ctx context.Context, teamID int64) (bool, error)
	HasApprovedSupervisor(ctx context.Context, teamID int64) (bool, error)

	// NEW: to create project at approve time (team-first)
	GetTeamContext(ctx context.Context, teamID int64) (*TeamContext, error)
	GetSupervisorTeamsReport(ctx context.Context, supervisorID int64) ([]*TeamFullDetails, int32 /*totalStudents*/, error)
	ListAvailableTeams(ctx context.Context, departmentID int64, limit, offset int) ([]*AvailableTeamData, int64, error)
	BatchCountTeamsBySupervisors(ctx context.Context, supervisorIDs []int64) (map[int64]int64, error)

	// Pre-Defense
	CreatePreDefenseSubmission(ctx context.Context, sub *PreDefenseSubmission) error
	GetPreDefenseSubmission(ctx context.Context, id string) (*PreDefenseSubmission, error)
	UpdatePreDefenseSubmission(ctx context.Context, sub *PreDefenseSubmission) error
	ListPreDefenseSubmissions(ctx context.Context, filter PreDefenseFilter) ([]*PreDefenseSubmission, int64, error)
	ListScheduledPreDefenses(ctx context.Context, filter ScheduleFilter) ([]*PreDefenseSubmission, error)
	AddCommissionMember(ctx context.Context, member *PreDefenseCommissionMember) error
	RemoveCommissionMember(ctx context.Context, submissionID string, userID int64) error
	GetCommissionMembers(ctx context.Context, submissionID string) ([]PreDefenseCommissionMember, error)
	UpdateCommissionMemberGrade(ctx context.Context, submissionID string, userID int64, grade int32, comment string) error
	AddPreDefenseDocument(ctx context.Context, doc *PreDefenseDocument) error
	GetPreDefenseDocuments(ctx context.Context, submissionID string) ([]PreDefenseDocument, error)
	AddPreDefenseHistory(ctx context.Context, entry *PreDefenseHistory) error
	GetPreDefenseHistory(ctx context.Context, submissionID string) ([]*PreDefenseHistory, error)
	SetProjectTopicRegisteredAt(ctx context.Context, projectID int64, t time.Time) error

	// ==================== Norm Control ====================
	EnsureNormCheckForSubmission(ctx context.Context, submissionID string) (*NormControlCheck, error)
	ListNormChecks(ctx context.Context, filter NormCheckFilter) ([]*NormControlCheck, int64, error)
	GetNormCheck(ctx context.Context, submissionID string) (*NormControlCheck, error)
	UpdateNormCheck(ctx context.Context, check *NormControlCheck) error

	ListNormIssues(ctx context.Context, submissionID string) ([]*NormControlIssue, error)
	GetNormIssue(ctx context.Context, id string) (*NormControlIssue, error)
	CreateNormIssue(ctx context.Context, issue *NormControlIssue) error
	UpdateNormIssue(ctx context.Context, issue *NormControlIssue) error
	DeleteNormIssue(ctx context.Context, id string) error

	CountUnresolvedCriticalIssues(ctx context.Context, submissionID string) (int64, error)

	AddNormHistory(ctx context.Context, h *NormControlHistory) error
	ListNormHistory(ctx context.Context, projectID int64) ([]*NormControlHistory, error)

	ListNormChecklists(ctx context.Context) ([]*NormControlChecklist, error)
	CreateNormChecklist(ctx context.Context, c *NormControlChecklist) error

	StatsNormIssuesByCategory(ctx context.Context, departmentID int64) (map[string]int64, error)

	// Supervisor Settings
	GetSupervisorSettings(ctx context.Context, userID, departmentID int64) (*SupervisorSettings, error)
	UpsertSupervisorSettings(ctx context.Context, s *SupervisorSettings) error
	CountSupervisorTeams(ctx context.Context, supervisorID int64) (int32, error)
	// ==================== Admin-tech Projects ====================
	ListProjectsAdmin(ctx context.Context, filter ProjectsAdminFilter) ([]*ProjectAdminRow, int64, error)
}

type TopicRegistrationFilter struct {
	DepartmentID int64
	SupervisorID int64
	TeamID       int64
	Status       string
	Limit        int
	Offset       int
}

type SubmissionFilter struct {
	DepartmentID int64
	StepID       int64
	TeamID       int64
	ReviewerID   int64
	Status       string
	Limit        int
	Offset       int
}

type DashboardStatsData struct {
	TotalStudents             int32
	TotalTeams                int32
	TotalProjects             int32
	CompletedProjects         int32
	PendingReviews            int32
	ActiveSupervisors         int32
	PendingTopicRegistrations int32
	PendingSupervisorRequests int32
	PendingPreDefenses        int32
	ScheduledPreDefenses      int32
	PreDefensesThisWeek       int32
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

type SupervisorRequestFilter struct {
	DepartmentID int64
	SupervisorID int64
	TeamID       int64
	Status       string
	Limit        int
	Offset       int
}

// TeamContext - минимальные данные, чтобы создать проект “по команде”.
type TeamContext struct {
	TeamID       int64
	TeamName     string
	UniversityID int64
	DepartmentID int64
	LeaderUserID int64
}
type ProjectsAdminFilter struct {
	DepartmentID int64
	TeamID       int64
	StudentID    int64
	Status       string
	Q            string
	Limit        int
	Offset       int
}

type ProjectAdminRow struct {
	ProjectID         int64      `gorm:"column:project_id"`
	Title             string     `gorm:"column:title"`
	Description       string     `gorm:"column:description"`
	StudentID         int64      `gorm:"column:student_id"`
	TeamID            int64      `gorm:"column:team_id"`
	UniversityID      int64      `gorm:"column:university_id"`
	DepartmentID      int64      `gorm:"column:department_id"`
	WorkflowName      string     `gorm:"column:workflow_name"`
	CurrentStateName  string     `gorm:"column:current_state_name"`
	Status            string     `gorm:"column:status"`
	DeadlineAt        *time.Time `gorm:"column:deadline_at"`
	TopicRegisteredAt *time.Time `gorm:"column:topic_registered_at"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
}

type repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &repository{db: db} }

// ==================== Grades ====================

func (r *repository) CreateGrade(ctx context.Context, grade *Grade) error {
	grade.CreatedAt = time.Now()
	grade.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(grade).Error
}

func (r *repository) GetGrade(ctx context.Context, projectID, stepID int64) (*Grade, error) {
	var grade Grade
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND state_id = ?", projectID, stepID).
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
		Order("state_id ASC").
		Find(&grades).Error
	return grades, err
}

func (r *repository) UpdateGrade(ctx context.Context, grade *Grade) error {
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
		query = query.Where("state_id = ?", stepID)
	}
	err := query.Order("created_at DESC").Find(&history).Error
	return history, err
}

// ==================== Topic Registrations ====================

func (r *repository) CreateTopicRegistration(ctx context.Context, reg *TopicRegistration) error {
	reg.CreatedAt = time.Now()
	reg.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(reg).Error
}

func (r *repository) GetTopicRegistration(ctx context.Context, id string) (*TopicRegistration, error) {
	var reg TopicRegistration
	err := r.db.WithContext(ctx).First(&reg, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &reg, nil
}

func (r *repository) GetTopicRegistrationByTeam(ctx context.Context, teamID int64) (*TopicRegistration, error) {
	var reg TopicRegistration
	err := r.db.WithContext(ctx).
		Where("team_id = ? AND status != ?", teamID, StatusRejected).
		Order("created_at DESC").
		First(&reg).Error
	if err != nil {
		return nil, err
	}
	return &reg, nil
}

func (r *repository) ListTopicRegistrations(ctx context.Context, filter TopicRegistrationFilter) ([]*TopicRegistration, int64, error) {
	query := r.db.WithContext(ctx).Model(&TopicRegistration{})

	if filter.DepartmentID > 0 {
		query = query.Joins("JOIN projects p ON p.id = admin_topic_registrations.project_id").
			Where("p.department_id = ?", filter.DepartmentID)
	}
	if filter.Status != "" {
		query = query.Where("admin_topic_registrations.status = ?", filter.Status)
	}
	if filter.SupervisorID > 0 {
		query = query.Where("admin_topic_registrations.supervisor_id = ?", filter.SupervisorID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var registrations []*TopicRegistration
	err := query.
		Order("admin_topic_registrations.created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&registrations).Error

	return registrations, total, err
}

func (r *repository) UpdateTopicRegistration(ctx context.Context, reg *TopicRegistration) error {
	reg.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(reg).Error
}

func (r *repository) CreateTopicRegistrationReview(ctx context.Context, review *TopicRegistrationReview) error {
	review.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(review).Error
}

func (r *repository) GetTopicRegistrationReviews(ctx context.Context, registrationID string) ([]*TopicRegistrationReview, error) {
	var reviews []*TopicRegistrationReview
	err := r.db.WithContext(ctx).
		Where("registration_id = ?", registrationID).
		Order("created_at DESC").
		Find(&reviews).Error
	return reviews, err
}

func (r *repository) CountPendingTopicRegistrations(ctx context.Context, departmentID int64) (int64, error) {
	var count int64
	q := r.db.WithContext(ctx).Table("admin_topic_registrations tr").
		Select("COUNT(*)").
		Joins("JOIN projects p ON p.id = tr.project_id").
		Where("tr.status = ? AND p.department_id = ?", StatusPending, departmentID)
	err := q.Scan(&count).Error
	return count, err
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
		query = query.Where("admin_submissions.state_id = ?", filter.StepID)
	}
	if filter.TeamID > 0 {
		query = query.Where("admin_submissions.team_id = ?", filter.TeamID)
	}
	if filter.Status != "" && filter.Status != "all" {
		query = query.Where("admin_submissions.status = ?", filter.Status) // ✅ FIX
	}
	if filter.DepartmentID > 0 {
		query = query.Joins("JOIN projects p ON p.id = admin_submissions.project_id").
			Where("p.department_id = ?", filter.DepartmentID)
	}
	if filter.ReviewerID > 0 {
		query = query.Where("admin_submissions.reviewer_id = ?", filter.ReviewerID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Order("admin_submissions.created_at DESC").
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

// ==================== Supervisor Request ====================

func (r *repository) CreateSupervisorRequest(ctx context.Context, req *SupervisorRequest) error {
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *repository) GetSupervisorRequest(ctx context.Context, id string) (*SupervisorRequest, error) {
	var req SupervisorRequest
	err := r.db.WithContext(ctx).First(&req, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *repository) GetSupervisorRequestWithDetails(ctx context.Context, id string) (*SupervisorRequestWithDetails, error) {
	var result SupervisorRequestWithDetails
	query := `
		SELECT 
			sr.*,
			COALESCE(t.name, '') as team_name,
			COALESCE(CONCAT(sup.first_name, ' ', sup.last_name), '') as supervisor_name,
			COALESCE(sup.email, '') as supervisor_email,
			COALESCE(CONCAT(req.first_name, ' ', req.last_name), '') as requester_name
		FROM admin_supervisor_requests sr
		LEFT JOIN teams t ON t.id = sr.team_id
		LEFT JOIN users sup ON sup.id = sr.supervisor_id
		LEFT JOIN users req ON req.id = sr.requested_by
		WHERE sr.id = ?
	`
	err := r.db.WithContext(ctx).Raw(query, id).Scan(&result).Error
	if err != nil {
		return nil, err
	}
	if result.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return &result, nil
}

func (r *repository) ListSupervisorRequests(ctx context.Context, filter SupervisorRequestFilter) ([]*SupervisorRequestWithDetails, int64, error) {
	var results []*SupervisorRequestWithDetails
	var total int64

	baseQuery := `
		FROM admin_supervisor_requests sr
		LEFT JOIN teams t ON t.id = sr.team_id
		LEFT JOIN users sup ON sup.id = sr.supervisor_id
		LEFT JOIN users req ON req.id = sr.requested_by
		WHERE 1=1
	`
	args := []interface{}{}

	if filter.SupervisorID > 0 {
		baseQuery += " AND sr.supervisor_id = ?"
		args = append(args, filter.SupervisorID)
	}
	if filter.TeamID > 0 {
		baseQuery += " AND sr.team_id = ?"
		args = append(args, filter.TeamID)
	}
	if filter.Status != "" && filter.Status != "all" {
		baseQuery += " AND sr.status = ?"
		args = append(args, filter.Status)
	}
	// NOTE: department filter is not directly present on supervisor_requests
	// because team_id can be NULL in schema. If needed later, join through projects/teams.

	countQuery := "SELECT COUNT(*) " + baseQuery
	r.db.WithContext(ctx).Raw(countQuery, args...).Scan(&total)

	selectQuery := `
		SELECT 
			sr.*,
			COALESCE(t.name, '') as team_name,
			COALESCE(CONCAT(sup.first_name, ' ', sup.last_name), '') as supervisor_name,
			COALESCE(sup.email, '') as supervisor_email,
			COALESCE(CONCAT(req.first_name, ' ', req.last_name), '') as requester_name
	` + baseQuery + " ORDER BY sr.created_at DESC"

	if filter.Limit > 0 {
		selectQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", filter.Limit, filter.Offset)
	}

	err := r.db.WithContext(ctx).Raw(selectQuery, args...).Scan(&results).Error
	return results, total, err
}

func (r *repository) ListSupervisorRequestsByTeam(ctx context.Context, teamID int64) ([]*SupervisorRequest, error) {
	var requests []*SupervisorRequest
	err := r.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

func (r *repository) ListSupervisorRequestsBySupervisor(ctx context.Context, supervisorID int64, status string) ([]*SupervisorRequestWithDetails, int64, error) {
	filter := SupervisorRequestFilter{
		SupervisorID: supervisorID,
		Status:       status,
		Limit:        100,
		Offset:       0,
	}
	return r.ListSupervisorRequests(ctx, filter)
}

func (r *repository) UpdateSupervisorRequest(ctx context.Context, req *SupervisorRequest) error {
	req.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(req).Error
}

func (r *repository) CreateSupervisorRequestHistory(ctx context.Context, history *SupervisorRequestHistory) error {
	history.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(history).Error
}

func (r *repository) GetSupervisorRequestHistory(ctx context.Context, requestID string) ([]*SupervisorRequestHistory, error) {
	var history []*SupervisorRequestHistory
	err := r.db.WithContext(ctx).
		Where("request_id = ?", requestID).
		Order("created_at DESC").
		Find(&history).Error
	return history, err
}

func (r *repository) CountPendingSupervisorRequests(ctx context.Context, supervisorID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&SupervisorRequest{}).
		Where("supervisor_id = ? AND status = ?", supervisorID, SupervisorRequestStatusPending).
		Count(&count).Error
	return count, err
}

func (r *repository) HasPendingSupervisorRequest(ctx context.Context, teamID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&SupervisorRequest{}).
		Where("team_id = ? AND status = ?", teamID, SupervisorRequestStatusPending).
		Count(&count).Error
	return count > 0, err
}

func (r *repository) HasApprovedSupervisor(ctx context.Context, teamID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&SupervisorRequest{}).
		Where("team_id = ? AND status = ?", teamID, SupervisorRequestStatusApproved).
		Count(&count).Error
	return count > 0, err
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
	// department filter isn't stored in admin_activities by default; keep simple
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

	var totalStudents int64
	r.db.WithContext(ctx).
		Table("users").
		Where("role = ? AND department_id = ? AND deleted_at IS NULL", "student", departmentID).
		Count(&totalStudents)
	stats.TotalStudents = int32(totalStudents)

	// Teams: use teams.department_id (migration 000016) - not via projects
	var totalTeams int64
	r.db.WithContext(ctx).
		Table("teams").
		Where("department_id = ? AND deleted_at IS NULL", departmentID).
		Count(&totalTeams)
	stats.TotalTeams = int32(totalTeams)

	var totalProjects int64
	r.db.WithContext(ctx).
		Table("projects").
		Where("department_id = ?", departmentID).
		Count(&totalProjects)
	stats.TotalProjects = int32(totalProjects)

	var completedProjects int64
	r.db.WithContext(ctx).
		Table("projects").
		Where("department_id = ? AND status = ?", departmentID, "completed").
		Count(&completedProjects)
	stats.CompletedProjects = int32(completedProjects)

	// Pending reviews: only within department via projects join
	var pendingReviews int64
	r.db.WithContext(ctx).
		Table("admin_submissions s").
		Joins("JOIN projects p ON p.id = s.project_id").
		Where("s.status = ? AND p.department_id = ?", StatusPending, departmentID).
		Count(&pendingReviews)
	stats.PendingReviews = int32(pendingReviews)

	// Active supervisors: distinct supervisor_id within department via teams join
	var activeSupervisors int64
	r.db.WithContext(ctx).
		Table("admin_supervisor_assignments a").
		Joins("JOIN teams t ON t.id = a.team_id").
		Where("t.department_id = ? AND t.deleted_at IS NULL", departmentID).
		Distinct("a.supervisor_id").
		Count(&activeSupervisors)
	stats.ActiveSupervisors = int32(activeSupervisors)

	var pendingTopicRegs int64
	r.db.WithContext(ctx).
		Table("admin_topic_registrations tr").
		Joins("JOIN projects p ON p.id = tr.project_id").
		Where("tr.status = ? AND p.department_id = ?", StatusPending, departmentID).
		Count(&pendingTopicRegs)
	stats.PendingTopicRegistrations = int32(pendingTopicRegs)

	// Optional: pending supervisor requests count
	var pendingSupReq int64
	r.db.WithContext(ctx).
		Table("admin_supervisor_requests sr").
		Joins("LEFT JOIN teams t ON t.id = sr.team_id").
		Where("sr.status = ? AND t.department_id = ?", SupervisorRequestStatusPending, departmentID).
		Count(&pendingSupReq)
	stats.PendingSupervisorRequests = int32(pendingSupReq)
	// ==================== Pre-Defense Stats ====================

	// Pending pre-defenses
	var pendingPreDefenses int64
	r.db.WithContext(ctx).
		Table("admin_pre_defense_submissions pds").
		Joins("JOIN projects p ON p.id = pds.project_id").
		Where("pds.status = ? AND p.department_id = ?", "pending", departmentID).
		Count(&pendingPreDefenses)
	stats.PendingPreDefenses = int32(pendingPreDefenses)

	// Scheduled pre-defenses
	var scheduledPreDefenses int64
	r.db.WithContext(ctx).
		Table("admin_pre_defense_submissions pds").
		Joins("JOIN projects p ON p.id = pds.project_id").
		Where("pds.status = ? AND p.department_id = ?", "scheduled", departmentID).
		Count(&scheduledPreDefenses)
	stats.ScheduledPreDefenses = int32(scheduledPreDefenses)

	now := time.Now()
	weekStart := now.AddDate(0, 0, -int(now.Weekday()))
	weekEnd := weekStart.AddDate(0, 0, 7)
	var preDefensesThisWeek int64
	r.db.WithContext(ctx).
		Table("admin_pre_defense_submissions pds").
		Joins("JOIN projects p ON p.id = pds.project_id").
		Where("pds.scheduled_date >= ? AND pds.scheduled_date < ? AND p.department_id = ?",
			weekStart, weekEnd, departmentID).
		Count(&preDefensesThisWeek)
	stats.PreDefensesThisWeek = int32(preDefensesThisWeek)

	return stats, nil
}

func (r *repository) GetStepProgressStats(ctx context.Context, departmentID int64, workflowID int64) ([]*StepProgressData, error) {
	var stats []*StepProgressData
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
		LEFT JOIN admin_submissions sub ON sub.project_id = p.id AND sub.state_id = s.id
		WHERE s.workflow_id = ?
		GROUP BY s.id, s.name, s.type
		ORDER BY s.order_index
	`
	r.db.WithContext(ctx).Raw(query, departmentID, workflowID).Scan(&stats)
	return stats, nil
}

func (r *repository) GetPendingReviewsCount(ctx context.Context, departmentID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("admin_submissions s").
		Joins("JOIN projects p ON p.id = s.project_id").
		Where("s.status = ? AND p.department_id = ?", StatusPending, departmentID).
		Count(&count).Error
	return count, err
}

// ==================== Students ====================

func (r *repository) GetStudentByID(ctx context.Context, studentID int64) (*StudentFullInfo, error) {
	var result StudentFullInfo
	query := `
		SELECT 
			u.id,
			u.email,
			u.first_name,
			u.last_name,
			u.role,
			u.university_id,
			u.department_id,
			COALESCE(tm.team_id, 0) as team_id,
			COALESCE(t.name, '') as team_name,
			COALESCE(p.id, 0) as project_id,
			COALESCE(p.title, '') as project_title,
			COALESCE(p.current_state_name, '') as current_step,
			u.created_at
		FROM users u
		LEFT JOIN team_members tm ON tm.user_id = u.id
		LEFT JOIN teams t ON t.id = tm.team_id AND t.deleted_at IS NULL
		LEFT JOIN projects p ON p.team_id = t.id
		WHERE u.id = ? AND u.deleted_at IS NULL
	`
	err := r.db.WithContext(ctx).Raw(query, studentID).Scan(&result).Error
	if err != nil {
		return nil, err
	}
	if result.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &result, nil
}

func (r *repository) GetStudentGrades(ctx context.Context, studentID int64) ([]*Grade, error) {
	var grades []*Grade
	query := `
		SELECT g.*
		FROM admin_grades g
		JOIN projects p ON p.id = g.project_id
		JOIN teams t ON t.id = p.team_id
		JOIN team_members tm ON tm.team_id = t.id
		WHERE tm.user_id = ? AND g.deleted_at IS NULL
		ORDER BY g.created_at DESC
	`
	err := r.db.WithContext(ctx).Raw(query, studentID).Scan(&grades).Error
	return grades, err
}

func (r *repository) GetStudentSubmissions(ctx context.Context, studentID int64) ([]*Submission, error) {
	var submissions []*Submission
	query := `
		SELECT s.*
		FROM admin_submissions s
		JOIN projects p ON p.id = s.project_id
		JOIN teams t ON t.id = p.team_id
		JOIN team_members tm ON tm.team_id = t.id
		WHERE tm.user_id = ? AND s.deleted_at IS NULL
		ORDER BY s.created_at DESC
		LIMIT 50
	`
	err := r.db.WithContext(ctx).Raw(query, studentID).Scan(&submissions).Error
	return submissions, err
}

// ==================== Teams ====================

func (r *repository) GetTeamFullDetails(ctx context.Context, teamID int64) (*TeamFullDetails, error) {
	var team TeamFullDetails

	teamQuery := `
		SELECT 
			t.id,
			t.name,
			COALESCE(p.id, 0) as project_id,
			COALESCE(p.title, '') as project_title,
			COALESCE(p.current_state_name, '') as current_step,
			COALESCE(p.status, 'active') as status,
			COALESCE(sa.supervisor_id, 0) as supervisor_id,
			t.created_at,
			t.updated_at
		FROM teams t
		LEFT JOIN projects p ON p.team_id = t.id
		LEFT JOIN admin_supervisor_assignments sa ON sa.team_id = t.id
		WHERE t.id = ? AND t.deleted_at IS NULL
	`
	if err := r.db.WithContext(ctx).Raw(teamQuery, teamID).Scan(&team).Error; err != nil {
		return nil, err
	}
	if team.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	membersQuery := `
		SELECT 
			tm.user_id,
			CONCAT(u.first_name, ' ', u.last_name) as full_name,
			u.email,
			tm.role,
			tm.created_at as joined_at
		FROM team_members tm
		JOIN users u ON u.id = tm.user_id
		WHERE tm.team_id = ?
		ORDER BY tm.role DESC, tm.created_at ASC
	`
	var members []*TeamMemberDetails
	if err := r.db.WithContext(ctx).Raw(membersQuery, teamID).Scan(&members).Error; err != nil {
		return nil, err
	}
	team.Members = members
	return &team, nil
}

func (r *repository) UpdateTeamByAdmin(ctx context.Context, teamID int64, updates *TeamAdminUpdateData) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()

		// name
		if updates != nil && updates.Name != nil && strings.TrimSpace(*updates.Name) != "" {
			if err := tx.Table("teams").
				Where("id = ?", teamID).
				Updates(map[string]any{
					"name":       strings.TrimSpace(*updates.Name),
					"updated_at": now,
				}).Error; err != nil {
				return err
			}
		}

		// supervisor assignment
		if updates != nil && updates.SupervisorID != nil && *updates.SupervisorID > 0 {
			assignedBy := updates.UpdatedBy
			if assignedBy <= 0 {
				return fmt.Errorf("updated_by is required for supervisor assignment audit")
			}

			// пробуем update; если rows=0 — create
			res := tx.Table("admin_supervisor_assignments").
				Where("team_id = ?", teamID).
				Updates(map[string]any{
					"supervisor_id": *updates.SupervisorID,
					"assigned_by":   assignedBy,
					"updated_at":    now,
				})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				if err := tx.Table("admin_supervisor_assignments").Create(map[string]any{
					"team_id":       teamID,
					"supervisor_id": *updates.SupervisorID,
					"assigned_by":   assignedBy,
					"created_at":    now,
					"updated_at":    now,
				}).Error; err != nil {
					return err
				}
			}
		}

		// members sync (только если передали непустой список — сохраняем текущее поведение контракта)
		if updates != nil && len(updates.MemberIDs) > 0 {
			// узнаём лидера (чтобы не удалить)
			var leaderID int64
			if err := tx.Table("team_members").
				Select("user_id").
				Where("team_id = ? AND role = ?", teamID, "leader").
				Limit(1).
				Scan(&leaderID).Error; err != nil {
				return err
			}

			var currentMemberIDs []int64
			if err := tx.Table("team_members").
				Where("team_id = ?", teamID).
				Pluck("user_id", &currentMemberIDs).Error; err != nil {
				return err
			}

			currentMap := make(map[int64]bool, len(currentMemberIDs))
			for _, id := range currentMemberIDs {
				currentMap[id] = true
			}

			newMap := make(map[int64]bool, len(updates.MemberIDs)+1)
			for _, id := range updates.MemberIDs {
				if id > 0 {
					newMap[id] = true
				}
			}
			// лидер всегда остаётся участником, даже если фронт “забыл” его прислать
			if leaderID > 0 {
				newMap[leaderID] = true
			}

			// delete removed (кроме лидера)
			for _, id := range currentMemberIDs {
				if !newMap[id] {
					if leaderID > 0 && id == leaderID {
						continue
					}
					if err := tx.Exec(
						`DELETE FROM team_members WHERE team_id = ? AND user_id = ?`,
						teamID, id,
					).Error; err != nil {
						return err
					}
				}
			}

			// add new (всех новых — как member; лидера не трогаем)
			for _, id := range updates.MemberIDs {
				if id <= 0 {
					continue
				}
				if leaderID > 0 && id == leaderID {
					continue
				}
				if !currentMap[id] {
					if err := tx.Table("team_members").Create(map[string]any{
						"team_id":    teamID,
						"user_id":    id,
						"role":       "member",
						"created_at": now,
					}).Error; err != nil {
						return err
					}
				}
			}

			// touch teams.updated_at
			if err := tx.Table("teams").Where("id = ?", teamID).Update("updated_at", now).Error; err != nil {
				return err
			}
		}

		// если апдейтили только supervisor/name без members — тоже обновим updated_at (кроме случая когда уже обновили выше)
		return tx.Table("teams").Where("id = ?", teamID).Update("updated_at", now).Error
	})
}
func (r *repository) DeleteTeamByAdmin(ctx context.Context, teamID int64, reason string, deletedBy int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()

		if err := tx.Create(&AdminActivity{
			ActivityType: ActivityTypeTeamDelete,
			Description:  fmt.Sprintf("Team %d deleted. Reason: %s", teamID, reason),
			ActorID:      deletedBy,
			TargetID:     teamID,
			TargetType:   "team",
			CreatedAt:    now,
		}).Error; err != nil {
			return err
		}

		// FIX: никаких Delete(nil)
		if err := tx.Exec(`DELETE FROM admin_supervisor_assignments WHERE team_id = ?`, teamID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DELETE FROM team_invites WHERE team_id = ?`, teamID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DELETE FROM team_members WHERE team_id = ?`, teamID).Error; err != nil {
			return err
		}

		return tx.Table("teams").Where("id = ?", teamID).Update("deleted_at", now).Error
	})
}

func (r *repository) AddTeamMember(ctx context.Context, teamID, userID int64, role string) error {
	if role == "" {
		role = "member"
	}
	return r.db.WithContext(ctx).Table("team_members").Create(map[string]interface{}{
		"team_id":    teamID,
		"user_id":    userID,
		"role":       role,
		"created_at": time.Now(),
	}).Error
}

func (r *repository) RemoveTeamMember(ctx context.Context, teamID, userID int64) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM team_members WHERE team_id = ? AND user_id = ?`,
		teamID, userID,
	).Error
}

// ==================== Grading History Full ====================

func (r *repository) GetGradingHistoryFull(ctx context.Context, projectID, stepID int64) ([]*GradeHistoryFull, error) {
	var history []*GradeHistoryFull
	query := `
		SELECT 
			gh.id,
			gh.project_id,
			gh.state_id as step_id,
			COALESCE(s.name, '') as step_name,
			gh.old_grade,
			gh.new_grade,
			gh.changed_by,
			CONCAT(u.first_name, ' ', u.last_name) as changer_name,
			COALESCE(gh.reason, '') as reason,
			gh.created_at as changed_at
		FROM admin_grade_history gh
		LEFT JOIN states s ON s.id = gh.state_id
		LEFT JOIN users u ON u.id = gh.changed_by
		WHERE gh.project_id = ?
	`
	args := []interface{}{projectID}
	if stepID > 0 {
		query += " AND gh.state_id = ?"
		args = append(args, stepID)
	}
	query += " ORDER BY gh.created_at DESC"
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&history).Error
	return history, err
}

// ==================== NEW: TeamContext ====================

func (r *repository) GetTeamContext(ctx context.Context, teamID int64) (*TeamContext, error) {
	if teamID <= 0 {
		return nil, fmt.Errorf("team_id is required")
	}

	type row struct {
		ID           int64
		Name         string
		UniversityID int64
		DepartmentID int64
		DeletedAt    *time.Time
	}

	var t row
	if err := r.db.WithContext(ctx).
		Table("teams").
		Select("id, name, university_id, department_id, deleted_at").
		Where("id = ?", teamID).
		Scan(&t).Error; err != nil {
		return nil, err
	}
	if t.ID == 0 || t.DeletedAt != nil {
		return nil, gorm.ErrRecordNotFound
	}

	leaderID := int64(0)
	_ = r.db.WithContext(ctx).
		Table("team_members").
		Select("user_id").
		Where("team_id = ? AND role = ?", teamID, "leader").
		Limit(1).
		Scan(&leaderID).Error

	if leaderID == 0 {
		_ = r.db.WithContext(ctx).
			Table("team_members").
			Select("user_id").
			Where("team_id = ?", teamID).
			Limit(1).
			Scan(&leaderID).Error
	}
	if leaderID == 0 {
		return nil, fmt.Errorf("team has no members (cannot resolve leader)")
	}

	return &TeamContext{
		TeamID:       t.ID,
		TeamName:     t.Name,
		UniversityID: t.UniversityID,
		DepartmentID: t.DepartmentID,
		LeaderUserID: leaderID,
	}, nil
}
func (r *repository) GetSupervisorTeamsReport(ctx context.Context, supervisorID int64) ([]*TeamFullDetails, int32, error) {
	if supervisorID <= 0 {
		return []*TeamFullDetails{}, 0, fmt.Errorf("supervisor_id is required")
	}

	// 1) Список команд преподавателя + проектная инфа
	var teams []*TeamFullDetails
	teamsQuery := `
		SELECT
			t.id,
			t.name,
			COALESCE(p.id, 0)    as project_id,
			COALESCE(p.title, '') as project_title,
			COALESCE(p.current_state_name, '') as current_step,
			COALESCE(p.status, 'active') as status,
			COALESCE(a.supervisor_id, 0) as supervisor_id,
			t.created_at,
			t.updated_at
		FROM admin_supervisor_assignments a
		JOIN teams t ON t.id = a.team_id
		LEFT JOIN projects p ON p.team_id = t.id
		WHERE a.supervisor_id = ?
		  AND t.deleted_at IS NULL
		ORDER BY t.created_at DESC
	`
	if err := r.db.WithContext(ctx).Raw(teamsQuery, supervisorID).Scan(&teams).Error; err != nil {
		return nil, 0, err
	}

	if len(teams) == 0 {
		return []*TeamFullDetails{}, 0, nil
	}

	teamIDs := make([]int64, 0, len(teams))
	for _, t := range teams {
		if t != nil && t.ID > 0 {
			teamIDs = append(teamIDs, t.ID)
		}
	}

	// 2) Точное общее число студентов (distinct)
	var total int64
	totalQuery := `
		SELECT COUNT(DISTINCT tm.user_id)
		FROM team_members tm
		WHERE tm.team_id IN (?)
	`
	if err := r.db.WithContext(ctx).Raw(totalQuery, teamIDs).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	// 3) Участники всех команд одним запросом
	type memberRow struct {
		TeamID   int64     `gorm:"column:team_id"`
		UserID   int64     `gorm:"column:user_id"`
		FullName string    `gorm:"column:full_name"`
		Email    string    `gorm:"column:email"`
		Role     string    `gorm:"column:role"`
		JoinedAt time.Time `gorm:"column:joined_at"`
	}

	var rows []*memberRow
	membersQuery := `
		SELECT
			tm.team_id,
			tm.user_id,
			CONCAT(u.first_name, ' ', u.last_name) as full_name,
			u.email,
			tm.role,
			tm.created_at as joined_at
		FROM team_members tm
		JOIN users u ON u.id = tm.user_id
		WHERE tm.team_id IN (?)
		ORDER BY tm.team_id ASC, tm.role DESC, tm.created_at ASC
	`
	if err := r.db.WithContext(ctx).Raw(membersQuery, teamIDs).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	mmap := make(map[int64][]*TeamMemberDetails, len(teamIDs))
	for _, m := range rows {
		if m == nil {
			continue
		}
		mmap[m.TeamID] = append(mmap[m.TeamID], &TeamMemberDetails{
			UserID:   m.UserID,
			FullName: m.FullName,
			Email:    m.Email,
			Role:     m.Role,
			JoinedAt: m.JoinedAt,
		})
	}

	for _, t := range teams {
		if t == nil {
			continue
		}
		t.Members = mmap[t.ID]
	}

	return teams, int32(total), nil
}
func (r *repository) ListAvailableTeams(ctx context.Context, departmentID int64, limit, offset int) ([]*AvailableTeamData, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// 1) total_count: команды кафедры без назначенного supervisor
	var total int64
	countQ := `
		SELECT COUNT(*)
		FROM teams t
		LEFT JOIN admin_supervisor_assignments a ON a.team_id = t.id
		WHERE t.department_id = ?
		  AND t.deleted_at IS NULL
		  AND a.team_id IS NULL
		  AND NOT EXISTS (
		    SELECT 1 FROM admin_supervisor_requests sr
		    WHERE sr.team_id = t.id AND sr.status = 'approved'
		  )
	`
	if err := r.db.WithContext(ctx).Raw(countQ, departmentID).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*AvailableTeamData{}, 0, nil
	}

	// 2) page teams
	type teamRow struct {
		ID        int64     `gorm:"column:id"`
		Name      string    `gorm:"column:name"`
		CreatedAt time.Time `gorm:"column:created_at"`
	}
	var rows []*teamRow

	listQ := `
		SELECT t.id, t.name, t.created_at
		FROM teams t
		LEFT JOIN admin_supervisor_assignments a ON a.team_id = t.id
		WHERE t.department_id = ?
		  AND t.deleted_at IS NULL
		  AND a.team_id IS NULL
		  AND NOT EXISTS (
		    SELECT 1 FROM admin_supervisor_requests sr
		    WHERE sr.team_id = t.id AND sr.status = 'approved'
		  )
		ORDER BY t.created_at DESC
		LIMIT ? OFFSET ?
	`
	if err := r.db.WithContext(ctx).Raw(listQ, departmentID, limit, offset).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	if len(rows) == 0 {
		return []*AvailableTeamData{}, total, nil
	}

	teamIDs := make([]int64, 0, len(rows))
	for _, t := range rows {
		teamIDs = append(teamIDs, t.ID)
	}

	// 3) members for all teams одним запросом
	type memberRow struct {
		TeamID   int64     `gorm:"column:team_id"`
		UserID   int64     `gorm:"column:user_id"`
		FullName string    `gorm:"column:full_name"`
		Email    string    `gorm:"column:email"`
		Role     string    `gorm:"column:role"`
		JoinedAt time.Time `gorm:"column:joined_at"`
	}
	var mrows []*memberRow
	membersQ := `
		SELECT
			tm.team_id,
			tm.user_id,
			CONCAT(u.first_name, ' ', u.last_name) as full_name,
			u.email,
			tm.role,
			tm.created_at as joined_at
		FROM team_members tm
		JOIN users u ON u.id = tm.user_id
		WHERE tm.team_id IN (?)
		ORDER BY tm.team_id ASC, tm.role DESC, tm.created_at ASC
	`
	if err := r.db.WithContext(ctx).Raw(membersQ, teamIDs).Scan(&mrows).Error; err != nil {
		return nil, 0, err
	}

	mmap := make(map[int64][]*TeamMemberDetails, len(teamIDs))
	for _, m := range mrows {
		mmap[m.TeamID] = append(mmap[m.TeamID], &TeamMemberDetails{
			UserID:   m.UserID,
			TeamID:   m.TeamID,
			FullName: m.FullName,
			Email:    m.Email,
			Role:     m.Role,
			JoinedAt: m.JoinedAt,
		})
	}

	out := make([]*AvailableTeamData, 0, len(rows))
	for _, t := range rows {
		members := mmap[t.ID]
		out = append(out, &AvailableTeamData{
			ID:          t.ID,
			Name:        t.Name,
			CreatedAt:   t.CreatedAt,
			Members:     members,
			MemberCount: int32(len(members)),
		})
	}
	return out, total, nil
}
func (r *repository) BatchCountTeamsBySupervisors(ctx context.Context, supervisorIDs []int64) (map[int64]int64, error) {
	if len(supervisorIDs) == 0 {
		return map[int64]int64{}, nil
	}

	type row struct {
		SupervisorID int64 `gorm:"column:supervisor_id"`
		Count        int64 `gorm:"column:cnt"`
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Table("admin_supervisor_assignments").
		Select("supervisor_id, COUNT(*) as cnt").
		Where("supervisor_id IN ?", supervisorIDs).
		Group("supervisor_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int64]int64, len(rows))
	for _, r := range rows {
		result[r.SupervisorID] = r.Count
	}
	return result, nil
}
func (r *repository) SetProjectTopicRegisteredAt(ctx context.Context, projectID int64, t time.Time) error {
	return r.db.WithContext(ctx).
		Table("projects").
		Where("id = ?", projectID).
		Update("topic_registered_at", t).Error
}
func (r *repository) GetSupervisorSettings(ctx context.Context, userID, departmentID int64) (*SupervisorSettings, error) {
	var s SupervisorSettings
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND department_id = ?", userID, departmentID).
		First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil // нет индивидуальных настроек — используем дефолт
	}
	return &s, err
}

func (r *repository) UpsertSupervisorSettings(ctx context.Context, s *SupervisorSettings) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND department_id = ?", s.UserID, s.DepartmentID).
		Assign(map[string]interface{}{
			"max_teams":  s.MaxTeams,
			"updated_by": s.UpdatedBy,
			"updated_at": time.Now(),
		}).
		FirstOrCreate(s).Error
}

func (r *repository) CountSupervisorTeams(ctx context.Context, supervisorID int64) (int32, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&SupervisorAssignment{}).
		Where("supervisor_id = ?", supervisorID).
		Count(&count).Error
	return int32(count), err
}

func (r *repository) ListProjectsAdmin(ctx context.Context, f ProjectsAdminFilter) ([]*ProjectAdminRow, int64, error) {
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	q := r.db.WithContext(ctx).Table("projects p").
		Select(`
			p.id as project_id,
			p.title,
			COALESCE(p.description,'') as description,
			p.student_id,
			COALESCE(p.team_id,0) as team_id,
			p.university_id,
			COALESCE(p.department_id,0) as department_id,
			COALESCE(p.workflow_name,'') as workflow_name,
			COALESCE(p.current_state_name,'') as current_state_name,
			p.status,
			p.deadline_at,
			p.topic_registered_at,
			p.created_at,
			p.updated_at
		`)

	if f.DepartmentID > 0 {
		q = q.Where("p.department_id = ?", f.DepartmentID)
	}
	if f.TeamID > 0 {
		q = q.Where("p.team_id = ?", f.TeamID)
	}
	if f.StudentID > 0 {
		q = q.Where("p.student_id = ?", f.StudentID)
	}
	if strings.TrimSpace(f.Status) != "" {
		q = q.Where("p.status = ?", strings.TrimSpace(f.Status))
	}
	if s := strings.TrimSpace(f.Q); s != "" {
		like := "%" + s + "%"
		q = q.Where("(p.title ILIKE ? OR p.description ILIKE ?)", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []*ProjectAdminRow
	err := q.Order("p.created_at DESC").
		Limit(f.Limit).
		Offset(f.Offset).
		Find(&rows).Error

	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
