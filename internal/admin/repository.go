package admin

import (
	"context"
	"fmt"
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

	// Grading History (расширенный)
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
}

// Filter structs
type TopicRegistrationFilter struct {
	DepartmentID int64
	TeamID       int64
	Status       string
	Limit        int
	Offset       int
}

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

// SupervisorRequestFilter - фильтр для списка запросов
type SupervisorRequestFilter struct {
	DepartmentID int64
	SupervisorID int64
	TeamID       int64
	Status       string
	Limit        int
	Offset       int
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// ==================== Grades ====================

func (r *repository) CreateGrade(ctx context.Context, grade *Grade) error {
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
	var regs []*TopicRegistration
	var total int64

	query := r.db.WithContext(ctx).Model(&TopicRegistration{})

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
		Find(&regs).Error

	return regs, total, err
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
	err := r.db.WithContext(ctx).
		Model(&TopicRegistration{}).
		Where("status = ?", StatusPending).
		Count(&count).Error
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

// ==================== Supervisor Request Implementation ====================

// CreateSupervisorRequest - создание запроса к супервайзеру
func (r *repository) CreateSupervisorRequest(ctx context.Context, req *SupervisorRequest) error {
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(req).Error
}

// GetSupervisorRequest - получение запроса по ID
func (r *repository) GetSupervisorRequest(ctx context.Context, id string) (*SupervisorRequest, error) {
	var req SupervisorRequest
	err := r.db.WithContext(ctx).First(&req, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

// GetSupervisorRequestWithDetails - получение запроса с деталями
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

// ListSupervisorRequests - список запросов с фильтрацией
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

	// Count total
	countQuery := "SELECT COUNT(*) " + baseQuery
	r.db.WithContext(ctx).Raw(countQuery, args...).Scan(&total)

	// Get data
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

// ListSupervisorRequestsByTeam - список запросов команды
func (r *repository) ListSupervisorRequestsByTeam(ctx context.Context, teamID int64) ([]*SupervisorRequest, error) {
	var requests []*SupervisorRequest
	err := r.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

// ListSupervisorRequestsBySupervisor - список запросов для супервайзера
func (r *repository) ListSupervisorRequestsBySupervisor(ctx context.Context, supervisorID int64, status string) ([]*SupervisorRequestWithDetails, int64, error) {
	filter := SupervisorRequestFilter{
		SupervisorID: supervisorID,
		Status:       status,
		Limit:        100,
		Offset:       0,
	}
	return r.ListSupervisorRequests(ctx, filter)
}

// UpdateSupervisorRequest - обновление запроса
func (r *repository) UpdateSupervisorRequest(ctx context.Context, req *SupervisorRequest) error {
	req.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(req).Error
}

// CreateSupervisorRequestHistory - создание записи в истории
func (r *repository) CreateSupervisorRequestHistory(ctx context.Context, history *SupervisorRequestHistory) error {
	history.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(history).Error
}

// GetSupervisorRequestHistory - получение истории запроса
func (r *repository) GetSupervisorRequestHistory(ctx context.Context, requestID string) ([]*SupervisorRequestHistory, error) {
	var history []*SupervisorRequestHistory
	err := r.db.WithContext(ctx).
		Where("request_id = ?", requestID).
		Order("created_at DESC").
		Find(&history).Error
	return history, err
}

// CountPendingSupervisorRequests - количество ожидающих запросов для супервайзера
func (r *repository) CountPendingSupervisorRequests(ctx context.Context, supervisorID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&SupervisorRequest{}).
		Where("supervisor_id = ? AND status = ?", supervisorID, SupervisorRequestStatusPending).
		Count(&count).Error
	return count, err
}

// HasPendingSupervisorRequest - есть ли активный запрос у команды
func (r *repository) HasPendingSupervisorRequest(ctx context.Context, teamID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&SupervisorRequest{}).
		Where("team_id = ? AND status = ?", teamID, SupervisorRequestStatusPending).
		Count(&count).Error
	return count > 0, err
}

// HasApprovedSupervisor - есть ли утверждённый супервайзер у команды
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

	var totalStudents, totalTeams, totalProjects, completedProjects, pendingReviews, activeSupervisors, pendingTopicRegs int64

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

	// Count pending reviews (submissions)
	r.db.WithContext(ctx).
		Table("admin_submissions").
		Where("status = ?", StatusPending).
		Count(&pendingReviews)
	stats.PendingReviews = int32(pendingReviews)

	// Count active supervisors
	r.db.WithContext(ctx).
		Table("admin_supervisor_assignments").
		Distinct("supervisor_id").
		Count(&activeSupervisors)
	stats.ActiveSupervisors = int32(activeSupervisors)

	// Count pending topic registrations
	r.db.WithContext(ctx).
		Table("admin_topic_registrations").
		Where("status = ?", StatusPending).
		Count(&pendingTopicRegs)
	stats.PendingTopicRegistrations = int32(pendingTopicRegs)

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
		LEFT JOIN admin_submissions sub ON sub.project_id = p.id AND sub.step_id = s.id
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
		Model(&Submission{}).
		Where("status = ?", StatusPending).
		Count(&count).Error
	return count, err
}

// GetStudentByID - получение полной информации о студенте
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
			COALESCE(t.project_id, 0) as project_id,
			COALESCE(p.title, '') as project_title,
			COALESCE(p.current_state, '') as current_step,
			u.created_at
		FROM users u
		LEFT JOIN team_members tm ON tm.user_id = u.id
		LEFT JOIN teams t ON t.id = tm.team_id AND t.deleted_at IS NULL
		LEFT JOIN projects p ON p.id = t.project_id
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

// GetStudentGrades - получение оценок студента
func (r *repository) GetStudentGrades(ctx context.Context, studentID int64) ([]*Grade, error) {
	var grades []*Grade

	query := `
		SELECT g.*
		FROM admin_grades g
		JOIN projects p ON p.id = g.project_id
		JOIN teams t ON t.project_id = p.id
		JOIN team_members tm ON tm.team_id = t.id
		WHERE tm.user_id = ? AND g.deleted_at IS NULL
		ORDER BY g.created_at DESC
	`

	err := r.db.WithContext(ctx).Raw(query, studentID).Scan(&grades).Error
	return grades, err
}

// GetStudentSubmissions - получение submissions студента
func (r *repository) GetStudentSubmissions(ctx context.Context, studentID int64) ([]*Submission, error) {
	var submissions []*Submission

	query := `
		SELECT s.*
		FROM admin_submissions s
		JOIN projects p ON p.id = s.project_id
		JOIN teams t ON t.project_id = p.id
		JOIN team_members tm ON tm.team_id = t.id
		WHERE tm.user_id = ? AND s.deleted_at IS NULL
		ORDER BY s.created_at DESC
		LIMIT 50
	`

	err := r.db.WithContext(ctx).Raw(query, studentID).Scan(&submissions).Error
	return submissions, err
}

// GetTeamFullDetails - полная информация о команде
func (r *repository) GetTeamFullDetails(ctx context.Context, teamID int64) (*TeamFullDetails, error) {
	var team TeamFullDetails

	// Основная информация о команде
	teamQuery := `
		SELECT 
			t.id,
			t.name,
			COALESCE(t.project_id, 0) as project_id,
			COALESCE(p.title, '') as project_title,
			COALESCE(p.current_state, '') as current_step,
			COALESCE(p.status, 'active') as status,
			COALESCE(sa.supervisor_id, 0) as supervisor_id,
			t.created_at,
			t.updated_at
		FROM teams t
		LEFT JOIN projects p ON p.id = t.project_id
		LEFT JOIN admin_supervisor_assignments sa ON sa.team_id = t.id
		WHERE t.id = ? AND t.deleted_at IS NULL
	`

	if err := r.db.WithContext(ctx).Raw(teamQuery, teamID).Scan(&team).Error; err != nil {
		return nil, err
	}

	if team.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	// Получаем участников команды
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

// UpdateTeamByAdmin - обновление команды администратором
func (r *repository) UpdateTeamByAdmin(ctx context.Context, teamID int64, updates *TeamAdminUpdateData) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Обновляем название команды если указано
		if updates.Name != nil && *updates.Name != "" {
			if err := tx.Table("teams").Where("id = ?", teamID).Update("name", *updates.Name).Error; err != nil {
				return err
			}
		}

		// Обновляем супервайзера если указан
		if updates.SupervisorID != nil && *updates.SupervisorID > 0 {
			// Проверяем существует ли запись
			var count int64
			tx.Table("admin_supervisor_assignments").Where("team_id = ?", teamID).Count(&count)

			if count > 0 {
				// Обновляем
				if err := tx.Table("admin_supervisor_assignments").
					Where("team_id = ?", teamID).
					Updates(map[string]interface{}{
						"supervisor_id": *updates.SupervisorID,
						"updated_at":    time.Now(),
					}).Error; err != nil {
					return err
				}
			} else {
				// Создаём новую запись
				if err := tx.Table("admin_supervisor_assignments").Create(map[string]interface{}{
					"team_id":       teamID,
					"supervisor_id": *updates.SupervisorID,
					"assigned_by":   1, // TODO: передавать актора
					"created_at":    time.Now(),
					"updated_at":    time.Now(),
				}).Error; err != nil {
					return err
				}
			}
		}

		// Обновляем состав команды если указан
		if len(updates.MemberIDs) > 0 {
			// Получаем текущих участников
			var currentMemberIDs []int64
			tx.Table("team_members").Where("team_id = ?", teamID).Pluck("user_id", &currentMemberIDs)

			currentMap := make(map[int64]bool)
			for _, id := range currentMemberIDs {
				currentMap[id] = true
			}

			newMap := make(map[int64]bool)
			for _, id := range updates.MemberIDs {
				newMap[id] = true
			}

			// Удаляем тех, кого нет в новом списке
			for _, id := range currentMemberIDs {
				if !newMap[id] {
					if err := tx.Table("team_members").
						Where("team_id = ? AND user_id = ?", teamID, id).
						Delete(nil).Error; err != nil {
						return err
					}
				}
			}

			// Добавляем новых
			for _, id := range updates.MemberIDs {
				if !currentMap[id] {
					if err := tx.Table("team_members").Create(map[string]interface{}{
						"team_id":    teamID,
						"user_id":    id,
						"role":       "member",
						"created_at": time.Now(),
					}).Error; err != nil {
						return err
					}
				}
			}
		}

		// Обновляем updated_at команды
		return tx.Table("teams").Where("id = ?", teamID).Update("updated_at", time.Now()).Error
	})
}

// DeleteTeamByAdmin - удаление команды администратором
func (r *repository) DeleteTeamByAdmin(ctx context.Context, teamID int64, reason string, deletedBy int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Логируем удаление
		if err := tx.Create(&AdminActivity{
			ActivityType: ActivityTypeTeamDelete,
			Description:  fmt.Sprintf("Team %d deleted. Reason: %s", teamID, reason),
			ActorID:      deletedBy,
			TargetID:     teamID,
			TargetType:   "team",
			CreatedAt:    time.Now(),
		}).Error; err != nil {
			return err
		}

		// Удаляем назначение супервайзера
		tx.Table("admin_supervisor_assignments").Where("team_id = ?", teamID).Delete(nil)

		// Удаляем приглашения
		tx.Table("team_invites").Where("team_id = ?", teamID).Delete(nil)

		// Удаляем участников
		tx.Table("team_members").Where("team_id = ?", teamID).Delete(nil)

		// Мягкое удаление команды
		return tx.Table("teams").Where("id = ?", teamID).Update("deleted_at", time.Now()).Error
	})
}

// AddTeamMember - добавление участника в команду
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

// RemoveTeamMember - удаление участника из команды
func (r *repository) RemoveTeamMember(ctx context.Context, teamID, userID int64) error {
	return r.db.WithContext(ctx).Table("team_members").
		Where("team_id = ? AND user_id = ?", teamID, userID).
		Delete(nil).Error
}

// GetGradingHistoryFull - расширенная история оценок
func (r *repository) GetGradingHistoryFull(ctx context.Context, projectID, stepID int64) ([]*GradeHistoryFull, error) {
	var history []*GradeHistoryFull

	query := `
		SELECT 
			gh.id,
			gh.project_id,
			gh.step_id,
			COALESCE(s.name, '') as step_name,
			gh.old_grade,
			gh.new_grade,
			gh.changed_by,
			CONCAT(u.first_name, ' ', u.last_name) as changer_name,
			COALESCE(gh.reason, '') as reason,
			gh.created_at as changed_at
		FROM admin_grade_history gh
		LEFT JOIN states s ON s.id = gh.step_id
		LEFT JOIN users u ON u.id = gh.changed_by
		WHERE gh.project_id = ?
	`

	args := []interface{}{projectID}

	if stepID > 0 {
		query += " AND gh.step_id = ?"
		args = append(args, stepID)
	}

	query += " ORDER BY gh.created_at DESC"

	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&history).Error
	return history, err
}
