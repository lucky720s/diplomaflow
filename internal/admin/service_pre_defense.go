package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SubmitPreDefense - студент/лидер подаёт заявку на предзащиту
func (s *Service) SubmitPreDefense(ctx context.Context, teamID, projectID, submittedBy int64, message string, documentIDs []string) (string, *PreDefenseSubmission, error) {
	resolvedTeamID, err := s.resolveTeamIDByProject(ctx, projectID, teamID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve team: %w", err)
	}

	var supervisorID *int64
	assignment, assignErr := s.repo.GetSupervisorAssignment(ctx, resolvedTeamID)
	if assignErr == nil && assignment != nil {
		supervisorID = &assignment.SupervisorID
	}

	now := time.Now().UTC()

	sub := &PreDefenseSubmission{
		ID:           uuid.New().String(),
		TeamID:       resolvedTeamID,
		ProjectID:    projectID,
		SupervisorID: supervisorID,
		SubmittedBy:  submittedBy,
		Status:       "pending",
		SubmittedAt:  now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreatePreDefenseSubmission(ctx, sub); err != nil {
		return "", nil, fmt.Errorf("failed to create pre-defense submission: %w", err)
	}

	for _, fileID := range documentIDs {
		doc := &PreDefenseDocument{
			ID:           uuid.New().String(),
			SubmissionID: sub.ID,
			FileName:     fileID,
			UploadedBy:   &submittedBy,
			UploadedAt:   now,
			CreatedAt:    now,
		}
		if docErr := s.repo.AddPreDefenseDocument(ctx, doc); docErr != nil {
			s.logger.Warn("Failed to add pre-defense document", zap.Error(docErr))
		}
	}

	_ = s.repo.AddPreDefenseHistory(ctx, &PreDefenseHistory{
		SubmissionID: sub.ID,
		Action:       "submitted",
		ActorID:      submittedBy,
		OldValue:     "",
		NewValue:     "pending",
		Comment:      message,
		CreatedAt:    now,
	})

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

	s.notifyTeamBestEffort(ctx, resolvedTeamID,
		"Заявка на предзащиту отправлена",
		"Заявка создана и отправлена на рассмотрение.",
		"/diplom",
		"predefense_submitted",
	)

	if supervisorID != nil && *supervisorID > 0 {
		s.notifyBestEffort(ctx, *supervisorID,
			"Новая заявка на предзащиту",
			fmt.Sprintf("Команда %d подала заявку на предзащиту.", resolvedTeamID),
			"/teacher/pre-defenses",
			"predefense_submitted_to_supervisor",
		)
	}

	return sub.ID, sub, nil
}

// SchedulePreDefense - админ/секретарь назначает дату предзащиты
func (s *Service) SchedulePreDefense(ctx context.Context, submissionID string, scheduledDate time.Time, scheduledTime, location, meetingLink string, durationMinutes int32, commissionMemberIDs []int64, scheduledBy int64, comment string) error {
	sub, err := s.repo.GetPreDefenseSubmission(ctx, submissionID)
	if err != nil {
		return status.Errorf(codes.NotFound, "submission not found: %v", err)
	}

	if sub.Status != "pending" && sub.Status != "rescheduled" {
		return status.Errorf(codes.FailedPrecondition, "cannot schedule pre-defense in status '%s'", sub.Status)
	}

	now := time.Now().UTC()
	oldStatus := sub.Status

	sub.ScheduledDate = &scheduledDate
	sub.ScheduledTime = scheduledTime
	sub.Location = location
	sub.MeetingLink = meetingLink
	sub.DurationMinutes = durationMinutes
	sub.Status = "scheduled"
	sub.UpdatedAt = now

	if err := s.repo.UpdatePreDefenseSubmission(ctx, sub); err != nil {
		return fmt.Errorf("failed to update submission: %w", err)
	}

	for _, memberID := range commissionMemberIDs {
		member := &PreDefenseCommissionMember{
			SubmissionID: sub.ID,
			UserID:       memberID,
			Role:         "member",
			CreatedAt:    now,
		}
		if addErr := s.repo.AddCommissionMember(ctx, member); addErr != nil {
			s.logger.Warn("Failed to add commission member",
				zap.Int64("user_id", memberID),
				zap.Error(addErr),
			)
		}
	}

	_ = s.repo.AddPreDefenseHistory(ctx, &PreDefenseHistory{
		SubmissionID: sub.ID,
		Action:       "scheduled",
		ActorID:      scheduledBy,
		OldValue:     oldStatus,
		NewValue:     "scheduled",
		Comment:      comment,
		CreatedAt:    now,
	})

	s.logger.Info("Pre-defense scheduled",
		zap.String("submission_id", sub.ID),
		zap.Time("date", scheduledDate),
		zap.String("time", scheduledTime),
	)

	msg := fmt.Sprintf("Назначено: %s %s. Локация: %s", scheduledDate.Format("2006-01-02"), scheduledTime, location)
	if meetingLink != "" {
		msg += fmt.Sprintf(". Ссылка: %s", meetingLink)
	}

	s.notifyTeamBestEffort(ctx, sub.TeamID,
		"Предзащита назначена",
		msg,
		"/diplom",
		"predefense_scheduled",
	)

	if sub.SupervisorID != nil && *sub.SupervisorID > 0 {
		s.notifyBestEffort(ctx, *sub.SupervisorID,
			"Предзащита назначена",
			msg,
			"/teacher/pre-defenses",
			"predefense_scheduled_supervisor",
		)
	}

	for _, uid := range commissionMemberIDs {
		if uid > 0 {
			s.notifyBestEffort(ctx, uid,
				"Вы добавлены в комиссию предзащиты",
				msg,
				"/teacher/pre-defenses",
				"predefense_commission_member_added",
			)
		}
	}

	return nil
}

// GradePreDefense - комиссия выставляет оценку
func (s *Service) GradePreDefense(
	ctx context.Context,
	submissionID string,
	gradedBy int64,
	grade int32,
	comment string,
	memberGrades []MemberGradeInput,
) error {
	sub, err := s.repo.GetPreDefenseSubmission(ctx, submissionID)
	if err != nil {
		return status.Errorf(codes.NotFound, "submission not found: %v", err)
	}

	if sub.Status != "scheduled" {
		return status.Errorf(codes.FailedPrecondition, "cannot grade pre-defense in status '%s'", sub.Status)
	}

	now := time.Now().UTC()

	sub.Status = "graded"
	sub.Grade = &grade
	sub.GradeComment = comment
	sub.GradedBy = &gradedBy
	sub.GradedAt = &now
	sub.UpdatedAt = now

	if err := s.repo.UpdatePreDefenseSubmission(ctx, sub); err != nil {
		return fmt.Errorf("update pre-defense submission: %w", err)
	}

	for _, mg := range memberGrades {
		if err := s.repo.UpdateCommissionMemberGrade(ctx, submissionID, mg.MemberID, mg.Grade, mg.Comment); err != nil {
			return fmt.Errorf("update commission member grade: %w", err)
		}
	}

	if err := s.repo.AddPreDefenseHistory(ctx, &PreDefenseHistory{
		SubmissionID: submissionID,
		Action:       "graded",
		ActorID:      gradedBy,
		OldValue:     "scheduled",
		NewValue:     "graded",
		Comment:      fmt.Sprintf("Grade: %d. %s", grade, comment),
		CreatedAt:    now,
	}); err != nil {
		return fmt.Errorf("add pre-defense history: %w", err)
	}

	// Чтобы заявка предзащиты не висела в общей очереди admin_submissions как pending.
	if err := s.repo.MarkPreDefenseWorkflowSubmissionReviewed(
		ctx,
		sub.ProjectID,
		sub.TeamID,
		gradedBy,
		"approved",
		fmt.Sprintf("Pre-defense graded: %d. %s", grade, comment),
	); err != nil {
		s.logger.Warn("failed to sync workflow submission after pre-defense grade",
			zap.String("submission_id", submissionID),
			zap.Int64("project_id", sub.ProjectID),
			zap.Int64("team_id", sub.TeamID),
			zap.Error(err),
		)
	}

	return nil
}

// CompletePreDefense - завершение предзащиты с результатом.
// ВАЖНО: сначала двигаем workflow проекта, и только потом помечаем саму предзащиту completed/failed.
// Иначе получится рассинхрон: admin_pre_defense_submissions уже completed, а projects.current_state_name
// всё ещё PRE_DEFENSE_1_REVIEW/PRE_DEFENSE_2_REVIEW.
func (s *Service) CompletePreDefense(ctx context.Context, submissionID string, completedBy int64, result, resultComment string, recommendations []string, allowResubmission bool) error {
	sub, err := s.repo.GetPreDefenseSubmission(ctx, submissionID)
	if err != nil {
		return status.Errorf(codes.NotFound, "submission not found: %v", err)
	}

	if sub.Status != "graded" {
		return status.Errorf(codes.FailedPrecondition, "cannot complete pre-defense in status '%s', must be 'graded'", sub.Status)
	}

	var finalStatus string
	var workflowStatus string
	var workflowEvents []string

	switch result {
	case "passed":
		finalStatus = "completed"
		workflowStatus = "approved"
		workflowEvents = []string{"PREDEFENSE1_PASSED", "PREDEFENSE2_PASSED"}
	case "conditional":
		// Conditional тоже считается допуском вперёд по workflow, но статус предзащиты оставляем отдельным.
		finalStatus = "conditional"
		workflowStatus = "approved"
		workflowEvents = []string{"PREDEFENSE1_PASSED", "PREDEFENSE2_PASSED"}
	case "failed":
		workflowStatus = "rejected"
		workflowEvents = []string{"PREDEFENSE1_FAILED", "PREDEFENSE2_FAILED"}
		if allowResubmission {
			finalStatus = "failed_resubmit"
		} else {
			finalStatus = "failed"
		}
	default:
		return status.Errorf(codes.InvalidArgument, "invalid result: %s, must be 'passed', 'failed', or 'conditional'", result)
	}

	// Главный фикс: не используем старый несуществующий PREDEFENSE_PASSED и не игнорируем ошибку.
	// AdvanceProjectByWorkflowEvent сам найдёт transition из текущего current_state_id проекта.
	// Пробуем сначала event для первой предзащиты, потом для второй. Сработает только тот,
	// у которого from_state совпадает с текущим projects.current_state_id.
	if err := s.advancePreDefenseWorkflow(ctx, sub.ProjectID, completedBy, resultComment, workflowEvents); err != nil {
		return status.Errorf(codes.FailedPrecondition,
			"failed to advance project workflow after pre-defense complete: %v", err)
	}

	now := time.Now().UTC()
	oldStatus := sub.Status

	sub.Result = result
	sub.ResultComment = resultComment
	sub.CompletedAt = &now
	sub.UpdatedAt = now
	sub.Status = finalStatus

	if err := s.repo.UpdatePreDefenseSubmission(ctx, sub); err != nil {
		return fmt.Errorf("failed to update pre-defense submission after workflow advance: %w", err)
	}

	if err := s.repo.AddPreDefenseHistory(ctx, &PreDefenseHistory{
		SubmissionID: sub.ID,
		Action:       "completed",
		ActorID:      completedBy,
		OldValue:     oldStatus,
		NewValue:     sub.Status,
		Comment:      fmt.Sprintf("Result: %s. %s", result, resultComment),
		CreatedAt:    now,
	}); err != nil {
		return fmt.Errorf("add pre-defense history: %w", err)
	}

	// Финальная синхронизация обычной workflow submission.
	if err := s.repo.MarkPreDefenseWorkflowSubmissionReviewed(
		ctx,
		sub.ProjectID,
		sub.TeamID,
		completedBy,
		workflowStatus,
		fmt.Sprintf("Pre-defense completed: %s. %s", result, resultComment),
	); err != nil {
		s.logger.Warn("failed to sync workflow submission after pre-defense complete",
			zap.String("submission_id", submissionID),
			zap.Int64("project_id", sub.ProjectID),
			zap.Int64("team_id", sub.TeamID),
			zap.String("workflow_status", workflowStatus),
			zap.Error(err),
		)
	}

	s.notifyTeamBestEffort(ctx, sub.TeamID,
		"Предзащита завершена",
		fmt.Sprintf("Результат: %s. %s", sub.Status, resultComment),
		"/diplom",
		"predefense_completed",
	)

	return nil
}

func (s *Service) advancePreDefenseWorkflow(ctx context.Context, projectID, actorID int64, comment string, eventNames []string) error {
	var lastErr error
	for _, eventName := range eventNames {
		if eventName == "" {
			continue
		}

		err := s.repo.AdvanceProjectByWorkflowEvent(ctx, projectID, eventName, actorID, comment)
		if err == nil {
			s.logger.Info("pre-defense workflow advanced",
				zap.Int64("project_id", projectID),
				zap.Int64("actor_id", actorID),
				zap.String("event_name", eventName),
			)
			return nil
		}

		lastErr = err
		s.logger.Debug("pre-defense workflow event did not match current project state",
			zap.Int64("project_id", projectID),
			zap.String("event_name", eventName),
			zap.Error(err),
		)
	}

	return fmt.Errorf("none of workflow events matched current project state for project_id=%d, events=%v, last_error=%w", projectID, eventNames, lastErr)
}

// ReschedulePreDefense - перенос предзащиты
func (s *Service) ReschedulePreDefense(ctx context.Context, submissionID string, rescheduledBy int64, newDate time.Time, newTime, newLocation, reason string) error {
	sub, err := s.repo.GetPreDefenseSubmission(ctx, submissionID)
	if err != nil {
		return status.Errorf(codes.NotFound, "submission not found: %v", err)
	}

	if sub.Status != "scheduled" {
		return status.Errorf(codes.FailedPrecondition, "can only reschedule a 'scheduled' pre-defense, current: '%s'", sub.Status)
	}

	now := time.Now().UTC()
	oldStatus := sub.Status

	sub.ScheduledDate = &newDate
	sub.ScheduledTime = newTime
	if newLocation != "" {
		sub.Location = newLocation
	}
	sub.Status = "scheduled"
	sub.UpdatedAt = now

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
		CreatedAt:    now,
	})

	msg := fmt.Sprintf("Новая дата: %s %s. Причина: %s", newDate.Format("2006-01-02"), newTime, reason)

	s.notifyTeamBestEffort(ctx, sub.TeamID,
		"Предзащита перенесена",
		msg,
		"/diplom",
		"predefense_rescheduled",
	)

	if sub.SupervisorID != nil && *sub.SupervisorID > 0 {
		s.notifyBestEffort(ctx, *sub.SupervisorID,
			"Предзащита перенесена",
			msg,
			"/teacher/pre-defenses",
			"predefense_rescheduled_supervisor",
		)
	}

	members, _ := s.repo.GetCommissionMembers(ctx, sub.ID)
	for _, m := range members {
		if m.UserID > 0 {
			s.notifyBestEffort(ctx, m.UserID,
				"Предзащита перенесена",
				msg,
				"/teacher/pre-defenses",
				"predefense_rescheduled_commission",
			)
		}
	}

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
		return nil, nil, status.Errorf(codes.NotFound, "submission not found: %v", err)
	}

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
	now := time.Now().UTC()

	member := &PreDefenseCommissionMember{
		SubmissionID: submissionID,
		UserID:       userID,
		Role:         role,
		CreatedAt:    now,
	}

	if err := s.repo.AddCommissionMember(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add commission member: %w", err)
	}

	_ = s.repo.AddPreDefenseHistory(ctx, &PreDefenseHistory{
		SubmissionID: submissionID,
		Action:       "commission_member_added",
		ActorID:      addedBy,
		NewValue:     fmt.Sprintf("user_id=%d, role=%s", userID, role),
		CreatedAt:    now,
	})

	s.notifyBestEffort(ctx, userID,
		"Вы добавлены в комиссию предзащиты",
		fmt.Sprintf("Роль: %s", role),
		"/teacher/pre-defenses",
		"predefense_commission_member_added",
	)

	return member, nil
}

// RemovePreDefenseCommissionMember - удалить члена комиссии
func (s *Service) RemovePreDefenseCommissionMember(ctx context.Context, submissionID string, userID int64, removedBy int64, reason string) error {
	if err := s.repo.RemoveCommissionMember(ctx, submissionID, userID); err != nil {
		return fmt.Errorf("failed to remove commission member: %w", err)
	}

	now := time.Now().UTC()

	_ = s.repo.AddPreDefenseHistory(ctx, &PreDefenseHistory{
		SubmissionID: submissionID,
		Action:       "commission_member_removed",
		ActorID:      removedBy,
		OldValue:     fmt.Sprintf("user_id=%d", userID),
		Comment:      reason,
		CreatedAt:    now,
	})

	msg := "Вы удалены из комиссии предзащиты."
	if reason != "" {
		msg = msg + " Причина: " + reason
	}

	s.notifyBestEffort(ctx, userID,
		"Изменение комиссии предзащиты",
		msg,
		"/teacher/pre-defenses",
		"predefense_commission_member_removed",
	)

	return nil
}

// MemberGradeInput - входные данные для оценки члена комиссии
type MemberGradeInput struct {
	MemberID int64
	Grade    int32
	Comment  string
}
