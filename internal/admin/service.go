package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

type Service struct {
	repo           Repository
	authClient     authv1.AuthServiceClient
	projectClient  projectv1.ProjectServiceClient
	teamClient     teamv1.TeamServiceClient
	workflowClient workflowv1.WorkflowServiceClient
	logger         *zap.Logger
}

func NewService(
	repo Repository,
	authClient authv1.AuthServiceClient,
	projectClient projectv1.ProjectServiceClient,
	teamClient teamv1.TeamServiceClient,
	workflowClient workflowv1.WorkflowServiceClient,
	logger *zap.Logger,
) *Service {
	return &Service{
		repo:           repo,
		authClient:     authClient,
		projectClient:  projectClient,
		teamClient:     teamClient,
		workflowClient: workflowClient,
		logger:         logger,
	}
}

// ==================== Topic Registration ====================

// SubmitTopicRegistration - команда отправляет заявление на регистрацию темы
func (s *Service) SubmitTopicRegistration(ctx context.Context, req *SubmitTopicRegistrationRequest) (*TopicRegistration, error) {
	// Проверяем, нет ли уже активного заявления для этой команды
	existing, err := s.repo.GetTopicRegistrationByTeam(ctx, req.TeamID)
	if err == nil && existing != nil {
		if existing.Status == StatusPending {
			return nil, errors.New("у команды уже есть заявление на рассмотрении")
		}
		if existing.Status == StatusApproved {
			return nil, errors.New("тема уже утверждена для этой команды")
		}
	}

	reg := &TopicRegistration{
		ID:               uuid.New().String(),
		TeamID:           req.TeamID,
		ProposedTopic:    req.ProposedTopic,
		TopicDescription: req.TopicDescription,
		SupervisorID:     req.SupervisorID,
		SubmittedBy:      req.SubmittedBy,
		Status:           StatusPending,
	}

	if err := s.repo.CreateTopicRegistration(ctx, reg); err != nil {
		return nil, fmt.Errorf("не удалось создать заявление: %w", err)
	}

	// Создаём запись в истории
	review := &TopicRegistrationReview{
		RegistrationID: reg.ID,
		ReviewerID:     req.SubmittedBy,
		Action:         "submitted",
		Comment:        "Заявление подано",
	}
	_ = s.repo.CreateTopicRegistrationReview(ctx, review)

	// Логируем активность
	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: ActivityTypeTopicRegistration,
		Description:  fmt.Sprintf("Заявление на тему '%s' подано командой %d", req.ProposedTopic, req.TeamID),
		ActorID:      req.SubmittedBy,
		TargetID:     req.TeamID,
		TargetType:   "team",
	})

	s.logger.Info("Topic registration submitted",
		zap.String("registration_id", reg.ID),
		zap.Int64("team_id", req.TeamID),
		zap.String("topic", req.ProposedTopic))

	return reg, nil
}

type SubmitTopicRegistrationRequest struct {
	TeamID           int64
	ProposedTopic    string
	TopicDescription string
	SupervisorID     int64
	SubmittedBy      int64
}

// ReviewTopicRegistration - комиссия проверяет заявление
func (s *Service) ReviewTopicRegistration(ctx context.Context, req *ReviewTopicRegistrationRequest) (*TopicRegistration, error) {
	reg, err := s.repo.GetTopicRegistration(ctx, req.RegistrationID)
	if err != nil {
		return nil, fmt.Errorf("заявление не найдено: %w", err)
	}

	if reg.Status != StatusPending && reg.Status != StatusRevisionRequested {
		return nil, errors.New("заявление не может быть рассмотрено в текущем статусе")
	}

	now := time.Now()
	reg.ReviewerID = &req.ReviewerID
	reg.ReviewedAt = &now
	reg.Comment = req.Comment

	switch req.Action {
	case "approve":
		reg.Status = StatusApproved
		// Здесь можно добавить логику создания проекта или обновления существующего
		s.logger.Info("Topic registration approved",
			zap.String("registration_id", req.RegistrationID),
			zap.Int64("team_id", reg.TeamID))

	case "reject":
		reg.Status = StatusRejected
		reg.RejectionReason = req.RejectionReason

	case "request_changes":
		reg.Status = StatusRevisionRequested

	default:
		return nil, errors.New("недопустимое действие")
	}

	if err := s.repo.UpdateTopicRegistration(ctx, reg); err != nil {
		return nil, fmt.Errorf("не удалось обновить заявление: %w", err)
	}

	// Создаём запись в истории
	review := &TopicRegistrationReview{
		RegistrationID: reg.ID,
		ReviewerID:     req.ReviewerID,
		Action:         req.Action,
		Comment:        req.Comment,
	}
	_ = s.repo.CreateTopicRegistrationReview(ctx, review)

	// Логируем активность
	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: ActivityTypeTopicApproval,
		Description:  fmt.Sprintf("Заявление %s: %s", reg.ID, req.Action),
		ActorID:      req.ReviewerID,
		TargetID:     reg.TeamID,
		TargetType:   "topic_registration",
	})

	return reg, nil
}

type ReviewTopicRegistrationRequest struct {
	RegistrationID  string
	ReviewerID      int64
	Action          string // approve, reject, request_changes
	Comment         string
	RejectionReason string
}

// ListTopicRegistrations - список заявлений на регистрацию темы
func (s *Service) ListTopicRegistrations(ctx context.Context, filter TopicRegistrationFilter) ([]*TopicRegistration, int64, error) {
	return s.repo.ListTopicRegistrations(ctx, filter)
}

// GetTopicRegistration - получить заявление с историей
func (s *Service) GetTopicRegistration(ctx context.Context, id string) (*TopicRegistration, []*TopicRegistrationReview, error) {
	reg, err := s.repo.GetTopicRegistration(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	reviews, _ := s.repo.GetTopicRegistrationReviews(ctx, id)
	return reg, reviews, nil
}

// ==================== Dashboard ====================
func (s *Service) GetDashboard(ctx context.Context, departmentID int64) (*DashboardResponse, error) {
	stats, err := s.repo.GetDashboardStats(ctx, departmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard stats: %w", err)
	}

	// Get active workflow for department
	wf, err := s.workflowClient.GetActiveWorkflowByDepartment(ctx, &workflowv1.GetActiveWorkflowByDepartmentRequest{
		DepartmentId: departmentID,
	})
	if err != nil {
		s.logger.Warn("No active workflow found", zap.Error(err))
	}

	var stepProgress []*StepProgressData
	if wf != nil {
		stepProgress, _ = s.repo.GetStepProgressStats(ctx, departmentID, wf.Id)
	}

	activities, _ := s.repo.GetRecentActivities(ctx, departmentID, 10)

	return &DashboardResponse{
		Stats:        stats,
		StepProgress: stepProgress,
		Activities:   activities,
	}, nil
}

type DashboardResponse struct {
	Stats        *DashboardStatsData
	StepProgress []*StepProgressData
	Activities   []*AdminActivity
}

// ==================== Students ====================
func (s *Service) ListStudents(ctx context.Context, req *ListStudentsRequest) ([]*StudentData, int64, error) {
	resp, err := s.authClient.ListUsers(ctx, &authv1.ListUsersRequest{
		UniversityId: req.UniversityID,
		Role:         "student",
		Page:         req.Page,
		PageSize:     req.PageSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list students: %w", err)
	}

	var students []*StudentData
	for _, u := range resp.Users {
		student := &StudentData{
			ID:        u.Id,
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
		}
		// Get team info
		teamResp, err := s.teamClient.GetMyTeam(ctx, &teamv1.GetMyTeamRequest{UserId: u.Id})
		if err == nil && teamResp.HasTeam {
			student.TeamID = teamResp.Team.TeamId
			student.TeamName = teamResp.Team.Name
			student.ProjectID = teamResp.Team.ProjectId
		}
		students = append(students, student)
	}
	return students, resp.TotalCount, nil
}

type ListStudentsRequest struct {
	UniversityID    int64
	DepartmentID    int64
	Search          string
	OnlyWithoutTeam bool
	Page            int32
	PageSize        int32
}

type StudentData struct {
	ID           int64
	Email        string
	FirstName    string
	LastName     string
	TeamID       int64
	TeamName     string
	ProjectID    int64
	ProjectTitle string
	CurrentStep  string
}

// ==================== Teams ====================
func (s *Service) ListAllTeams(ctx context.Context, req *ListAllTeamsRequest) ([]*TeamAdminData, int64, error) {
	resp, err := s.teamClient.ListTeams(ctx, &teamv1.ListTeamsRequest{
		DepartmentId: req.DepartmentID,
		ProjectId:    req.ProjectID,
		Page:         req.Page,
		PageSize:     req.PageSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list teams: %w", err)
	}

	var teams []*TeamAdminData
	for _, t := range resp.Teams {
		team := &TeamAdminData{
			ID:          t.Id,
			Name:        t.Name,
			ProjectID:   t.ProjectId,
			MemberCount: int32(len(t.Members)),
		}

		// Get supervisor assignment
		assignment, err := s.repo.GetSupervisorAssignment(ctx, t.Id)
		if err == nil && assignment != nil {
			team.SupervisorID = assignment.SupervisorID
		}

		// Get topic registration status
		topicReg, err := s.repo.GetTopicRegistrationByTeam(ctx, t.Id)
		if err == nil && topicReg != nil {
			team.TopicStatus = topicReg.Status
			team.ProposedTopic = topicReg.ProposedTopic
		}

		// Get project info
		if t.ProjectId > 0 {
			projResp, err := s.projectClient.GetProject(ctx, &projectv1.GetProjectRequest{ProjectId: t.ProjectId})
			if err == nil {
				team.ProjectTitle = projResp.Title
				team.CurrentStep = projResp.CurrentState
			}
		}

		teams = append(teams, team)
	}

	return teams, resp.TotalCount, nil
}

type ListAllTeamsRequest struct {
	DepartmentID int64
	ProjectID    int64
	Status       string
	Search       string
	Page         int32
	PageSize     int32
}

type TeamAdminData struct {
	ID            int64
	Name          string
	ProjectID     int64
	ProjectTitle  string
	CurrentStep   string
	MemberCount   int32
	SupervisorID  int64
	Status        string
	TopicStatus   string
	ProposedTopic string
}

// ==================== Submissions ====================
func (s *Service) CreateSubmission(ctx context.Context, req *CreateSubmissionRequest) (*Submission, error) {
	dataBytes, err := json.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	filesBytes, err := json.Marshal(req.FileIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal files: %w", err)
	}

	sub := &Submission{
		ID:          uuid.New().String(),
		ProjectID:   req.ProjectID,
		TeamID:      req.TeamID,
		StepID:      req.StepID,
		SubmittedBy: req.SubmittedBy,
		Status:      StatusPending,
		Data:        datatypes.JSON(dataBytes),
		Files:       datatypes.JSON(filesBytes),
	}

	if err := s.repo.CreateSubmission(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}

	// Create initial review entry
	review := &SubmissionReview{
		SubmissionID: sub.ID,
		ReviewerID:   req.SubmittedBy,
		Action:       "submitted",
	}
	_ = s.repo.CreateSubmissionReview(ctx, review)

	// Log activity
	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: ActivityTypeSubmission,
		Description:  fmt.Sprintf("Submission created for project %d, step %d", req.ProjectID, req.StepID),
		ActorID:      req.SubmittedBy,
		TargetID:     req.ProjectID,
		TargetType:   "project",
	})

	return sub, nil
}

type CreateSubmissionRequest struct {
	ProjectID   int64
	TeamID      int64
	StepID      int64
	SubmittedBy int64
	Data        map[string]interface{}
	FileIDs     []string
}

func (s *Service) ReviewSubmission(ctx context.Context, req *ReviewSubmissionRequest) (*Submission, error) {
	sub, err := s.repo.GetSubmission(ctx, req.SubmissionID)
	if err != nil {
		return nil, fmt.Errorf("submission not found: %w", err)
	}

	if sub.Status != StatusPending && sub.Status != StatusRevisionRequested {
		return nil, errors.New("submission cannot be reviewed in current status")
	}

	now := time.Now()
	sub.ReviewerID = &req.ReviewerID
	sub.ReviewComment = req.Comment
	sub.ReviewedAt = &now

	switch req.Action {
	case "approve":
		sub.Status = StatusApproved
		// Если указана оценка, создаём запись об оценке
		if req.Grade > 0 {
			grade := &Grade{
				ProjectID: sub.ProjectID,
				StepID:    sub.StepID,
				TeamID:    sub.TeamID,
				Grade:     req.Grade,
				Comment:   req.Comment,
				GradedBy:  req.ReviewerID,
			}
			_ = s.repo.CreateGrade(ctx, grade)
		}
	case "reject":
		sub.Status = StatusRejected
	case "request_changes":
		sub.Status = StatusRevisionRequested
	default:
		return nil, errors.New("invalid action")
	}

	if err := s.repo.UpdateSubmission(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to update submission: %w", err)
	}

	// Create review history
	review := &SubmissionReview{
		SubmissionID: sub.ID,
		ReviewerID:   req.ReviewerID,
		Action:       req.Action,
		Comment:      req.Comment,
	}
	if req.Grade > 0 {
		review.Grade = &req.Grade
	}
	_ = s.repo.CreateSubmissionReview(ctx, review)

	// Log activity
	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: ActivityTypeReview,
		Description:  fmt.Sprintf("Submission %s: %s", sub.ID, req.Action),
		ActorID:      req.ReviewerID,
		TargetID:     sub.ProjectID,
		TargetType:   "submission",
	})

	return sub, nil
}

type ReviewSubmissionRequest struct {
	SubmissionID string
	ReviewerID   int64
	Action       string
	Comment      string
	Grade        int32
}

func (s *Service) ListSubmissions(ctx context.Context, filter SubmissionFilter) ([]*Submission, int64, error) {
	return s.repo.ListSubmissions(ctx, filter)
}

func (s *Service) GetSubmission(ctx context.Context, id string) (*Submission, []*SubmissionReview, error) {
	sub, err := s.repo.GetSubmission(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	reviews, _ := s.repo.GetSubmissionReviews(ctx, id)
	return sub, reviews, nil
}

// ==================== Grading (только баллы) ====================
func (s *Service) SetStepGrade(ctx context.Context, req *SetGradeRequest) (*Grade, error) {
	// Проверяем валидность оценки
	if req.Grade < 0 || req.Grade > 100 {
		return nil, errors.New("оценка должна быть от 0 до 100 баллов")
	}

	// Check if grade exists
	existing, err := s.repo.GetGrade(ctx, req.ProjectID, req.StepID)
	if err == nil && existing != nil {
		// Update existing grade
		oldGrade := existing.Grade
		existing.Grade = req.Grade
		existing.Comment = req.Comment
		existing.GradedBy = req.GraderID

		if err := s.repo.UpdateGrade(ctx, existing); err != nil {
			return nil, fmt.Errorf("failed to update grade: %w", err)
		}

		// Record history
		_ = s.repo.CreateGradeHistory(ctx, &GradeHistory{
			GradeID:   existing.ID,
			ProjectID: req.ProjectID,
			StepID:    req.StepID,
			OldGrade:  oldGrade,
			NewGrade:  req.Grade,
			ChangedBy: req.GraderID,
			Reason:    "Grade updated",
		})

		return existing, nil
	}

	// Create new grade
	grade := &Grade{
		ProjectID: req.ProjectID,
		StepID:    req.StepID,
		Grade:     req.Grade,
		Comment:   req.Comment,
		GradedBy:  req.GraderID,
	}

	if err := s.repo.CreateGrade(ctx, grade); err != nil {
		return nil, fmt.Errorf("failed to create grade: %w", err)
	}

	// Log activity
	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: ActivityTypeGrade,
		Description:  fmt.Sprintf("Оценка %d баллов выставлена для проекта %d, этап %d", req.Grade, req.ProjectID, req.StepID),
		ActorID:      req.GraderID,
		TargetID:     req.ProjectID,
		TargetType:   "project",
	})

	return grade, nil
}

type SetGradeRequest struct {
	ProjectID int64
	StepID    int64
	Grade     int32
	Comment   string
	GraderID  int64
}

func (s *Service) GetProjectGrades(ctx context.Context, projectID int64) ([]*Grade, float32, error) {
	grades, err := s.repo.GetGradesByProject(ctx, projectID)
	if err != nil {
		return nil, 0, err
	}

	var total int32
	for _, g := range grades {
		total += g.Grade
	}

	var avg float32
	if len(grades) > 0 {
		avg = float32(total) / float32(len(grades))
	}

	return grades, avg, nil
}

// ==================== Supervisors ====================
func (s *Service) AssignSupervisor(ctx context.Context, teamID, supervisorID, assignedBy int64) error {
	existing, err := s.repo.GetSupervisorAssignment(ctx, teamID)
	if err == nil && existing != nil {
		// Update existing
		existing.SupervisorID = supervisorID
		existing.AssignedBy = assignedBy
		return s.repo.UpdateSupervisorAssignment(ctx, existing)
	}

	// Create new
	assignment := &SupervisorAssignment{
		TeamID:       teamID,
		SupervisorID: supervisorID,
		AssignedBy:   assignedBy,
	}

	if err := s.repo.AssignSupervisor(ctx, assignment); err != nil {
		return fmt.Errorf("failed to assign supervisor: %w", err)
	}

	// Log activity
	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: ActivityTypeSupervisorAssign,
		Description:  fmt.Sprintf("Supervisor %d assigned to team %d", supervisorID, teamID),
		ActorID:      assignedBy,
		TargetID:     teamID,
		TargetType:   "team",
	})

	return nil
}

func (s *Service) ListSupervisors(ctx context.Context, departmentID, universityID int64, page, pageSize int32) ([]*SupervisorData, int64, error) {
	resp, err := s.authClient.ListUsers(ctx, &authv1.ListUsersRequest{
		UniversityId: universityID,
		Role:         "teacher",
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list supervisors: %w", err)
	}

	var supervisors []*SupervisorData
	for _, u := range resp.Users {
		sup := &SupervisorData{
			ID:       u.Id,
			FullName: u.FirstName + " " + u.LastName,
			Email:    u.Email,
			MaxTeams: 5,
		}

		count, _ := s.repo.CountTeamsBySupervisor(ctx, u.Id)
		sup.TeamsCount = int32(count)

		teamIDs, _ := s.repo.GetTeamsBySupervisor(ctx, u.Id)
		sup.AssignedTeamIDs = teamIDs

		supervisors = append(supervisors, sup)
	}

	return supervisors, resp.TotalCount, nil
}

type SupervisorData struct {
	ID              int64
	FullName        string
	Email           string
	Position        string
	TeamsCount      int32
	MaxTeams        int32
	AssignedTeamIDs []int64
}

// ==================== Workflow Progress ====================
func (s *Service) GetWorkflowProgress(ctx context.Context, departmentID, workflowID int64) ([]*StepProgressData, error) {
	return s.repo.GetStepProgressStats(ctx, departmentID, workflowID)
}

func (s *Service) ListPendingReviews(ctx context.Context, departmentID int64, page, pageSize int32) ([]*Submission, int64, error) {
	filter := SubmissionFilter{
		DepartmentID: departmentID,
		Status:       StatusPending,
		Limit:        int(pageSize),
		Offset:       int((page - 1) * pageSize),
	}
	return s.repo.ListSubmissions(ctx, filter)
}
