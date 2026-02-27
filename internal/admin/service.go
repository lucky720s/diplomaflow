package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Service struct {
	repo               Repository
	authClient         authv1.AuthServiceClient
	projectClient      projectv1.ProjectServiceClient
	teamClient         teamv1.TeamServiceClient
	workflowClient     workflowv1.WorkflowServiceClient
	notificationClient notificationv1.NotificationServiceClient
	logger             *zap.Logger
}
type SubmitDocumentRequest struct {
	ProjectID   int64
	SubmittedBy int64
	FileIDs     []string
	Comment     string
}

func NewService(
	repo Repository,
	authClient authv1.AuthServiceClient,
	projectClient projectv1.ProjectServiceClient,
	teamClient teamv1.TeamServiceClient,
	workflowClient workflowv1.WorkflowServiceClient,
	notificationClient notificationv1.NotificationServiceClient,
	logger *zap.Logger,
) *Service {
	return &Service{
		repo:               repo,
		authClient:         authClient,
		projectClient:      projectClient,
		teamClient:         teamClient,
		workflowClient:     workflowClient,
		notificationClient: notificationClient,
		logger:             logger,
	}
}

// ==================== Helpers ====================

func (s *Service) internalCtx(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "x-internal-service", "admin_service")
}

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
func (s *Service) performWorkflowAction(ctx context.Context, actorID int64, actorRole string, projectID int64, actionName string, payload map[string]interface{}) error {
	pbPayload, _ := structpb.NewStruct(payload)

	actCtx := metadata.AppendToOutgoingContext(
		ctx,
		"x-internal-service", "admin_service",
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

// tryPerformIfAvailable checks current state transitions and performs event if available+can_execute.
func (s *Service) tryPerformIfAvailable(ctx context.Context, actorID int64, actorRole string, projectID int64, eventName string, payload map[string]interface{}) error {
	rt, err := s.projectClient.GetProjectRuntime(s.internalCtx(ctx), &projectv1.GetProjectRuntimeRequest{
		ProjectId: projectID,
	})
	if err != nil {
		return fmt.Errorf("GetProjectRuntime failed: %w", err)
	}
	if rt == nil {
		return errors.New("project runtime is nil")
	}

	outCtx := metadata.AppendToOutgoingContext(
		ctx,
		"x-internal-service", "admin_service",
		"x-user-id", fmt.Sprintf("%d", actorID),
		"x-user-role", actorRole,
	)

	tr, err := s.workflowClient.GetAvailableTransitions(outCtx, &workflowv1.GetAvailableTransitionsRequest{
		ProjectId:      projectID,
		CurrentStateId: rt.CurrentStateId,
		UserId:         actorID,
		UserRole:       actorRole,
	})

	if err != nil {
		// Если runtime API недоступно/ошибка — fallback: просто пробуем PerformAction.
		s.logger.Warn("GetAvailableTransitions failed, fallback to PerformAction",
			zap.Error(err),
			zap.Int64("project_id", projectID),
			zap.String("event", eventName),
		)
		return s.performWorkflowAction(ctx, actorID, actorRole, projectID, eventName, payload)
	}

	available := false
	can := false
	for _, at := range tr.Transitions {
		if at == nil || at.Transition == nil {
			continue
		}
		if at.Transition.EventName == eventName {
			available = true
			can = at.CanExecute
			if !can {
				return fmt.Errorf("transition %s blocked: %s", eventName, at.BlockedReason)
			}
		}
	}
	if !available {
		// silently skip (workflow may not have this transition)
		return nil
	}
	return s.performWorkflowAction(ctx, actorID, actorRole, projectID, eventName, payload)
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

	existing, err := s.repo.GetTopicRegistrationByTeam(ctx, teamID)
	if err == nil && existing != nil {
		switch existing.Status {
		case StatusPending:
			return nil, status.Error(codes.AlreadyExists,
				"у команды уже есть заявление на рассмотрении")
		case StatusApproved:
			return nil, status.Error(codes.AlreadyExists,
				"тема уже утверждена для этой команды")
		case StatusRejected:
			// Rejected → можно подать заново. Не блокируем.
		default:
			// Неизвестный статус — пропускаем (fail-open)
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

	// =====================================================================
	// FIX: двигаем workflow TOPIC_SUBMISSION → TOPIC_APPROVAL
	// Без этого проект остаётся на TOPIC_SUBMISSION, и при approve
	// переход TOPIC_APPROVED (доступный только из TOPIC_APPROVAL) не найдётся.
	// Используем tryPerformIfAvailable — тот же паттерн, что и в
	// createProjectForTeamOnApprove для TEAM_FORMED / SUPERVISOR_SELECTED.
	// =====================================================================
	actorID, actorRole := s.callerFromContext(ctx, req.SubmittedBy, "student")
	if wfErr := s.tryPerformIfAvailable(ctx, actorID, actorRole, req.ProjectID, "TOPIC_SUBMITTED", map[string]interface{}{
		"source":          "admin_service",
		"registration_id": reg.ID,
		"proposed_topic":  req.ProposedTopic,
	}); wfErr != nil {
		s.logger.Warn("Failed to perform TOPIC_SUBMITTED transition (non-blocking)",
			zap.Error(wfErr),
			zap.Int64("project_id", req.ProjectID),
		)
	}
	// notify supervisor (если это его тема)
	s.notifyBestEffort(ctx, reg.SupervisorID,
		"Новая тема на согласование",
		fmt.Sprintf("Команда «%d» подала тему: %s", teamID, reg.ProposedTopic),
		fmt.Sprintf("/teacher/diploms/%d", reg.ProjectID),
		"topic_registration_submitted",
	)

	// notify team
	s.notifyTeamBestEffort(ctx, teamID,
		"Тема отправлена на согласование",
		fmt.Sprintf("Тема: %s", reg.ProposedTopic),
		"/diplom",
		"topic_registration_submitted_team",
	)

	return reg, nil
}

type ReviewTopicRegistrationRequest struct {
	RegistrationID  string
	ReviewerID      int64
	Action          string // approve, reject, request_changes
	Comment         string
	RejectionReason string
}

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
		return nil, status.Error(codes.FailedPrecondition,
			"заявление не может быть рассмотрено в текущем статусе")
	}

	actorID, actorRole := s.callerFromContext(ctx, req.ReviewerID, "commission")

	switch req.Action {
	case "approve":
		_ = s.tryPerformIfAvailable(ctx, actorID, actorRole, reg.ProjectID, "TOPIC_SUBMITTED", map[string]interface{}{
			"source":          "admin_service",
			"registration_id": reg.ID,
			"auto":            true,
		})
		_ = s.tryPerformIfAvailable(ctx, actorID, actorRole, reg.ProjectID, "TOPIC_APPROVED", map[string]interface{}{
			"source":          "admin_service",
			"registration_id": reg.ID,
			"reviewer_id":     req.ReviewerID,
			"comment":         req.Comment,
		})
		reg.Status = StatusApproved
		if err := s.repo.SetProjectTopicRegisteredAt(ctx, reg.ProjectID, time.Now().UTC()); err != nil {
			s.logger.Warn("Failed to set topic_registered_at",
				zap.Error(err),
				zap.Int64("project_id", reg.ProjectID),
			)
		}

	case "reject":
		_ = s.tryPerformIfAvailable(ctx, actorID, actorRole, reg.ProjectID, "TOPIC_SUBMITTED", map[string]interface{}{
			"source":          "admin_service",
			"registration_id": reg.ID,
			"auto":            true,
		})
		_ = s.tryPerformIfAvailable(ctx, actorID, actorRole, reg.ProjectID, "TOPIC_REJECTED", map[string]interface{}{
			"source":           "admin_service",
			"registration_id":  reg.ID,
			"reviewer_id":      req.ReviewerID,
			"rejection_reason": req.RejectionReason,
			"comment":          req.Comment,
		})
		reg.Status = StatusRejected
		reg.RejectionReason = req.RejectionReason

	case "request_changes":
		reg.Status = StatusRevisionRequested

	default:
		return nil, errors.New("недопустимое действие")
	}

	now := time.Now()
	reg.ReviewerID = &req.ReviewerID
	reg.ReviewedAt = &now
	reg.Comment = req.Comment

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
	switch req.Action {
	case "approve":
		s.notifyTeamBestEffort(ctx, reg.TeamID,
			"Тема утверждена",
			fmt.Sprintf("Тема: %s", reg.ProposedTopic),
			"/diplom",
			"topic_registration_approved",
		)
	case "reject":
		reason := req.RejectionReason
		if reason == "" {
			reason = "Причина не указана"
		}
		s.notifyTeamBestEffort(ctx, reg.TeamID,
			"Тема отклонена",
			fmt.Sprintf("Причина: %s", reason),
			"/diplom",
			"topic_registration_rejected",
		)
	case "request_changes":
		msg := req.Comment
		if msg == "" {
			msg = "Нужны правки"
		}
		s.notifyTeamBestEffort(ctx, reg.TeamID,
			"Нужны правки по теме",
			msg,
			"/diplom",
			"topic_registration_changes_requested",
		)
	}

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

	// Build student list first (no N+1 gRPC calls)
	students := make([]*StudentData, 0, len(resp.Users))
	for _, u := range resp.Users {
		students = append(students, &StudentData{
			ID:        u.Id,
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
		})
	}

	// Best-effort: enrich with team info using ListTeams (single call)
	// For MVP this is acceptable; for production consider a batch endpoint
	if req.DepartmentID > 0 && len(students) > 0 {
		teamsResp, tErr := s.teamClient.ListTeams(s.forwardCtx(ctx), &teamv1.ListTeamsRequest{
			DepartmentId: req.DepartmentID,
			Page:         1,
			PageSize:     500, // reasonable upper bound for a department
		})
		if tErr == nil && teamsResp != nil {
			// Build userID -> team mapping from all teams' members
			userTeamMap := make(map[int64]*TeamData)
			for _, t := range teamsResp.Teams {
				if t == nil {
					continue
				}
				for _, m := range t.Members {
					if m != nil && m.UserId > 0 {
						userTeamMap[m.UserId] = &TeamData{
							ID:   t.Id,
							Name: t.Name,
						}
					}
				}
			}
			// Enrich students
			for _, s := range students {
				if td, ok := userTeamMap[s.ID]; ok {
					s.TeamID = td.ID
					s.TeamName = td.Name
				}
			}
		}
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
	resp, err := s.teamClient.ListTeams(s.forwardCtx(ctx), &teamv1.ListTeamsRequest{
		DepartmentId: req.DepartmentID,
		Page:         req.Page,
		PageSize:     req.PageSize,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, 0, st.Err()
		}
		return nil, 0, status.Errorf(codes.Internal, "failed to list teams: %v", err)
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

	teacherIDs := make([]int64, 0, len(resp.Users))
	for _, u := range resp.Users {
		teacherIDs = append(teacherIDs, u.Id)
	}
	teamCounts := make(map[int64]int32)
	if len(teacherIDs) > 0 {
		batchCounts, err := s.repo.BatchCountTeamsBySupervisors(ctx, teacherIDs)
		if err == nil {
			for id, cnt := range batchCounts {
				teamCounts[id] = int32(cnt)
			}
		}
	}

	var supervisors []*SupervisorData
	for _, u := range resp.Users {
		supervisors = append(supervisors, &SupervisorData{
			ID:         u.Id,
			FullName:   u.FirstName + " " + u.LastName,
			Email:      u.Email,
			Position:   "Teacher",
			TeamsCount: teamCounts[u.Id],
			MaxTeams:   5,
		})
	}

	return supervisors, resp.TotalCount, nil
}

func (s *Service) AssignSupervisor(ctx context.Context, teamID, supervisorID, assignedBy int64, silent bool) error {
	updated := false //nolint:govet,govet,govet
	prevSupervisorID := int64(0)

	existing, err := s.repo.GetSupervisorAssignment(ctx, teamID)
	if err == nil && existing != nil {
		prevSupervisorID = existing.SupervisorID
		existing.SupervisorID = supervisorID
		existing.AssignedBy = assignedBy
		if err = s.repo.UpdateSupervisorAssignment(ctx, existing); err != nil { //nolint:govet,govet,govet
			return fmt.Errorf("failed to update supervisor assignment: %w", err)
		}
		updated = true
	} else {
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to get supervisor assignment: %w", err)
		}

		assignment := &SupervisorAssignment{
			TeamID:       teamID,
			SupervisorID: supervisorID,
			AssignedBy:   assignedBy,
		}
		if err := s.repo.AssignSupervisor(ctx, assignment); err != nil {
			return fmt.Errorf("failed to assign supervisor: %w", err)
		}
	}

	desc := fmt.Sprintf("Supervisor %d assigned to team %d", supervisorID, teamID)
	if updated && prevSupervisorID > 0 && prevSupervisorID != supervisorID {
		desc = fmt.Sprintf("Supervisor %d reassigned to team %d (from %d)", supervisorID, teamID, prevSupervisorID)
	}
	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: ActivityTypeSupervisorAssign,
		Description:  desc,
		ActorID:      assignedBy,
		TargetID:     teamID,
		TargetType:   "team",
	})

	if !silent {
		projectID := int64(0)
		if td, derr := s.repo.GetTeamFullDetails(ctx, teamID); derr == nil && td != nil {
			projectID = td.ProjectID
		}

		teacherLink := fmt.Sprintf("/teacher/boards/%d", teamID)
		if projectID > 0 {
			teacherLink = fmt.Sprintf("/teacher/diploms/%d", projectID)
		}

		s.notifyTeamBestEffort(ctx, teamID,
			"Назначен научный руководитель",
			"В вашей команде назначен научный руководитель. Можно продолжать работу над дипломом.",
			"/diplom",
			"supervisor_assigned_admin",
		)

		s.notifyBestEffort(ctx, supervisorID,
			"Вам назначена команда",
			fmt.Sprintf("Вам назначена команда %d для руководства.", teamID),
			teacherLink,
			"supervisor_assigned_to_team",
		)
	}

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
			Silent:    true,
		})
	}
	switch req.Action {
	case "approve":
		msg := "Документ принят."
		if req.Grade > 0 {
			msg = fmt.Sprintf("Документ принят. Оценка: %d", req.Grade)
		}
		s.notifyTeamBestEffort(ctx, sub.TeamID,
			"Проверка документа: принято",
			msg,
			"/diplom",
			"submission_review_approved",
		)
	case "reject":
		s.notifyTeamBestEffort(ctx, sub.TeamID,
			"Проверка документа: отклонено",
			req.Comment,
			"/diplom",
			"submission_review_rejected",
		)
	case "request_changes":
		s.notifyTeamBestEffort(ctx, sub.TeamID,
			"Проверка документа: нужны правки",
			req.Comment,
			"/diplom",
			"submission_review_changes_requested",
		)
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
	Silent    bool
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
		if !req.Silent {
			teamID := int64(0)
			if rt, rtErr := s.projectClient.GetProjectRuntime(s.internalCtx(ctx), &projectv1.GetProjectRuntimeRequest{ProjectId: req.ProjectID}); rtErr == nil && rt != nil {
				teamID = rt.TeamId
			}
			if teamID > 0 {
				s.notifyTeamBestEffort(ctx, teamID,
					"Выставлена оценка за этап",
					fmt.Sprintf("Оценка: %d. %s", req.Grade, req.Comment),
					"/diplom",
					"step_grade_set",
				)
			}
		}

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
	if !req.Silent {
		teamID := int64(0)
		if rt, rtErr := s.projectClient.GetProjectRuntime(s.internalCtx(ctx), &projectv1.GetProjectRuntimeRequest{ProjectId: req.ProjectID}); rtErr == nil && rt != nil {
			teamID = rt.TeamId
		}
		if teamID > 0 {
			s.notifyTeamBestEffort(ctx, teamID,
				"Выставлена оценка за этап",
				fmt.Sprintf("Оценка: %d. %s", req.Grade, req.Comment),
				"/diplom",
				"step_grade_set",
			)
		}
	}

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

// ==================== Supervisor Request (supports project-first AND team-first) ====================

type CreateSupervisorRequestInput struct {
	ProjectID     int64 // may be 0 for team-first
	TeamID        int64 // required for team-first, optional for project-first
	SupervisorID  int64
	RequestedBy   int64
	Message       string
	ProposedTopic string
}

func (s *Service) CreateSupervisorRequest(ctx context.Context, req *CreateSupervisorRequestInput) (*SupervisorRequestWithDetails, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.SupervisorID <= 0 {
		return nil, errors.New("supervisor_id is required")
	}
	if req.RequestedBy <= 0 {
		return nil, errors.New("requested_by is required")
	}

	// project-first OR team-first
	teamID := int64(0)
	var projectIDPtr *int64

	if req.ProjectID > 0 {
		tid, err := s.resolveTeamIDByProject(ctx, req.ProjectID, req.TeamID)
		if err != nil {
			return nil, err
		}
		teamID = tid
		pid := req.ProjectID
		projectIDPtr = &pid
	} else {
		// team-first: no project yet
		if req.TeamID <= 0 {
			return nil, errors.New("team_id is required when project_id is not provided")
		}
		teamID = req.TeamID
		projectIDPtr = nil
	}
	actorID, actorRole := s.callerFromContext(ctx, req.RequestedBy, "")
	isTeacherSelfClaim := (actorRole == "teacher" || actorRole == "admin") &&
		actorID > 0 &&
		actorID == req.RequestedBy &&
		req.RequestedBy == req.SupervisorID

	if !isTeacherSelfClaim {
		if err := s.ensureUserInTeam(ctx, teamID, req.RequestedBy); err != nil {
			return nil, err
		}
	}

	hasPending, err := s.repo.HasPendingSupervisorRequest(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to check pending requests: %w", err)
	}
	if hasPending {
		return nil, status.Error(codes.AlreadyExists,
			"у команды уже есть активный запрос к супервайзеру")
	}

	hasApproved, err := s.repo.HasApprovedSupervisor(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to check approved supervisor: %w", err)
	}
	if hasApproved {
		return nil, status.Error(codes.AlreadyExists,
			"у команды уже есть утверждённый научный руководитель")
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	supervisorRequest := &SupervisorRequest{
		ID:            uuid.New().String(),
		TeamID:        teamID,
		ProjectID:     projectIDPtr, // NULL if team-first
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
		Description:  fmt.Sprintf("Team %d sent request to supervisor %d (project_id=%v)", teamID, req.SupervisorID, req.ProjectID),
		ActorID:      req.RequestedBy,
		TargetID:     req.SupervisorID,
		TargetType:   "supervisor_request",
	})

	details, derr := s.repo.GetSupervisorRequestWithDetails(ctx, supervisorRequest.ID)
	if derr != nil {
		return nil, derr
	}

	s.notifyBestEffort(ctx, req.SupervisorID,
		"Новая заявка на руководство",
		fmt.Sprintf("Команда «%s» отправила заявку. Предложенная тема: %s", details.TeamName, details.ProposedTopic),
		"/teacher/requests",
		"supervisor_request_created",
	)
	// notify team (confirmation)
	s.notifyTeamBestEffort(ctx, teamID,
		"Заявка научному руководителю отправлена",
		"Ожидайте ответа преподавателя. Вы можете отменить заявку и выбрать другого руководителя.",
		"/diplom/select-supervisor",
		"supervisor_request_sent",
	)

	return details, nil
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
	Requests      []*SupervisorRequestWithDetails
	TotalCount    int64
	PendingCount  int32
	ActiveTeams   int32
	TotalStudents int32
	Teams         []*TeamFullDetails
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

	teams, totalStudents, err := s.repo.GetSupervisorTeamsReport(ctx, supervisorID)
	if err != nil {
		return nil, fmt.Errorf("failed to build supervisor teams report: %w", err)
	}
	return &MySupervisorRequestsResponse{
		Requests:      requests,
		TotalCount:    total,
		PendingCount:  int32(pendingCount),
		ActiveTeams:   int32(len(teams)),
		TotalStudents: totalStudents,
		Teams:         teams,
	}, nil
}

type RespondToSupervisorRequestInput struct {
	RequestID    string
	SupervisorID int64
	Action       string // approve, reject
	RejectReason string
	Comment      string
}

func (s *Service) createProjectForTeamOnApprove(ctx context.Context, teamID int64, supervisorID int64, proposedTopic string) (int64, error) {
	tc, err := s.repo.GetTeamContext(ctx, teamID)
	if err != nil {
		return 0, fmt.Errorf("GetTeamContext failed: %w", err)
	}

	wf, err := s.workflowClient.GetActiveWorkflowByDepartment(ctx, &workflowv1.GetActiveWorkflowByDepartmentRequest{
		DepartmentId: tc.DepartmentID,
	})
	if err != nil {
		return 0, fmt.Errorf("GetActiveWorkflowByDepartment failed: %w", err)
	}
	if wf == nil || wf.Id == 0 {
		return 0, errors.New("no active workflow for department")
	}

	title := fmt.Sprintf("Diploma project (%s)", tc.TeamName)
	desc := proposedTopic
	if desc == "" {
		desc = "Created on supervisor approval"
	}

	// Create project (owner = team leader)
	cr, err := s.projectClient.CreateProject(s.internalCtx(ctx), &projectv1.CreateProjectRequest{
		Title:        title,
		Description:  desc,
		StudentId:    tc.LeaderUserID,
		WorkflowName: wf.Name,
		UniversityId: tc.UniversityID,
		DepartmentId: tc.DepartmentID,
		TeamId:       tc.TeamID,
	})
	if err != nil {
		return 0, fmt.Errorf("CreateProject failed: %w", err)
	}
	if cr == nil || cr.ProjectId == 0 {
		return 0, errors.New("CreateProject returned empty response")
	}

	// Best-effort lock team composition at this point
	_, _ = s.teamClient.LockTeamComposition(s.internalCtx(ctx), &teamv1.LockTeamCompositionRequest{
		TeamId: tc.TeamID,
		Reason: "supervisor_selected",
	})

	// Best-effort: move workflow forward (TEAM_FORMED then SUPERVISOR_SELECTED), if available
	// TEAM_FORMED as leader/student
	if err := s.tryPerformIfAvailable(ctx, tc.LeaderUserID, "student", cr.ProjectId, "TEAM_FORMED", map[string]interface{}{
		"source":  "admin_service",
		"team_id": tc.TeamID,
		"comment": "Auto after supervisor approval",
	}); err != nil {
		s.logger.Error("auto-advance TEAM_FORMED failed",
			zap.Int64("project_id", cr.ProjectId),
			zap.Int64("leader_id", tc.LeaderUserID),
			zap.Error(err),
		)
	} else {
		s.logger.Info("auto-advance TEAM_FORMED OK", zap.Int64("project_id", cr.ProjectId))

		// SUPERVISOR_SELECTED as teacher (только если TEAM_FORMED прошёл)
		if err := s.tryPerformIfAvailable(ctx, supervisorID, "teacher", cr.ProjectId, "SUPERVISOR_SELECTED", map[string]interface{}{
			"source":        "admin_service",
			"team_id":       tc.TeamID,
			"supervisor_id": supervisorID,
			"comment":       "Approved",
		}); err != nil {
			s.logger.Error("auto-advance SUPERVISOR_SELECTED failed",
				zap.Int64("project_id", cr.ProjectId),
				zap.Int64("supervisor_id", supervisorID),
				zap.Error(err),
			)
		} else {
			s.logger.Info("auto-advance SUPERVISOR_SELECTED OK", zap.Int64("project_id", cr.ProjectId))
		}
	}

	return cr.ProjectId, nil
}

func (s *Service) RespondToSupervisorRequest(ctx context.Context, req *RespondToSupervisorRequestInput) (*SupervisorRequestWithDetails, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}

	supervisorReq, err := s.repo.GetSupervisorRequest(ctx, req.RequestID)
	if err != nil {
		return nil, fmt.Errorf("supervisor request not found: %w", err)
	}

	if supervisorReq.SupervisorID != req.SupervisorID {
		return nil, status.Error(codes.PermissionDenied,
			"вы не можете ответить на этот запрос")
	}
	if supervisorReq.Status != SupervisorRequestStatusPending {
		return nil, status.Error(codes.FailedPrecondition,
			"запрос уже обработан")
	}

	now := time.Now()
	supervisorReq.RespondedAt = &now

	switch req.Action {
	case "approve":
		if supervisorReq.ProjectID == nil || *supervisorReq.ProjectID <= 0 {
			if supervisorReq.TeamID <= 0 {
				return nil, errors.New("cannot approve: request has no team_id and no project_id")
			}
			pid, err := s.createProjectForTeamOnApprove(ctx, supervisorReq.TeamID, supervisorReq.SupervisorID, supervisorReq.ProposedTopic)
			if err != nil {
				return nil, err
			}
			supervisorReq.ProjectID = &pid
		}

		// Assignment (projection)
		if supervisorReq.TeamID > 0 {
			if err := s.AssignSupervisor(ctx, supervisorReq.TeamID, supervisorReq.SupervisorID, supervisorReq.SupervisorID, true); err != nil {
				return nil, fmt.Errorf("failed to assign supervisor: %w", err)
			}

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

	details, derr := s.repo.GetSupervisorRequestWithDetails(ctx, supervisorReq.ID)
	if derr != nil {
		return nil, derr
	}

	if req.Action == "approve" {
		// после approve у команды уже создаётся project (team-first) и дальше жизнь в дипломе
		s.notifyTeamBestEffort(ctx, supervisorReq.TeamID,
			"Научный руководитель одобрил заявку",
			fmt.Sprintf("Заявка одобрена. Руководитель: %s. Теперь можно продолжать работу над дипломом.", details.SupervisorName),
			"/diplom",
			"supervisor_request_approved",
		)
	} else if req.Action == "reject" {
		// после reject логично вернуть на страницу команды, чтобы выбрать другого и отправить заявку
		reason := req.RejectReason
		if reason == "" {
			reason = "Причина не указана"
		}
		s.notifyTeamBestEffort(ctx, supervisorReq.TeamID,
			"Заявка на научного руководителя отклонена",
			fmt.Sprintf("Заявка отклонена. Причина: %s", reason),
			"/diplom/select-supervisor",
			"supervisor_request_rejected",
		)
	}

	return details, nil
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
	s.notifyBestEffort(ctx, supervisorReq.SupervisorID,
		"Заявка отменена",
		"Команда отменила заявку на руководство.",
		"/teacher/requests",
		"supervisor_request_cancelled",
	)
	// notify team
	s.notifyTeamBestEffort(ctx, supervisorReq.TeamID,
		"Заявка научному руководителю отменена",
		"Вы можете выбрать другого руководителя и отправить новую заявку.",
		"/diplom/select-supervisor",
		"supervisor_request_cancelled_team",
	)

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
	before, err := s.repo.GetTeamFullDetails(ctx, req.TeamID)
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

	after, gerr := s.repo.GetTeamFullDetails(ctx, req.TeamID)
	if gerr != nil {
		return nil, gerr
	}

	// name changed
	if req.Name != "" && before.Name != after.Name {
		s.notifyTeamBestEffort(ctx, req.TeamID,
			"Изменено название команды",
			fmt.Sprintf("Новое название: %s", after.Name),
			"/team",
			"team_admin_name_updated",
		)
	}

	// supervisor changed
	if req.SupervisorID > 0 && before.SupervisorID != after.SupervisorID {
		projectID := after.ProjectID
		teacherLink := fmt.Sprintf("/teacher/boards/%d", req.TeamID)
		if projectID > 0 {
			teacherLink = fmt.Sprintf("/teacher/diploms/%d", projectID)
		}

		s.notifyTeamBestEffort(ctx, req.TeamID,
			"Изменён научный руководитель",
			"Команде назначен (или изменён) научный руководитель.",
			"/diplom",
			"team_supervisor_changed_admin",
		)

		s.notifyBestEffort(ctx, after.SupervisorID,
			"Вам назначена команда",
			fmt.Sprintf("Вам назначена команда «%s».", after.Name),
			teacherLink,
			"supervisor_assigned_to_team",
		)
	}

	return after, nil
}

func (s *Service) DeleteTeamByAdmin(ctx context.Context, teamID int64, reason string, deletedBy int64) error {
	team, err := s.repo.GetTeamFullDetails(ctx, teamID)
	if err != nil {
		return fmt.Errorf("team not found: %w", err)
	}
	for _, m := range team.Members {
		if m == nil || m.UserID <= 0 {
			continue
		}
		s.notifyBestEffort(ctx, m.UserID,
			"Команда удалена администратором",
			fmt.Sprintf("Команда «%s» была удалена. Причина: %s", team.Name, reason),
			"/team",
			"team_deleted_admin",
		)
	}

	return s.repo.DeleteTeamByAdmin(ctx, teamID, reason, deletedBy)

}

func (s *Service) GetGradingHistoryFull(ctx context.Context, projectID, stepID int64) ([]*GradeHistoryFull, error) {
	return s.repo.GetGradingHistoryFull(ctx, projectID, stepID)
}
func (s *Service) ensureUserInTeam(ctx context.Context, teamID, userID int64) error {
	if teamID <= 0 || userID <= 0 {
		return errors.New("team_id and user_id are required")
	}
	t, err := s.teamClient.GetTeam(s.internalCtx(ctx), &teamv1.GetTeamRequest{TeamId: teamID})
	if err != nil {
		return fmt.Errorf("GetTeam failed: %w", err)
	}
	for _, m := range t.Members {
		if m != nil && m.UserId == userID {
			return nil
		}
	}
	return status.Error(codes.PermissionDenied, "forbidden: user is not a member of the team")
}
func (s *Service) ListAvailableTeams(ctx context.Context, departmentID int64, page, pageSize int32) ([]*AvailableTeamData, int64, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	limit := int(pageSize)
	offset := int((page - 1) * pageSize)

	return s.repo.ListAvailableTeams(ctx, departmentID, limit, offset)
}
func (s *Service) forwardCtx(ctx context.Context) context.Context {
	in, _ := metadata.FromIncomingContext(ctx)
	out := metadata.NewOutgoingContext(ctx, in.Copy())
	return metadata.AppendToOutgoingContext(out, "x-internal-service", "admin_service")
}
func (s *Service) SubmitDocumentForStep(ctx context.Context, req *SubmitDocumentRequest) (*Submission, error) {
	teamID, err := s.resolveTeamIDByProject(ctx, req.ProjectID, 0)
	if err != nil {
		return nil, err
	}

	// Получаем текущее состояние проекта
	rt, err := s.projectClient.GetProjectRuntime(s.internalCtx(ctx),
		&projectv1.GetProjectRuntimeRequest{ProjectId: req.ProjectID})
	if err != nil {
		return nil, err
	}

	// Проверяем что текущее состояние — DOCUMENT_UPLOAD тип
	filesJSON, _ := json.Marshal(req.FileIDs)
	dataJSON, _ := json.Marshal(map[string]interface{}{
		"file_ids": req.FileIDs,
		"comment":  req.Comment,
	})

	sub := &Submission{
		ID:          uuid.New().String(),
		ProjectID:   req.ProjectID,
		TeamID:      teamID,
		StepID:      rt.CurrentStateId,
		SubmittedBy: req.SubmittedBy,
		Status:      StatusPending,
		Data:        datatypes.JSON(dataJSON),
		Files:       datatypes.JSON(filesJSON),
	}

	if err := s.repo.CreateSubmission(ctx, sub); err != nil {
		return nil, err
	}
	s.ensureNormCheckIfNeeded(ctx, sub)

	// Двигаем workflow
	eventName := s.getEventNameForState(rt.CurrentStateName)
	if eventName != "" {
		actorID, actorRole := s.callerFromContext(ctx, req.SubmittedBy, "student")
		_ = s.tryPerformIfAvailable(ctx, actorID, actorRole, req.ProjectID, eventName, map[string]interface{}{
			"file_ids":      req.FileIDs,
			"submission_id": sub.ID,
		})
	}
	// notify team
	s.notifyTeamBestEffort(ctx, teamID,
		"Документ отправлен на проверку",
		"Файл(ы) загружены и отправлены на рассмотрение.",
		"/diplom",
		"submission_submitted",
	)

	// notify supervisor if assigned
	if a, aerr := s.repo.GetSupervisorAssignment(ctx, teamID); aerr == nil && a != nil && a.SupervisorID > 0 {
		s.notifyBestEffort(ctx, a.SupervisorID,
			"Новый документ на проверку",
			fmt.Sprintf("Команда %d отправила документ(ы) на проверку.", teamID),
			fmt.Sprintf("/teacher/diploms/%d", req.ProjectID),
			"submission_submitted_to_reviewer",
		)
	}

	return sub, nil
}

func (s *Service) getEventNameForState(stateName string) string {
	mapping := map[string]string{
		"PRE_DEFENSE_1": "PREDEFENSE1_SUBMITTED",
		"PRE_DEFENSE_2": "PREDEFENSE2_SUBMITTED",
		"NORM_CONTROL":  "",
		"ECONOMICS":     "ECONOMICS_SUBMITTED",
		"ANTIPLAGIAT":   "ANTIPLAGIAT_SUBMITTED",
	}
	return mapping[stateName]
}
func (s *Service) notifyBestEffort(ctx context.Context, userID int64, title, message, link, nType string) {
	if s.notificationClient == nil || userID <= 0 {
		return
	}
	nctx := metadata.AppendToOutgoingContext(ctx, "x-internal-service", "admin_service")
	_, err := s.notificationClient.SendNotification(nctx, &notificationv1.SendNotificationRequest{
		UserId:  userID,
		Title:   title,
		Message: message,
		Link:    link,
		Type:    nType,
	})
	if err != nil {
		s.logger.Warn("send notification failed", zap.Int64("user_id", userID), zap.Error(err))
	}
}
func (s *Service) notifyTeamBestEffort(ctx context.Context, teamID int64, title, message, link, nType string) {
	if s.notificationClient == nil || teamID <= 0 {
		return
	}

	t, err := s.teamClient.GetTeam(s.internalCtx(ctx), &teamv1.GetTeamRequest{TeamId: teamID})
	if err != nil || t == nil {
		s.logger.Warn("notifyTeamBestEffort: GetTeam failed", zap.Int64("team_id", teamID), zap.Error(err))
		return
	}

	seen := make(map[int64]struct{}, len(t.Members))
	for _, m := range t.Members {
		if m == nil || m.UserId <= 0 {
			continue
		}
		if _, ok := seen[m.UserId]; ok {
			continue
		}
		seen[m.UserId] = struct{}{}
		s.notifyBestEffort(ctx, m.UserId, title, message, link, nType)
	}
}
