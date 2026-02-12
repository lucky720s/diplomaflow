package admin

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
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

// ==================== Helpers ====================

func (s *Service) internalCtx(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "x-internal-service", "admin_service")
}

// extract user_id/role from incoming ctx (gateway -> admin_service), fallback if absent.
func (s *Service) callerFromContext(ctx context.Context, fallbackUserID int64, fallbackRole string) (int64, string) {
	uid := fallbackUserID
	role := fallbackRole

	if md, ok := metadata.FromIncomingContext(ctx); ok && md != nil {
		if v := md.Get("x-user-id"); len(v) > 0 {
			if parsed, err := strconv.ParseInt(v[0], 10, 64); err == nil && parsed > 0 {
				uid = parsed
			}
		}
		if v := md.Get("x-user-role"); len(v) > 0 && v[0] != "" {
			role = v[0]
		}
	}
	return uid, role
}

// resolveTeamIDByProject enforces project-first:
// - projectID must exist
// - project must have team_id (solo is also a team of size 1)
// - if teamID is provided (non-zero), it must match project's team_id
func (s *Service) resolveTeamIDByProject(ctx context.Context, projectID int64, teamID int64) (int64, error) {
	if projectID <= 0 {
		return 0, errors.New("project_id is required")
	}

	rt, err := s.projectClient.GetProjectRuntime(s.internalCtx(ctx), &projectv1.GetProjectRuntimeRequest{
		ProjectId: projectID,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get project runtime: %w", err)
	}
	if rt == nil || rt.TeamId <= 0 {
		// По нашей модели: даже solo = team из 1.
		return 0, errors.New("project has no team_id (team is required, including solo)")
	}

	if teamID == 0 {
		return rt.TeamId, nil
	}
	if teamID != rt.TeamId {
		return 0, fmt.Errorf("team_id mismatch: provided=%d, project.team_id=%d", teamID, rt.TeamId)
	}
	return teamID, nil
}

// performWorkflowAction calls project_service.PerformAction with proper metadata.
// action_name MUST equal workflow Transition.event_name [[11]].
func (s *Service) performWorkflowAction(ctx context.Context, actorID int64, actorRole string, projectID int64, actionName string, payload map[string]interface{}) error {
	pbPayload, _ := structpb.NewStruct(payload)

	actCtx := metadata.AppendToOutgoingContext(
		ctx,
		"x-user-id", fmt.Sprintf("%d", actorID),
		"x-user-role", actorRole,
	)

	_, err := s.projectClient.PerformAction(actCtx, &projectv1.PerformActionRequest{
		ProjectId:  projectID,
		ActionName: actionName,
		Payload:    pbPayload,
	})
	return err
}

// ==================== Topic Registration ====================

type SubmitTopicRegistrationRequest struct {
	ProjectID        int64
	TeamID           int64 // optional (0 allowed)
	ProposedTopic    string
	TopicDescription string
	SupervisorID     int64
	SubmittedBy      int64
}

func (s *Service) SubmitTopicRegistration(ctx context.Context, req *SubmitTopicRegistrationRequest) (*TopicRegistration, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.ProjectID <= 0 {
		return nil, errors.New("project_id is required")
	}
	if req.SubmittedBy <= 0 {
		return nil, errors.New("submitted_by is required")
	}
	if req.SupervisorID <= 0 {
		return nil, errors.New("supervisor_id is required")
	}
	if req.ProposedTopic == "" {
		return nil, errors.New("proposed_topic is required")
	}

	teamID, err := s.resolveTeamIDByProject(ctx, req.ProjectID, req.TeamID)
	if err != nil {
		return nil, err
	}

	// Ensure no active reg for team
	existing, err := s.repo.GetTopicRegistrationByTeam(ctx, teamID)
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
		TeamID:           teamID,
		ProjectID:        req.ProjectID,
		ProposedTopic:    req.ProposedTopic,
		TopicDescription: req.TopicDescription,
		SupervisorID:     req.SupervisorID,
		SubmittedBy:      req.SubmittedBy,
		Status:           StatusPending,
	}

	if err := s.repo.CreateTopicRegistration(ctx, reg); err != nil {
		return nil, fmt.Errorf("не удалось создать заявление: %w", err)
	}

	// Best-effort: lock team composition
	_, _ = s.teamClient.LockTeamComposition(s.internalCtx(ctx), &teamv1.LockTeamCompositionRequest{
		TeamId: teamID,
		Reason: "topic_registration_submitted",
	})

	_ = s.repo.CreateTopicRegistrationReview(ctx, &TopicRegistrationReview{
		RegistrationID: reg.ID,
		ReviewerID:     req.SubmittedBy,
		Action:         "submitted",
		Comment:        "Заявление подано",
	})

	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: ActivityTypeTopicRegistration,
		Description:  fmt.Sprintf("Заявление на тему '%s' подано (project=%d, team=%d)", req.ProposedTopic, req.ProjectID, teamID),
		ActorID:      req.SubmittedBy,
		TargetID:     teamID,
		TargetType:   "team",
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

// STRICT: approve/reject обязаны двигать workflow, иначе не фиксируем решение в admin-проекции.
func (s *Service) ReviewTopicRegistration(ctx context.Context, req *ReviewTopicRegistrationRequest) (*TopicRegistration, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}

	reg, err := s.repo.GetTopicRegistration(ctx, req.RegistrationID)
	if err != nil {
		return nil, fmt.Errorf("заявление не найдено: %w", err)
	}
	if reg.ProjectID <= 0 {
		return nil, errors.New("topic registration has no project_id (data inconsistent)")
	}

	if reg.Status != StatusPending && reg.Status != StatusRevisionRequested {
		return nil, errors.New("заявление не может быть рассмотрено в текущем статусе")
	}

	actorID, actorRole := s.callerFromContext(ctx, req.ReviewerID, "commission")

	// 1) workflow ядро
	switch req.Action {
	case "approve":
		if wfErr := s.performWorkflowAction(ctx, actorID, actorRole, reg.ProjectID, "TOPIC_APPROVED", map[string]interface{}{
			"source":          "admin_service",
			"registration_id": reg.ID,
			"reviewer_id":     req.ReviewerID,
			"comment":         req.Comment,
		}); wfErr != nil {
			return nil, fmt.Errorf("workflow transition TOPIC_APPROVED failed: %w", wfErr)
		}
	case "reject":
		if wfErr := s.performWorkflowAction(ctx, actorID, actorRole, reg.ProjectID, "TOPIC_REJECTED", map[string]interface{}{
			"source":           "admin_service",
			"registration_id":  reg.ID,
			"reviewer_id":      req.ReviewerID,
			"rejection_reason": req.RejectionReason,
			"comment":          req.Comment,
		}); wfErr != nil {
			return nil, fmt.Errorf("workflow transition TOPIC_REJECTED failed: %w", wfErr)
		}
	case "request_changes":
		// workflow не двигаем
	default:
		return nil, errors.New("недопустимое действие")
	}

	// 2) admin projection
	now := time.Now()
	reg.ReviewerID = &req.ReviewerID
	reg.ReviewedAt = &now
	reg.Comment = req.Comment

	switch req.Action {
	case "approve":
		reg.Status = StatusApproved
	case "reject":
		reg.Status = StatusRejected
		reg.RejectionReason = req.RejectionReason
	case "request_changes":
		reg.Status = StatusRevisionRequested
	}

	if err := s.repo.UpdateTopicRegistration(ctx, reg); err != nil {
		return nil, fmt.Errorf("не удалось обновить заявление: %w", err)
	}

	_ = s.repo.CreateTopicRegistrationReview(ctx, &TopicRegistrationReview{
		RegistrationID: reg.ID,
		ReviewerID:     req.ReviewerID,
		Action:         req.Action,
		Comment:        req.Comment,
	})

	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: ActivityTypeTopicApproval,
		Description:  fmt.Sprintf("Topic registration %s: %s", reg.ID, req.Action),
		ActorID:      req.ReviewerID,
		TargetID:     reg.TeamID,
		TargetType:   "topic_registration",
	})

	return reg, nil
}

func (s *Service) ListTopicRegistrations(ctx context.Context, filter TopicRegistrationFilter) ([]*TopicRegistration, int64, error) {
	return s.repo.ListTopicRegistrations(ctx, filter)
}

func (s *Service) GetTopicRegistration(ctx context.Context, id string) (*TopicRegistration, []*TopicRegistrationReview, error) {
	reg, err := s.repo.GetTopicRegistration(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	reviews, _ := s.repo.GetTopicRegistrationReviews(ctx, id)
	return reg, reviews, nil
}

// ==================== Dashboard ====================

type DashboardResponse struct {
	Stats        *DashboardStatsData
	StepProgress []*StepProgressData
	Activities   []*AdminActivity
}

func (s *Service) GetDashboard(ctx context.Context, departmentID int64) (*DashboardResponse, error) {
	stats, err := s.repo.GetDashboardStats(ctx, departmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard stats: %w", err)
	}

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

// ==================== Students ====================

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

func (s *Service) ListStudents(ctx context.Context, req *ListStudentsRequest) ([]*StudentData, int64, error) {
	resp, err := s.authClient.ListUsers(ctx, &authv1.ListUsersRequest{
		UniversityId: req.UniversityID,
		DepartmentId: req.DepartmentID,
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

		teamResp, err := s.teamClient.GetMyTeam(ctx, &teamv1.GetMyTeamRequest{UserId: u.Id})
		if err == nil && teamResp.HasTeam {
			student.TeamID = teamResp.Team.TeamId
			student.TeamName = teamResp.Team.Name
		}

		students = append(students, student)
	}

	return students, resp.TotalCount, nil
}

// ==================== Teams ====================

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
		teams = append(teams, &TeamData{
			ID:          t.Id,
			Name:        t.Name,
			MemberCount: int32(len(t.Members)),
			Status:      "active",
		})
	}

	return teams, resp.TotalCount, nil
}

// ==================== Supervisors ====================

type SupervisorData struct {
	ID         int64
	FullName   string
	Email      string
	Position   string
	TeamsCount int32
	MaxTeams   int32
}

func (s *Service) ListSupervisors(ctx context.Context, departmentID, universityID int64, page, pageSize int32) ([]*SupervisorData, int64, error) {
	resp, err := s.authClient.ListUsers(ctx, &authv1.ListUsersRequest{
		UniversityId: universityID,
		DepartmentId: departmentID,
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

	review := &SubmissionReview{
		SubmissionID: sub.ID,
		ReviewerID:   req.ReviewerID,
		Action:       req.Action,
		Comment:      req.Comment,
		Grade:        &req.Grade,
	}
	_ = s.repo.CreateSubmissionReview(ctx, review)

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

// ==================== Grading ====================

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

func (s *Service) SetStepGrade(ctx context.Context, req *SetGradeRequest) (*Grade, error) {
	existing, err := s.repo.GetGrade(ctx, req.ProjectID, req.StepID)
	if err == nil && existing != nil {
		oldGrade := existing.Grade
		existing.Grade = req.Grade
		existing.Comment = req.Comment
		existing.GradedBy = req.GraderID

		if err := s.repo.UpdateGrade(ctx, existing); err != nil {
			return nil, fmt.Errorf("failed to update grade: %w", err)
		}

		_ = s.repo.CreateGradeHistory(ctx, &GradeHistory{
			GradeID:   existing.ID,
			ProjectID: req.ProjectID,
			StepID:    req.StepID,
			OldGrade:  oldGrade,
			NewGrade:  req.Grade,
			ChangedBy: req.GraderID,
			Reason:    req.Comment,
		})

		return existing, nil
	}

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

	_ = s.repo.CreateGradeHistory(ctx, &GradeHistory{
		GradeID:   grade.ID,
		ProjectID: req.ProjectID,
		StepID:    req.StepID,
		OldGrade:  0,
		NewGrade:  req.Grade,
		ChangedBy: req.GraderID,
		Reason:    "Initial grade",
	})

	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: ActivityTypeGrade,
		Description:  fmt.Sprintf("Grade %d set for project %d, step %d", req.Grade, req.ProjectID, req.StepID),
		ActorID:      req.GraderID,
		TargetID:     req.ProjectID,
		TargetType:   "project",
	})

	return grade, nil
}

// ==================== Workflow Progress ====================

func (s *Service) GetWorkflowProgress(ctx context.Context, departmentID, workflowID int64) ([]*StepProgressData, error) {
	if workflowID == 0 {
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

// ==================== Supervisor Request (project-first) ====================

type CreateSupervisorRequestInput struct {
	ProjectID     int64
	TeamID        int64 // optional (0 allowed)
	SupervisorID  int64
	RequestedBy   int64
	Message       string
	ProposedTopic string
}

func (s *Service) CreateSupervisorRequest(ctx context.Context, req *CreateSupervisorRequestInput) (*SupervisorRequestWithDetails, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.ProjectID <= 0 {
		return nil, errors.New("project_id is required")
	}
	if req.SupervisorID <= 0 {
		return nil, errors.New("supervisor_id is required")
	}
	if req.RequestedBy <= 0 {
		return nil, errors.New("requested_by is required")
	}

	teamID, err := s.resolveTeamIDByProject(ctx, req.ProjectID, req.TeamID)
	if err != nil {
		return nil, err
	}

	hasPending, err := s.repo.HasPendingSupervisorRequest(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to check pending requests: %w", err)
	}
	if hasPending {
		return nil, errors.New("у команды уже есть активный запрос к супервайзеру")
	}

	hasApproved, err := s.repo.HasApprovedSupervisor(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to check approved supervisor: %w", err)
	}
	if hasApproved {
		return nil, errors.New("у команды уже есть утверждённый научный руководитель")
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	supervisorRequest := &SupervisorRequest{
		ID:            uuid.New().String(),
		TeamID:        teamID,
		ProjectID:     req.ProjectID,
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

	_ = s.repo.CreateSupervisorRequestHistory(ctx, &SupervisorRequestHistory{
		RequestID: supervisorRequest.ID,
		Action:    SupervisorRequestActionCreated,
		ActorID:   req.RequestedBy,
		Comment:   "Запрос создан",
	})

	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: ActivityTypeSupervisorRequest,
		Description:  fmt.Sprintf("Team %d (project %d) sent request to supervisor %d", teamID, req.ProjectID, req.SupervisorID),
		ActorID:      req.RequestedBy,
		TargetID:     req.SupervisorID,
		TargetType:   "supervisor_request",
	})

	return s.repo.GetSupervisorRequestWithDetails(ctx, supervisorRequest.ID)
}

func (s *Service) GetSupervisorRequest(ctx context.Context, requestID string) (*SupervisorRequestWithDetails, []*SupervisorRequestHistory, error) {
	req, err := s.repo.GetSupervisorRequestWithDetails(ctx, requestID)
	if err != nil {
		return nil, nil, fmt.Errorf("supervisor request not found: %w", err)
	}
	history, _ := s.repo.GetSupervisorRequestHistory(ctx, requestID)
	return req, history, nil
}

func (s *Service) ListSupervisorRequests(ctx context.Context, filter SupervisorRequestFilter) ([]*SupervisorRequestWithDetails, int64, error) {
	return s.repo.ListSupervisorRequests(ctx, filter)
}

type MySupervisorRequestsResponse struct {
	Requests     []*SupervisorRequestWithDetails
	TotalCount   int64
	PendingCount int32
}

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

type RespondToSupervisorRequestInput struct {
	RequestID    string
	SupervisorID int64
	Action       string // approve, reject
	RejectReason string
	Comment      string
}

// STRICT approve: сначала workflow, потом admin projection + assignment.
// Контракт PerformAction: action_name = Transition.event_name [[11]].
func (s *Service) RespondToSupervisorRequest(ctx context.Context, req *RespondToSupervisorRequestInput) (*SupervisorRequestWithDetails, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}

	supervisorReq, err := s.repo.GetSupervisorRequest(ctx, req.RequestID)
	if err != nil {
		return nil, fmt.Errorf("supervisor request not found: %w", err)
	}

	if supervisorReq.SupervisorID != req.SupervisorID {
		return nil, errors.New("вы не можете ответить на этот запрос")
	}
	if supervisorReq.Status != SupervisorRequestStatusPending {
		return nil, errors.New("запрос уже обработан")
	}
	if supervisorReq.ProjectID <= 0 {
		return nil, errors.New("request has no project_id (data inconsistent)")
	}

	now := time.Now()
	supervisorReq.RespondedAt = &now

	switch req.Action {
	case "approve":
		actorID, actorRole := s.callerFromContext(ctx, req.SupervisorID, "teacher")

		// 1) workflow ядро
		if wfErr := s.performWorkflowAction(ctx, actorID, actorRole, supervisorReq.ProjectID, "SUPERVISOR_SELECTED", map[string]interface{}{
			"source":        "admin_service",
			"request_id":    supervisorReq.ID,
			"supervisor_id": supervisorReq.SupervisorID,
			"comment":       req.Comment,
		}); wfErr != nil {
			return nil, fmt.Errorf("workflow transition SUPERVISOR_SELECTED failed: %w", wfErr)
		}

		// 2) assignment (projection)
		if err := s.AssignSupervisor(ctx, supervisorReq.TeamID, supervisorReq.SupervisorID, supervisorReq.SupervisorID); err != nil {
			return nil, fmt.Errorf("failed to assign supervisor: %w", err)
		}

		supervisorReq.Status = SupervisorRequestStatusApproved

	case "reject":
		supervisorReq.Status = SupervisorRequestStatusRejected
		supervisorReq.RejectReason = req.RejectReason

	default:
		return nil, errors.New("недопустимое действие")
	}

	if err := s.repo.UpdateSupervisorRequest(ctx, supervisorReq); err != nil {
		return nil, fmt.Errorf("failed to update supervisor request: %w", err)
	}

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

	activityType := ActivityTypeSupervisorRequestApproved
	if req.Action == "reject" {
		activityType = ActivityTypeSupervisorRequestRejected
	}

	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: activityType,
		Description:  fmt.Sprintf("Supervisor %d %s request %s", req.SupervisorID, req.Action, req.RequestID),
		ActorID:      req.SupervisorID,
		TargetID:     supervisorReq.TeamID,
		TargetType:   "supervisor_request",
	})

	return s.repo.GetSupervisorRequestWithDetails(ctx, supervisorReq.ID)
}

func (s *Service) CancelSupervisorRequest(ctx context.Context, requestID string, cancelledBy int64, reason string) error {
	supervisorReq, err := s.repo.GetSupervisorRequest(ctx, requestID)
	if err != nil {
		return fmt.Errorf("supervisor request not found: %w", err)
	}

	if supervisorReq.Status != SupervisorRequestStatusPending {
		return errors.New("можно отменить только ожидающий запрос")
	}

	now := time.Now()
	supervisorReq.Status = SupervisorRequestStatusCancelled
	supervisorReq.RespondedAt = &now

	if err := s.repo.UpdateSupervisorRequest(ctx, supervisorReq); err != nil {
		return fmt.Errorf("failed to cancel supervisor request: %w", err)
	}

	comment := "Запрос отменён"
	if reason != "" {
		comment = reason
	}

	_ = s.repo.CreateSupervisorRequestHistory(ctx, &SupervisorRequestHistory{
		RequestID: supervisorReq.ID,
		Action:    SupervisorRequestActionCancelled,
		ActorID:   cancelledBy,
		Comment:   comment,
	})

	return nil
}

// ==================== Details used by handlers ====================

type StudentDetailResponse struct {
	Student     *StudentFullInfo
	Grades      []*Grade
	Submissions []*Submission
}

func (s *Service) GetStudentFullInfo(ctx context.Context, studentID int64) (*StudentDetailResponse, error) {
	student, err := s.repo.GetStudentByID(ctx, studentID)
	if err != nil {
		return nil, fmt.Errorf("student not found: %w", err)
	}

	grades, _ := s.repo.GetStudentGrades(ctx, studentID)
	submissions, _ := s.repo.GetStudentSubmissions(ctx, studentID)

	return &StudentDetailResponse{
		Student:     student,
		Grades:      grades,
		Submissions: submissions,
	}, nil
}

type TeamDetailResponse struct {
	Team              *TeamFullDetails
	Supervisor        *SupervisorBasicInfo
	Grades            []*Grade
	Submissions       []*Submission
	TopicRegistration *TopicRegistration
}

func (s *Service) GetTeamFullDetails(ctx context.Context, teamID int64) (*TeamDetailResponse, error) {
	team, err := s.repo.GetTeamFullDetails(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("team not found: %w", err)
	}

	response := &TeamDetailResponse{Team: team}

	if team.ProjectID > 0 {
		grades, _, _ := s.GetProjectGrades(ctx, team.ProjectID)
		response.Grades = grades

		submissions, _, _ := s.repo.ListSubmissions(ctx, SubmissionFilter{
			TeamID: teamID,
			Limit:  50,
			Offset: 0,
			Status: "",
			StepID: 0,
		})
		response.Submissions = submissions
	}

	topicReg, _ := s.repo.GetTopicRegistrationByTeam(ctx, teamID)
	response.TopicRegistration = topicReg

	return response, nil
}

type UpdateTeamByAdminRequest struct {
	TeamID       int64
	Name         string
	SupervisorID int64
	MemberIDs    []int64
	UpdatedBy    int64
}

func (s *Service) UpdateTeamByAdmin(ctx context.Context, req *UpdateTeamByAdminRequest) (*TeamFullDetails, error) {
	_, err := s.repo.GetTeamFullDetails(ctx, req.TeamID)
	if err != nil {
		return nil, fmt.Errorf("team not found: %w", err)
	}

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

	if err := s.repo.UpdateTeamByAdmin(ctx, req.TeamID, updates); err != nil {
		return nil, fmt.Errorf("failed to update team: %w", err)
	}

	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: ActivityTypeTeamUpdate,
		Description:  fmt.Sprintf("Team %d updated by admin", req.TeamID),
		ActorID:      req.UpdatedBy,
		TargetID:     req.TeamID,
		TargetType:   "team",
	})

	return s.repo.GetTeamFullDetails(ctx, req.TeamID)
}

func (s *Service) DeleteTeamByAdmin(ctx context.Context, teamID int64, reason string, deletedBy int64) error {
	_, err := s.repo.GetTeamFullDetails(ctx, teamID)
	if err != nil {
		return fmt.Errorf("team not found: %w", err)
	}
	return s.repo.DeleteTeamByAdmin(ctx, teamID, reason, deletedBy)
}

func (s *Service) GetGradingHistoryFull(ctx context.Context, projectID, stepID int64) ([]*GradeHistoryFull, error) {
	return s.repo.GetGradingHistoryFull(ctx, projectID, stepID)
}
