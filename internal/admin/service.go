package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
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

func (s *Service) ListAllTeams(ctx context.Context, req *ListAllTeamsRequest) ([]*TeamData, int64, error) {
	resp, err := s.teamClient.ListTeams(ctx, &teamv1.ListTeamsRequest{
		DepartmentId: req.DepartmentID,
		Page:         req.Page,
		PageSize:     req.PageSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list teams: %w", err)
	}

	var teams []*TeamData
	for _, t := range resp.Teams {
		team := &TeamData{
			ID:          t.Id,
			Name:        t.Name,
			MemberCount: int32(len(t.Members)),
			Status:      "active",
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

type TeamData struct {
	ID           int64
	Name         string
	ProjectID    int64
	ProjectTitle string
	CurrentStep  string
	MemberCount  int32
	Status       string
}

// ==================== Supervisors ====================

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
		teamsCount, _ := s.repo.CountTeamsBySupervisor(ctx, u.Id)

		supervisors = append(supervisors, &SupervisorData{
			ID:         u.Id,
			FullName:   u.FirstName + " " + u.LastName,
			Email:      u.Email,
			Position:   "Senior Lecturer",
			TeamsCount: int32(teamsCount),
			MaxTeams:   5,
		})
	}

	return supervisors, resp.TotalCount, nil
}

type SupervisorData struct {
	ID         int64
	FullName   string
	Email      string
	Position   string
	TeamsCount int32
	MaxTeams   int32
}

func (s *Service) AssignSupervisor(ctx context.Context, teamID, supervisorID, assignedBy int64) error {
	existing, err := s.repo.GetSupervisorAssignment(ctx, teamID)
	if err == nil && existing != nil {
		existing.SupervisorID = supervisorID
		existing.AssignedBy = assignedBy
		return s.repo.UpdateSupervisorAssignment(ctx, existing)
	}

	assignment := &SupervisorAssignment{
		TeamID:       teamID,
		SupervisorID: supervisorID,
		AssignedBy:   assignedBy,
	}

	if err := s.repo.AssignSupervisor(ctx, assignment); err != nil {
		return fmt.Errorf("failed to assign supervisor: %w", err)
	}

	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: ActivityTypeSupervisorAssign,
		Description:  fmt.Sprintf("Supervisor %d assigned to team %d", supervisorID, teamID),
		ActorID:      assignedBy,
		TargetID:     teamID,
		TargetType:   "team",
	})

	return nil
}

// ==================== Submissions ====================

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

func (s *Service) ReviewSubmission(ctx context.Context, req *ReviewSubmissionRequest) (*Submission, error) {
	sub, err := s.repo.GetSubmission(ctx, req.SubmissionID)
	if err != nil {
		return nil, fmt.Errorf("submission not found: %w", err)
	}

	now := time.Now()
	sub.ReviewerID = &req.ReviewerID
	sub.ReviewedAt = &now
	sub.ReviewComment = req.Comment

	switch req.Action {
	case "approve":
		sub.Status = StatusApproved
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

	// Create review record
	review := &SubmissionReview{
		SubmissionID: sub.ID,
		ReviewerID:   req.ReviewerID,
		Action:       req.Action,
		Comment:      req.Comment,
		Grade:        &req.Grade,
	}
	_ = s.repo.CreateSubmissionReview(ctx, review)

	// If grade provided and approved, create/update grade
	if req.Grade > 0 && req.Action == "approve" {
		_, _ = s.SetStepGrade(ctx, &SetGradeRequest{
			ProjectID: sub.ProjectID,
			StepID:    sub.StepID,
			Grade:     req.Grade,
			Comment:   req.Comment,
			GraderID:  req.ReviewerID,
		})
	}

	return sub, nil
}

type ReviewSubmissionRequest struct {
	SubmissionID string
	ReviewerID   int64
	Action       string
	Comment      string
	Grade        int32
}

// ==================== Grading (только баллы) ====================

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

func (s *Service) SetStepGrade(ctx context.Context, req *SetGradeRequest) (*Grade, error) {
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

		// Create history record
		history := &GradeHistory{
			GradeID:   existing.ID,
			ProjectID: req.ProjectID,
			StepID:    req.StepID,
			OldGrade:  oldGrade,
			NewGrade:  req.Grade,
			ChangedBy: req.GraderID,
			Reason:    req.Comment,
		}
		_ = s.repo.CreateGradeHistory(ctx, history)

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

	// Create history record
	history := &GradeHistory{
		GradeID:   grade.ID,
		ProjectID: req.ProjectID,
		StepID:    req.StepID,
		OldGrade:  0,
		NewGrade:  req.Grade,
		ChangedBy: req.GraderID,
		Reason:    "Initial grade",
	}
	_ = s.repo.CreateGradeHistory(ctx, history)

	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: ActivityTypeGrade,
		Description:  fmt.Sprintf("Grade %d set for project %d, step %d", req.Grade, req.ProjectID, req.StepID),
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

// ==================== Workflow Progress ====================

func (s *Service) GetWorkflowProgress(ctx context.Context, departmentID, workflowID int64) ([]*StepProgressData, error) {
	if workflowID == 0 {
		// Get active workflow
		wf, err := s.workflowClient.GetActiveWorkflowByDepartment(ctx, &workflowv1.GetActiveWorkflowByDepartmentRequest{
			DepartmentId: departmentID,
		})
		if err != nil {
			return nil, fmt.Errorf("no active workflow found: %w", err)
		}
		workflowID = wf.Id
	}
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

// ==================== Supervisor Request Service ====================

// CreateSupervisorRequest - создание запроса команды к супервайзеру
func (s *Service) CreateSupervisorRequest(ctx context.Context, req *CreateSupervisorRequestInput) (*SupervisorRequestWithDetails, error) {
	// Проверяем, нет ли уже ожидающего запроса от этой команды
	hasPending, err := s.repo.HasPendingSupervisorRequest(ctx, req.TeamID)
	if err != nil {
		return nil, fmt.Errorf("failed to check pending requests: %w", err)
	}
	if hasPending {
		return nil, errors.New("у команды уже есть активный запрос к супервайзеру")
	}

	// Проверяем, нет ли уже утверждённого супервайзера
	hasApproved, err := s.repo.HasApprovedSupervisor(ctx, req.TeamID)
	if err != nil {
		return nil, fmt.Errorf("failed to check approved supervisor: %w", err)
	}
	if hasApproved {
		return nil, errors.New("у команды уже есть утверждённый научный руководитель")
	}

	// Устанавливаем срок действия запроса (7 дней)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	supervisorRequest := &SupervisorRequest{
		ID:            uuid.New().String(),
		TeamID:        req.TeamID,
		SupervisorID:  req.SupervisorID,
		RequestedBy:   req.RequestedBy,
		Status:        SupervisorRequestStatusPending,
		Message:       req.Message,
		ProposedTopic: req.ProposedTopic,
		ExpiresAt:     &expiresAt,
	}

	if err := s.repo.CreateSupervisorRequest(ctx, supervisorRequest); err != nil {
		return nil, fmt.Errorf("failed to create supervisor request: %w", err)
	}

	// Создаём запись в истории
	history := &SupervisorRequestHistory{
		RequestID: supervisorRequest.ID,
		Action:    SupervisorRequestActionCreated,
		ActorID:   req.RequestedBy,
		Comment:   "Запрос создан",
	}
	_ = s.repo.CreateSupervisorRequestHistory(ctx, history)

	// Логируем активность
	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: ActivityTypeSupervisorRequest,
		Description:  fmt.Sprintf("Команда %d отправила запрос супервайзеру %d", req.TeamID, req.SupervisorID),
		ActorID:      req.RequestedBy,
		TargetID:     req.SupervisorID,
		TargetType:   "supervisor_request",
	})

	s.logger.Info("Supervisor request created",
		zap.String("request_id", supervisorRequest.ID),
		zap.Int64("team_id", req.TeamID),
		zap.Int64("supervisor_id", req.SupervisorID))

	// Возвращаем запрос с деталями
	return s.repo.GetSupervisorRequestWithDetails(ctx, supervisorRequest.ID)
}

// CreateSupervisorRequestInput - входные данные для создания запроса
type CreateSupervisorRequestInput struct {
	TeamID        int64
	SupervisorID  int64
	RequestedBy   int64
	Message       string
	ProposedTopic string
}

// GetSupervisorRequest - получение запроса с историей
func (s *Service) GetSupervisorRequest(ctx context.Context, requestID string) (*SupervisorRequestWithDetails, []*SupervisorRequestHistory, error) {
	req, err := s.repo.GetSupervisorRequestWithDetails(ctx, requestID)
	if err != nil {
		return nil, nil, fmt.Errorf("supervisor request not found: %w", err)
	}

	history, _ := s.repo.GetSupervisorRequestHistory(ctx, requestID)

	return req, history, nil
}

// ListSupervisorRequests - список запросов с фильтрацией
func (s *Service) ListSupervisorRequests(ctx context.Context, filter SupervisorRequestFilter) ([]*SupervisorRequestWithDetails, int64, error) {
	return s.repo.ListSupervisorRequests(ctx, filter)
}

// ListMySupervisorRequests - входящие запросы для супервайзера
func (s *Service) ListMySupervisorRequests(ctx context.Context, supervisorID int64, status string, page, pageSize int32) (*MySupervisorRequestsResponse, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}

	filter := SupervisorRequestFilter{
		SupervisorID: supervisorID,
		Status:       status,
		Limit:        int(pageSize),
		Offset:       int((page - 1) * pageSize),
	}

	requests, total, err := s.repo.ListSupervisorRequests(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list supervisor requests: %w", err)
	}

	pendingCount, _ := s.repo.CountPendingSupervisorRequests(ctx, supervisorID)

	return &MySupervisorRequestsResponse{
		Requests:     requests,
		TotalCount:   total,
		PendingCount: int32(pendingCount),
	}, nil
}

// MySupervisorRequestsResponse - ответ со списком запросов для супервайзера
type MySupervisorRequestsResponse struct {
	Requests     []*SupervisorRequestWithDetails
	TotalCount   int64
	PendingCount int32
}

// RespondToSupervisorRequest - ответ супервайзера на запрос
func (s *Service) RespondToSupervisorRequest(ctx context.Context, req *RespondToSupervisorRequestInput) (*SupervisorRequestWithDetails, error) {
	// Получаем запрос
	supervisorReq, err := s.repo.GetSupervisorRequest(ctx, req.RequestID)
	if err != nil {
		return nil, fmt.Errorf("supervisor request not found: %w", err)
	}

	// Проверяем, что отвечает правильный супервайзер
	if supervisorReq.SupervisorID != req.SupervisorID {
		return nil, errors.New("вы не можете ответить на этот запрос")
	}

	// Проверяем статус
	if supervisorReq.Status != SupervisorRequestStatusPending {
		return nil, errors.New("запрос уже обработан")
	}

	now := time.Now()
	supervisorReq.RespondedAt = &now

	switch req.Action {
	case "approve":
		supervisorReq.Status = SupervisorRequestStatusApproved

		// Автоматически назначаем супервайзера команде
		err := s.AssignSupervisor(ctx, supervisorReq.TeamID, supervisorReq.SupervisorID, supervisorReq.SupervisorID)
		if err != nil {
			s.logger.Warn("Failed to auto-assign supervisor",
				zap.Error(err),
				zap.Int64("team_id", supervisorReq.TeamID))
		}

		s.logger.Info("Supervisor request approved",
			zap.String("request_id", req.RequestID),
			zap.Int64("supervisor_id", req.SupervisorID))

	case "reject":
		supervisorReq.Status = SupervisorRequestStatusRejected
		supervisorReq.RejectReason = req.RejectReason

		s.logger.Info("Supervisor request rejected",
			zap.String("request_id", req.RequestID),
			zap.Int64("supervisor_id", req.SupervisorID),
			zap.String("reason", req.RejectReason))

	default:
		return nil, errors.New("недопустимое действие")
	}

	if err := s.repo.UpdateSupervisorRequest(ctx, supervisorReq); err != nil {
		return nil, fmt.Errorf("failed to update supervisor request: %w", err)
	}

	// Создаём запись в истории
	history := &SupervisorRequestHistory{
		RequestID: supervisorReq.ID,
		Action:    req.Action,
		ActorID:   req.SupervisorID,
		Comment:   req.Comment,
	}
	if req.Action == "reject" && req.RejectReason != "" {
		history.Comment = req.RejectReason
	}
	_ = s.repo.CreateSupervisorRequestHistory(ctx, history)

	// Логируем активность
	activityType := ActivityTypeSupervisorRequestApproved
	if req.Action == "reject" {
		activityType = ActivityTypeSupervisorRequestRejected
	}
	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: activityType,
		Description:  fmt.Sprintf("Супервайзер %d %s запрос %s", req.SupervisorID, req.Action, req.RequestID),
		ActorID:      req.SupervisorID,
		TargetID:     supervisorReq.TeamID,
		TargetType:   "supervisor_request",
	})

	return s.repo.GetSupervisorRequestWithDetails(ctx, supervisorReq.ID)
}

// RespondToSupervisorRequestInput - входные данные для ответа на запрос
type RespondToSupervisorRequestInput struct {
	RequestID    string
	SupervisorID int64
	Action       string // approve, reject
	RejectReason string
	Comment      string
}

// CancelSupervisorRequest - отмена запроса командой
func (s *Service) CancelSupervisorRequest(ctx context.Context, requestID string, cancelledBy int64, reason string) error {
	// Получаем запрос
	supervisorReq, err := s.repo.GetSupervisorRequest(ctx, requestID)
	if err != nil {
		return fmt.Errorf("supervisor request not found: %w", err)
	}

	// Проверяем статус
	if supervisorReq.Status != SupervisorRequestStatusPending {
		return errors.New("можно отменить только ожидающий запрос")
	}

	// Обновляем статус
	now := time.Now()
	supervisorReq.Status = SupervisorRequestStatusCancelled
	supervisorReq.RespondedAt = &now

	if err := s.repo.UpdateSupervisorRequest(ctx, supervisorReq); err != nil {
		return fmt.Errorf("failed to cancel supervisor request: %w", err)
	}

	// Создаём запись в истории
	comment := "Запрос отменён"
	if reason != "" {
		comment = reason
	}
	history := &SupervisorRequestHistory{
		RequestID: supervisorReq.ID,
		Action:    SupervisorRequestActionCancelled,
		ActorID:   cancelledBy,
		Comment:   comment,
	}
	_ = s.repo.CreateSupervisorRequestHistory(ctx, history)

	s.logger.Info("Supervisor request cancelled",
		zap.String("request_id", requestID),
		zap.Int64("cancelled_by", cancelledBy))

	return nil
}

// ==================== Приоритет 1 — Дореализация ====================

// GetStudentFullInfo - получение детальной информации о студенте
func (s *Service) GetStudentFullInfo(ctx context.Context, studentID int64) (*StudentDetailResponse, error) {
	// Получаем основную информацию о студенте
	student, err := s.repo.GetStudentByID(ctx, studentID)
	if err != nil {
		return nil, fmt.Errorf("student not found: %w", err)
	}

	// Получаем оценки студента
	grades, _ := s.repo.GetStudentGrades(ctx, studentID)

	// Получаем submissions студента
	submissions, _ := s.repo.GetStudentSubmissions(ctx, studentID)

	return &StudentDetailResponse{
		Student:     student,
		Grades:      grades,
		Submissions: submissions,
	}, nil
}

// StudentDetailResponse - ответ с полной информацией о студенте
type StudentDetailResponse struct {
	Student     *StudentFullInfo
	Grades      []*Grade
	Submissions []*Submission
}

// GetTeamFullDetails - получение детальной информации о команде
func (s *Service) GetTeamFullDetails(ctx context.Context, teamID int64) (*TeamDetailResponse, error) {
	// Получаем полную информацию о команде
	team, err := s.repo.GetTeamFullDetails(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("team not found: %w", err)
	}

	response := &TeamDetailResponse{
		Team: team,
	}

	// Получаем информацию о супервайзере
	if team.SupervisorID > 0 {
		usersResp, err := s.authClient.ListUsers(ctx, &authv1.ListUsersRequest{
			Role:     "teacher",
			Page:     1,
			PageSize: 100,
		})
		if err == nil {
			for _, u := range usersResp.Users {
				if u.Id == team.SupervisorID {
					response.Supervisor = &SupervisorBasicInfo{
						ID:       u.Id,
						FullName: u.FirstName + " " + u.LastName,
						Email:    u.Email,
					}
					break
				}
			}
		}
	}

	// Получаем оценки проекта если есть
	if team.ProjectID > 0 {
		grades, _, _ := s.GetProjectGrades(ctx, team.ProjectID)
		response.Grades = grades

		// Получаем submissions
		submissions, _, _ := s.repo.ListSubmissions(ctx, SubmissionFilter{
			TeamID: teamID,
			Limit:  50,
			Offset: 0,
		})
		response.Submissions = submissions
	}

	// Получаем заявление на регистрацию темы
	topicReg, _ := s.repo.GetTopicRegistrationByTeam(ctx, teamID)
	response.TopicRegistration = topicReg

	return response, nil
}

// TeamDetailResponse - ответ с полной информацией о команде
type TeamDetailResponse struct {
	Team              *TeamFullDetails
	Supervisor        *SupervisorBasicInfo
	Grades            []*Grade
	Submissions       []*Submission
	TopicRegistration *TopicRegistration
}

// UpdateTeamByAdmin - обновление команды администратором
func (s *Service) UpdateTeamByAdmin(ctx context.Context, req *UpdateTeamByAdminRequest) (*TeamFullDetails, error) {
	// Проверяем существование команды
	_, err := s.repo.GetTeamFullDetails(ctx, req.TeamID)
	if err != nil {
		return nil, fmt.Errorf("team not found: %w", err)
	}

	// Подготавливаем обновления
	updates := &TeamAdminUpdateData{}

	if req.Name != "" {
		updates.Name = &req.Name
	}

	if req.SupervisorID > 0 {
		updates.SupervisorID = &req.SupervisorID
	}

	if len(req.MemberIDs) > 0 {
		updates.MemberIDs = req.MemberIDs
	}

	// Выполняем обновление
	if err := s.repo.UpdateTeamByAdmin(ctx, req.TeamID, updates); err != nil {
		return nil, fmt.Errorf("failed to update team: %w", err)
	}

	// Логируем активность
	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: ActivityTypeTeamUpdate,
		Description:  fmt.Sprintf("Team %d updated by admin", req.TeamID),
		ActorID:      req.UpdatedBy,
		TargetID:     req.TeamID,
		TargetType:   "team",
	})

	s.logger.Info("Team updated by admin",
		zap.Int64("team_id", req.TeamID),
		zap.Int64("updated_by", req.UpdatedBy))

	// Возвращаем обновлённую команду
	return s.repo.GetTeamFullDetails(ctx, req.TeamID)
}

// UpdateTeamByAdminRequest - запрос на обновление команды
type UpdateTeamByAdminRequest struct {
	TeamID       int64
	Name         string
	SupervisorID int64
	MemberIDs    []int64
	UpdatedBy    int64
}

// DeleteTeamByAdmin - удаление команды администратором
func (s *Service) DeleteTeamByAdmin(ctx context.Context, teamID int64, reason string, deletedBy int64) error {
	// Проверяем существование команды
	team, err := s.repo.GetTeamFullDetails(ctx, teamID)
	if err != nil {
		return fmt.Errorf("team not found: %w", err)
	}

	// Проверяем, нет ли активного проекта
	if team.ProjectID > 0 && team.Status == "active" {
		s.logger.Warn("Deleting team with active project",
			zap.Int64("team_id", teamID),
			zap.Int64("project_id", team.ProjectID))
	}

	// Удаляем команду
	if err := s.repo.DeleteTeamByAdmin(ctx, teamID, reason, deletedBy); err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}

	s.logger.Info("Team deleted by admin",
		zap.Int64("team_id", teamID),
		zap.String("reason", reason),
		zap.Int64("deleted_by", deletedBy))

	return nil
}

// GetGradingHistoryFull - получение расширенной истории изменений оценок
func (s *Service) GetGradingHistoryFull(ctx context.Context, projectID, stepID int64) ([]*GradeHistoryFull, error) {
	return s.repo.GetGradingHistoryFull(ctx, projectID, stepID)
}
