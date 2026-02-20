package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SubmitPreDefense - студент/лидер подаёт заявку на предзащиту
func (s *Service) SubmitPreDefense(ctx context.Context, teamID, projectID, submittedBy int64, message string, documentIDs []string) (string, *PreDefenseSubmission, error) {
	// Resolve team_id через project если нужно
	resolvedTeamID, err := s.resolveTeamIDByProject(ctx, projectID, teamID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve team: %w", err)
	}

	// Получаем supervisor_id из admin_supervisor_assignments
	var supervisorID *int64
	assignment, assignErr := s.repo.GetSupervisorAssignment(ctx, resolvedTeamID)
	if assignErr == nil && assignment != nil {
		supervisorID = &assignment.SupervisorID
	}

	sub := &PreDefenseSubmission{
		ID:           uuid.New().String(),
		TeamID:       resolvedTeamID,
		ProjectID:    projectID,
		SupervisorID: supervisorID,
		SubmittedBy:  submittedBy,
		Status:       "pending",
		SubmittedAt:  time.Now().UTC(),
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.repo.CreatePreDefenseSubmission(ctx, sub); err != nil {
		return "", nil, fmt.Errorf("failed to create pre-defense submission: %w", err)
	}

	// Добавляем документы
	for _, fileID := range documentIDs {
		doc := &PreDefenseDocument{
			ID:           uuid.New().String(),
			SubmissionID: sub.ID,
			FileName:     fileID, // будет обогащено позже через file_service
			UploadedBy:   &submittedBy,
			UploadedAt:   time.Now().UTC(),
			CreatedAt:    time.Now().UTC(),
		}
		if docErr := s.repo.AddPreDefenseDocument(ctx, doc); docErr != nil {
			s.logger.Warn("Failed to add pre-defense document", zap.Error(docErr))
		}
	}

	// Записываем в историю
	_ = s.repo.AddPreDefenseHistory(ctx, &PreDefenseHistory{
		SubmissionID: sub.ID,
		Action:       "submitted",
		ActorID:      submittedBy,
		OldValue:     "",
		NewValue:     "pending",
		Comment:      message,
	})

	// Логируем активность
	_ = s.repo.LogActivity(ctx, &AdminActivity{
		ActivityType: "PRE_DEFENSE_SUBMITTED",
		Description:  fmt.Sprintf("Pre-defense submitted for project %d by user %d", projectID, submittedBy),
		ActorID:      submittedBy,
		TargetID:     projectID,
		TargetType:   "project",
	})

	s.logger.Info("Pre-defense submitted",
		zap.String("submission_id", sub.ID),
		zap.Int64("team_id", resolvedTeamID),
		zap.Int64("project_id", projectID),
	)

	return sub.ID, sub, nil
}

// SchedulePreDefense - админ/секретарь назначает дату предзащиты
func (s *Service) SchedulePreDefense(ctx context.Context, submissionID string, scheduledDate time.Time, scheduledTime, location, meetingLink string, durationMinutes int32, commissionMemberIDs []int64, scheduledBy int64, comment string) error {
	sub, err := s.repo.GetPreDefenseSubmission(ctx, submissionID)
	if err != nil {
		return fmt.Errorf("submission not found: %w", err)
	}

	if sub.Status != "pending" && sub.Status != "rescheduled" {
		return status.Errorf(codes.FailedPrecondition, "cannot schedule pre-defense in status '%s'", sub.Status)
	}

	oldStatus := sub.Status
	sub.ScheduledDate = &scheduledDate
	sub.ScheduledTime = scheduledTime
	sub.Location = location
	sub.MeetingLink = meetingLink
	sub.DurationMinutes = durationMinutes
	sub.Status = "scheduled"
	sub.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdatePreDefenseSubmission(ctx, sub); err != nil {
		return fmt.Errorf("failed to update submission: %w", err)
	}

	// Добавить членов комиссии
	for _, memberID := range commissionMemberIDs {
		member := &PreDefenseCommissionMember{
			SubmissionID: sub.ID,
			UserID:       memberID,
			Role:         "member",
			CreatedAt:    time.Now().UTC(),
		}
		if addErr := s.repo.AddCommissionMember(ctx, member); addErr != nil {
			s.logger.Warn("Failed to add commission member",
				zap.Int64("user_id", memberID),
				zap.Error(addErr),
			)
		}
	}

	// История
	_ = s.repo.AddPreDefenseHistory(ctx, &PreDefenseHistory{
		SubmissionID: sub.ID,
		Action:       "scheduled",
		ActorID:      scheduledBy,
		OldValue:     oldStatus,
		NewValue:     "scheduled",
		Comment:      comment,
	})

	s.logger.Info("Pre-defense scheduled",
		zap.String("submission_id", sub.ID),
		zap.Time("date", scheduledDate),
		zap.String("time", scheduledTime),
	)

	return nil
}

// GradePreDefense - оценивание предзащиты
func (s *Service) GradePreDefense(ctx context.Context, submissionID string, gradedBy int64, grade int32, comment string, memberGrades []MemberGradeInput) error {
	sub, err := s.repo.GetPreDefenseSubmission(ctx, submissionID)
	if err != nil {
		return fmt.Errorf("submission not found: %w", err)
	}

	if sub.Status != "scheduled" && sub.Status != "in_progress" {
		return status.Errorf(codes.FailedPrecondition, "cannot grade pre-defense in status '%s'", sub.Status)
	}

	now := time.Now().UTC()
	oldStatus := sub.Status
	sub.Grade = &grade
	sub.GradeComment = comment
	sub.GradedBy = &gradedBy
	sub.GradedAt = &now
	sub.Status = "graded"
	sub.UpdatedAt = now

	if err := s.repo.UpdatePreDefenseSubmission(ctx, sub); err != nil {
		return fmt.Errorf("failed to update submission: %w", err)
	}

	// Сохранить индивидуальные оценки членов комиссии
	for _, mg := range memberGrades {
		if updateErr := s.repo.UpdateCommissionMemberGrade(ctx, sub.ID, mg.MemberID, mg.Grade, mg.Comment); updateErr != nil {
			s.logger.Warn("Failed to update commission member grade",
				zap.Int64("member_id", mg.MemberID),
				zap.Error(updateErr),
			)
		}
	}

	rt, rtErr := s.projectClient.GetProjectRuntime(s.internalCtx(ctx), &projectv1.GetProjectRuntimeRequest{
		ProjectId: sub.ProjectID,
	})
	stepID := int64(0)
	if rtErr == nil && rt != nil {
		stepID = rt.CurrentStateId
	}
	if stepID == 0 {
		s.logger.Warn("StepID is 0 for pre-defense grade, grade may be indistinguishable",
			zap.String("submission_id", sub.ID),
			zap.Int64("project_id", sub.ProjectID),
		)
	}

	_, _ = s.SetStepGrade(ctx, &SetGradeRequest{
		ProjectID: sub.ProjectID,
		StepID:    stepID,
		Grade:     grade,
		Comment:   fmt.Sprintf("Pre-defense grade: %s", comment),
		GraderID:  gradedBy,
	})

	// История
	_ = s.repo.AddPreDefenseHistory(ctx, &PreDefenseHistory{
		SubmissionID: sub.ID,
		Action:       "graded",
		ActorID:      gradedBy,
		OldValue:     oldStatus,
		NewValue:     "graded",
		Comment:      fmt.Sprintf("Grade: %d. %s", grade, comment),
	})

	return nil
}

// CompletePreDefense - завершение предзащиты с результатом
func (s *Service) CompletePreDefense(ctx context.Context, submissionID string, completedBy int64, result, resultComment string, recommendations []string, allowResubmission bool) error {
	sub, err := s.repo.GetPreDefenseSubmission(ctx, submissionID)
	if err != nil {
		return fmt.Errorf("submission not found: %w", err)
	}

	if sub.Status != "graded" {
		return status.Errorf(codes.FailedPrecondition, "cannot complete pre-defense in status '%s', must be 'graded'", sub.Status)
	}

	now := time.Now().UTC()
	oldStatus := sub.Status
	sub.Result = result
	sub.ResultComment = resultComment
	sub.CompletedAt = &now
	sub.UpdatedAt = now

	switch result {
	case "passed":
		sub.Status = "completed"
	case "failed":
		if allowResubmission {
			sub.Status = "failed_resubmit"
		} else {
			sub.Status = "failed"
		}
	case "conditional":
		sub.Status = "conditional"
	default:
		return fmt.Errorf("invalid result: %s, must be 'passed', 'failed', or 'conditional'", result)
	}

	if err := s.repo.UpdatePreDefenseSubmission(ctx, sub); err != nil {
		return fmt.Errorf("failed to update submission: %w", err)
	}

	if result == "passed" {
		actorID, actorRole := s.callerFromContext(ctx, completedBy, "commission")
		_ = s.tryPerformIfAvailable(ctx, actorID, actorRole, sub.ProjectID,
			"PREDEFENSE_PASSED", map[string]interface{}{
				"source":        "admin_service",
				"submission_id": sub.ID,
				"result":        result,
				"grade":         sub.Grade,
			})
	}

	// История
	_ = s.repo.AddPreDefenseHistory(ctx, &PreDefenseHistory{
		SubmissionID: sub.ID,
		Action:       "completed",
		ActorID:      completedBy,
		OldValue:     oldStatus,
		NewValue:     sub.Status,
		Comment:      fmt.Sprintf("Result: %s. %s", result, resultComment),
	})

	return nil
}

// ReschedulePreDefense - перенос предзащиты
func (s *Service) ReschedulePreDefense(ctx context.Context, submissionID string, rescheduledBy int64, newDate time.Time, newTime, newLocation, reason string) error {
	sub, err := s.repo.GetPreDefenseSubmission(ctx, submissionID)
	if err != nil {
		return fmt.Errorf("submission not found: %w", err)
	}

	if sub.Status != "scheduled" {
		return status.Errorf(codes.FailedPrecondition, "can only reschedule a 'scheduled' pre-defense, current: '%s'", sub.Status)
	}

	oldStatus := sub.Status
	sub.ScheduledDate = &newDate
	sub.ScheduledTime = newTime
	if newLocation != "" {
		sub.Location = newLocation
	}
	sub.Status = "scheduled" // остаётся scheduled
	sub.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdatePreDefenseSubmission(ctx, sub); err != nil {
		return fmt.Errorf("failed to reschedule: %w", err)
	}

	_ = s.repo.AddPreDefenseHistory(ctx, &PreDefenseHistory{
		SubmissionID: sub.ID,
		Action:       "rescheduled",
		ActorID:      rescheduledBy,
		OldValue:     oldStatus,
		NewValue:     "scheduled",
		Comment:      reason,
	})

	return nil
}

func (s *Service) ListPreDefenseSubmissions(ctx context.Context, filter PreDefenseFilter) ([]*PreDefenseSubmission, int64, error) {
	subs, total, err := s.repo.ListPreDefenseSubmissions(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	for _, sub := range subs {
		s.enrichPreDefenseNames(ctx, sub)
	}

	return subs, total, nil
}

// GetPreDefenseSubmission - получение предзащиты с деталями
func (s *Service) GetPreDefenseSubmission(ctx context.Context, submissionID string) (*PreDefenseSubmission, []*PreDefenseHistory, error) {
	sub, err := s.repo.GetPreDefenseSubmission(ctx, submissionID)
	if err != nil {
		return nil, nil, fmt.Errorf("submission not found: %w", err)
	}

	// Загружаем связанные данные
	commission, _ := s.repo.GetCommissionMembers(ctx, submissionID)
	sub.Commission = commission

	documents, _ := s.repo.GetPreDefenseDocuments(ctx, submissionID)
	sub.Documents = documents
	s.enrichPreDefenseNames(ctx, sub)
	history, _ := s.repo.GetPreDefenseHistory(ctx, submissionID)

	return sub, history, nil
}

// ListScheduledPreDefenses - расписание предзащит
func (s *Service) ListScheduledPreDefenses(ctx context.Context, filter ScheduleFilter) ([]*PreDefenseSubmission, error) {
	return s.repo.ListScheduledPreDefenses(ctx, filter)
}

// AddPreDefenseCommissionMember - добавить члена комиссии
func (s *Service) AddPreDefenseCommissionMember(ctx context.Context, submissionID string, userID int64, role string, addedBy int64) (*PreDefenseCommissionMember, error) {
	member := &PreDefenseCommissionMember{
		SubmissionID: submissionID,
		UserID:       userID,
		Role:         role,
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.repo.AddCommissionMember(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add commission member: %w", err)
	}

	_ = s.repo.AddPreDefenseHistory(ctx, &PreDefenseHistory{
		SubmissionID: submissionID,
		Action:       "commission_member_added",
		ActorID:      addedBy,
		NewValue:     fmt.Sprintf("user_id=%d, role=%s", userID, role),
	})

	return member, nil
}

// RemovePreDefenseCommissionMember - удалить члена комиссии
func (s *Service) RemovePreDefenseCommissionMember(ctx context.Context, submissionID string, userID int64, removedBy int64, reason string) error {
	if err := s.repo.RemoveCommissionMember(ctx, submissionID, userID); err != nil {
		return fmt.Errorf("failed to remove commission member: %w", err)
	}

	_ = s.repo.AddPreDefenseHistory(ctx, &PreDefenseHistory{
		SubmissionID: submissionID,
		Action:       "commission_member_removed",
		ActorID:      removedBy,
		OldValue:     fmt.Sprintf("user_id=%d", userID),
		Comment:      reason,
	})

	return nil
}

// MemberGradeInput - входные данные для оценки члена комиссии
type MemberGradeInput struct {
	MemberID int64
	Grade    int32
	Comment  string
}
