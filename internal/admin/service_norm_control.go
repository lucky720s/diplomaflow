package admin

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Service) NormListPending(ctx context.Context, filter NormCheckFilter) ([]*NormControlCheck, int64, error) {
	return s.repo.ListNormChecks(ctx, filter)
}

func (s *Service) NormGetDocument(ctx context.Context, submissionID string) (*NormControlCheck, []*NormControlIssue, error) {
	// запрет работы с не-NORM_CONTROL submission
	if _, err := s.assertNormSubmission(ctx, submissionID); err != nil {
		return nil, nil, err
	}

	check, err := s.repo.EnsureNormCheckForSubmission(ctx, submissionID)
	if err != nil {
		return nil, nil, err
	}

	issues, err := s.repo.ListNormIssues(ctx, submissionID)
	if err != nil {
		return nil, nil, err
	}

	return check, issues, nil
}

func (s *Service) NormStartReview(ctx context.Context, submissionID string, actorID int64) (*NormControlCheck, error) {
	if actorID <= 0 {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	sub, err := s.assertNormSubmission(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	err = s.assertProjectInNormControl(ctx, sub.ProjectID)
	if err != nil {
		return nil, err
	}

	_, err = s.repo.EnsureNormCheckForSubmission(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	r, ok := s.repo.(*repository)
	if !ok || r.db == nil {
		return nil, status.Error(codes.Internal, "repo misconfigured")
	}

	now := time.Now().UTC()

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var c NormControlCheck

		txErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&c, "submission_id = ?", submissionID).Error
		if txErr != nil {
			return txErr
		}
		if c.Status != NormStatusSubmitted {
			return status.Error(codes.FailedPrecondition, "cannot start review in current status")
		}

		c.Status = NormStatusInReview
		c.StartedAt = &now
		c.CheckerID = &actorID
		c.UpdatedAt = now

		txErr = tx.Save(&c).Error
		if txErr != nil {
			return txErr
		}

		// sync admin_submissions.reviewer_id (и updated_at)
		txErr = tx.Model(&Submission{}).
			Where("id = ?", submissionID).
			Updates(map[string]interface{}{
				"reviewer_id": actorID,
				"updated_at":  now,
			}).Error
		if txErr != nil {
			return txErr
		}

		// history
		sid := submissionID
		act := actorID
		txErr = tx.Create(&NormControlHistory{
			ProjectID:    c.ProjectID,
			SubmissionID: &sid,
			Action:       "started_review",
			ActorID:      &act,
			Comment:      "",
			CreatedAt:    now,
		}).Error
		return txErr
	})

	if err != nil {
		return nil, err
	}

	return s.repo.GetNormCheck(ctx, submissionID)
}

func (s *Service) NormAddIssue(ctx context.Context, submissionID string, actorID int64, issue *NormControlIssue) (*NormControlIssue, error) {
	if actorID <= 0 {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}
	if issue == nil || issue.Description == "" || issue.Category == "" || issue.Severity == "" {
		return nil, status.Error(codes.InvalidArgument, "category, severity, description are required")
	}

	sub, err := s.assertNormSubmission(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	err = s.assertProjectInNormControl(ctx, sub.ProjectID)
	if err != nil {
		return nil, err
	}

	check, err := s.repo.GetNormCheck(ctx, submissionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_, e2 := s.repo.EnsureNormCheckForSubmission(ctx, submissionID)
			if e2 != nil {
				return nil, e2
			}
			check, err = s.repo.GetNormCheck(ctx, submissionID)
		}
	}
	if err != nil {
		return nil, err
	}

	if check.Status != NormStatusInReview {
		return nil, status.Error(codes.FailedPrecondition, "document is not in_review")
	}
	if check.CheckerID == nil || *check.CheckerID != actorID {
		return nil, status.Error(codes.PermissionDenied, "only assigned checker can add issues")
	}

	issue.SubmissionID = submissionID
	err = s.repo.CreateNormIssue(ctx, issue)
	if err != nil {
		return nil, err
	}

	sid := submissionID
	act := actorID
	_ = s.repo.AddNormHistory(ctx, &NormControlHistory{
		ProjectID:    check.ProjectID,
		SubmissionID: &sid,
		Action:       "issue_added",
		ActorID:      &act,
		Comment:      issue.Description,
	})

	return issue, nil
}

func (s *Service) NormUpdateIssue(ctx context.Context, issueID string, actorID int64, upd *NormControlIssue) (*NormControlIssue, error) {
	if issueID == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if actorID <= 0 {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	issue, err := s.repo.GetNormIssue(ctx, issueID)
	if err != nil {
		return nil, err
	}

	check, err := s.repo.GetNormCheck(ctx, issue.SubmissionID)
	if err != nil {
		return nil, err
	}
	if check.Status != NormStatusInReview {
		return nil, status.Error(codes.FailedPrecondition, "document is not in_review")
	}
	if check.CheckerID == nil || *check.CheckerID != actorID {
		return nil, status.Error(codes.PermissionDenied, "only assigned checker can update issues")
	}

	// Apply updates (all fields as full update)
	if upd != nil {
		if upd.Category != "" {
			issue.Category = upd.Category
		}
		if upd.Severity != "" {
			issue.Severity = upd.Severity
		}
		if upd.Description != "" {
			issue.Description = upd.Description
		}
		issue.PageNumber = upd.PageNumber
		issue.LineNumber = upd.LineNumber

		if upd.IsResolved && !issue.IsResolved {
			now := time.Now().UTC()
			issue.IsResolved = true
			issue.ResolvedAt = &now
		} else if !upd.IsResolved && issue.IsResolved {
			issue.IsResolved = false
			issue.ResolvedAt = nil
		}
	}

	if err := s.repo.UpdateNormIssue(ctx, issue); err != nil {
		return nil, err
	}

	subID := issue.SubmissionID
	act := actorID
	_ = s.repo.AddNormHistory(ctx, &NormControlHistory{
		ProjectID:    check.ProjectID,
		SubmissionID: &subID,
		Action:       "issue_updated",
		ActorID:      &act,
		Comment:      issue.Description,
	})

	return issue, nil
}

func (s *Service) NormDeleteIssue(ctx context.Context, issueID string, actorID int64) error {
	if issueID == "" {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	if actorID <= 0 {
		return status.Error(codes.Unauthenticated, "unauthorized")
	}

	issue, err := s.repo.GetNormIssue(ctx, issueID)
	if err != nil {
		return err
	}
	check, err := s.repo.GetNormCheck(ctx, issue.SubmissionID)
	if err != nil {
		return err
	}
	if check.Status != NormStatusInReview {
		return status.Error(codes.FailedPrecondition, "document is not in_review")
	}
	if check.CheckerID == nil || *check.CheckerID != actorID {
		return status.Error(codes.PermissionDenied, "only assigned checker can delete issues")
	}

	if err := s.repo.DeleteNormIssue(ctx, issueID); err != nil {
		return err
	}

	subID := issue.SubmissionID
	act := actorID
	_ = s.repo.AddNormHistory(ctx, &NormControlHistory{
		ProjectID:    check.ProjectID,
		SubmissionID: &subID,
		Action:       "issue_deleted",
		ActorID:      &act,
		Comment:      "",
	})

	return nil
}
func (s *Service) NormApprove(ctx context.Context, submissionID string, actorID int64, comment string) (*NormControlCheck, error) {
	if submissionID == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if actorID <= 0 {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	sub, err := s.assertNormSubmission(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	err = s.assertProjectInNormControl(ctx, sub.ProjectID)
	if err != nil {
		return nil, err
	}

	r, ok := s.repo.(*repository)
	if !ok || r.db == nil {
		return nil, status.Error(codes.Internal, "repo misconfigured")
	}

	_, err = s.repo.EnsureNormCheckForSubmission(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	// 1) DB approve (без внешних вызовов внутри транзакции)
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var c NormControlCheck

		txErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&c, "submission_id = ?", submissionID).Error
		if txErr != nil {
			return txErr
		}
		if c.Status != NormStatusInReview {
			return status.Error(codes.FailedPrecondition, "document is not in_review")
		}
		if c.CheckerID == nil || *c.CheckerID != actorID {
			return status.Error(codes.PermissionDenied, "only assigned checker can approve")
		}

		var cnt int64
		txErr = tx.Model(&NormControlIssue{}).
			Where("submission_id = ? AND severity = ? AND is_resolved = false", submissionID, "critical").
			Count(&cnt).Error
		if txErr != nil {
			return txErr
		}
		if cnt > 0 {
			return status.Error(codes.FailedPrecondition, "cannot approve: unresolved critical issues exist")
		}

		c.Status = NormStatusApproved
		c.OverallComment = comment
		c.ReviewedAt = &now
		c.UpdatedAt = now

		txErr = tx.Save(&c).Error
		if txErr != nil {
			return txErr
		}

		var ssub Submission
		txErr = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&ssub, "id = ?", submissionID).Error
		if txErr != nil {
			return txErr
		}

		ssub.Status = StatusApproved
		ssub.ReviewerID = &actorID
		ssub.ReviewComment = comment
		ssub.ReviewedAt = &now
		ssub.UpdatedAt = now

		txErr = tx.Save(&ssub).Error
		return txErr
	})
	if err != nil {
		return nil, err
	}

	check, err := s.repo.GetNormCheck(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	// 2) Workflow transition. teamID нужен для уведомления; лидера тут не
	//    используем — двигаем напрямую (см. ниже).
	_, teamID, err := s.normTeamLeader(ctx, check.ProjectID, check.TeamID)
	if err != nil {
		// откатить approve, чтобы не оставлять approved
		_ = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			_ = tx.Model(&NormControlCheck{}).
				Where("submission_id = ?", submissionID).
				Updates(map[string]interface{}{
					"status":          NormStatusInReview,
					"overall_comment": "",
					"reviewed_at":     nil,
					"updated_at":      time.Now().UTC(),
				}).Error

			_ = tx.Model(&Submission{}).
				Where("id = ?", submissionID).
				Updates(map[string]interface{}{
					"status":         StatusPending,
					"reviewer_id":    nil,
					"review_comment": "",
					"reviewed_at":    nil,
					"updated_at":     time.Now().UTC(),
				}).Error
			return nil
		})
		return nil, err
	}

	// Двигаем workflow напрямую (NORM_CONTROL → ECONOMICS по событию
	// NORMCONTROL_SUBMITTED). tryPerformIfAvailable шёл через движок с проверкой
	// ролей/условий и молча пропускал переход — проект застревал на NORM_CONTROL,
	// и студент грузил файл заново. Прямой вызов ищет переход по from_state+event
	// и не блокируется условиями (тот же паттерн, что у предзащиты и тем).
	wfErr := s.repo.AdvanceProjectByWorkflowEvent(ctx, check.ProjectID, "NORMCONTROL_SUBMITTED", actorID, comment)
	if wfErr != nil {
		_ = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			_ = tx.Model(&NormControlCheck{}).
				Where("submission_id = ?", submissionID).
				Updates(map[string]interface{}{
					"status":          NormStatusInReview,
					"overall_comment": "",
					"reviewed_at":     nil,
					"updated_at":      time.Now().UTC(),
				}).Error

			_ = tx.Model(&Submission{}).
				Where("id = ?", submissionID).
				Updates(map[string]interface{}{
					"status":         StatusPending,
					"reviewer_id":    nil,
					"review_comment": "",
					"reviewed_at":    nil,
					"updated_at":     time.Now().UTC(),
				}).Error
			return nil
		})

		return nil, status.Errorf(codes.Internal, "workflow transition failed: %v", wfErr)
	}

	// 3) Audit/history (best-effort)
	_ = s.repo.CreateSubmissionReview(ctx, &SubmissionReview{
		SubmissionID: submissionID,
		ReviewerID:   actorID,
		Action:       "approve",
		Comment:      comment,
	})

	sid := submissionID
	act := actorID
	_ = s.repo.AddNormHistory(ctx, &NormControlHistory{
		ProjectID:    check.ProjectID,
		SubmissionID: &sid,
		Action:       "approved",
		ActorID:      &act,
		Comment:      comment,
	})

	// 4) Notify team
	s.notifyTeamBestEffort(ctx, teamID,
		"Документ по нормоконтролю утверждён",
		"Нормоконтроль пройден.",
		"/diplom",
		"norm_control_approved",
	)

	return s.repo.GetNormCheck(ctx, submissionID)
}

func (s *Service) NormReturn(ctx context.Context, submissionID string, actorID int64, comment string) (*NormControlCheck, error) {
	if submissionID == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if actorID <= 0 {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	sub, err := s.assertNormSubmission(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	err = s.assertProjectInNormControl(ctx, sub.ProjectID)
	if err != nil {
		return nil, err
	}

	r, ok := s.repo.(*repository)
	if !ok || r.db == nil {
		return nil, status.Error(codes.Internal, "repo misconfigured")
	}

	_, err = s.repo.EnsureNormCheckForSubmission(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var c NormControlCheck

		txErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&c, "submission_id = ?", submissionID).Error
		if txErr != nil {
			return txErr
		}
		if c.Status != NormStatusInReview {
			return status.Error(codes.FailedPrecondition, "document is not in_review")
		}
		if c.CheckerID == nil || *c.CheckerID != actorID {
			return status.Error(codes.PermissionDenied, "only assigned checker can return")
		}

		c.Status = NormStatusReturned
		c.OverallComment = comment
		c.ReviewedAt = &now
		c.UpdatedAt = now

		txErr = tx.Save(&c).Error
		if txErr != nil {
			return txErr
		}

		var ssub Submission
		txErr = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&ssub, "id = ?", submissionID).Error
		if txErr != nil {
			return txErr
		}

		ssub.Status = StatusRevisionRequested
		ssub.ReviewerID = &actorID
		ssub.ReviewComment = comment
		ssub.ReviewedAt = &now
		ssub.UpdatedAt = now

		txErr = tx.Save(&ssub).Error
		if txErr != nil {
			return txErr
		}

		_ = tx.Create(&SubmissionReview{
			SubmissionID: submissionID,
			ReviewerID:   actorID,
			Action:       "request_changes",
			Comment:      comment,
			CreatedAt:    now,
		}).Error

		sid := submissionID
		act := actorID
		_ = tx.Create(&NormControlHistory{
			ProjectID:    c.ProjectID,
			SubmissionID: &sid,
			Action:       "returned",
			ActorID:      &act,
			Comment:      comment,
			CreatedAt:    now,
		}).Error

		return nil
	})
	if err != nil {
		return nil, err
	}

	check, err := s.repo.GetNormCheck(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	// Notify team
	_, teamID, _ := s.normTeamLeader(ctx, check.ProjectID, check.TeamID)
	msg := comment
	if msg == "" {
		msg = "Требуются исправления."
	}
	s.notifyTeamBestEffort(ctx, teamID,
		"Нормоконтроль: требуются исправления",
		msg,
		"/diplom",
		"norm_control_returned",
	)

	return check, nil
}

func (s *Service) NormHistory(ctx context.Context, projectID int64) ([]*NormControlHistory, error) {
	if projectID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}
	return s.repo.ListNormHistory(ctx, projectID)
}

func (s *Service) NormStats(ctx context.Context, departmentID int64) (map[string]int64, error) {
	if departmentID <= 0 {
		return map[string]int64{}, nil
	}
	return s.repo.StatsNormIssuesByCategory(ctx, departmentID)
}

func (s *Service) NormListChecklists(ctx context.Context) ([]*NormControlChecklist, error) {
	return s.repo.ListNormChecklists(ctx)
}

func (s *Service) NormCreateChecklist(ctx context.Context, actorID int64, name, category, itemsJSON string) (*NormControlChecklist, error) {
	if actorID <= 0 {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}
	if name == "" || category == "" || itemsJSON == "" {
		return nil, status.Error(codes.InvalidArgument, "name, category, items_json are required")
	}
	// validate JSON
	var tmp any
	if err := json.Unmarshal([]byte(itemsJSON), &tmp); err != nil {
		return nil, status.Error(codes.InvalidArgument, "items_json must be valid JSON")
	}

	b := []byte(itemsJSON)
	c := &NormControlChecklist{
		Name:      name,
		Category:  category,
		Items:     b,
		CreatedBy: &actorID,
		IsActive:  true,
	}
	if err := s.repo.CreateNormChecklist(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// ===== helpers (norm-control) =====

func (s *Service) assertNormSubmission(ctx context.Context, submissionID string) (*Submission, error) {
	if submissionID == "" {
		return nil, status.Error(codes.InvalidArgument, "submission_id is required")
	}

	sub, err := s.repo.GetSubmission(ctx, submissionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "submission not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load submission: %v", err)
	}
	if sub == nil {
		return nil, status.Error(codes.NotFound, "submission not found")
	}

	// Resolve state name from DB (states.id = submission.StepID)
	r, ok := s.repo.(*repository)
	if !ok || r == nil || r.db == nil {
		return nil, status.Error(codes.Internal, "repo misconfigured")
	}

	var stateName string
	if err := r.db.WithContext(ctx).
		Raw("SELECT name FROM states WHERE id = ? LIMIT 1", sub.StepID).
		Scan(&stateName).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resolve step name: %v", err)
	}

	if stateName != "NORM_CONTROL" {
		return nil, status.Error(codes.FailedPrecondition, "submission is not for NORM_CONTROL step")
	}

	return sub, nil
}

func (s *Service) assertProjectInNormControl(ctx context.Context, projectID int64) error {
	if projectID <= 0 {
		return status.Error(codes.InvalidArgument, "project_id is required")
	}
	rt, err := s.projectClient.GetProjectRuntime(s.internalCtx(ctx), &projectv1.GetProjectRuntimeRequest{
		ProjectId: projectID,
	})
	if err != nil {
		return status.Errorf(codes.Internal, "GetProjectRuntime failed: %v", err)
	}
	if rt == nil {
		return status.Error(codes.Internal, "project runtime is nil")
	}
	if rt.CurrentStateName != "NORM_CONTROL" {
		return status.Error(codes.FailedPrecondition, "project is not in NORM_CONTROL state")
	}
	return nil
}

func (s *Service) normTeamLeader(ctx context.Context, projectID int64, teamIDPtr *int64) (int64, int64, error) {
	teamID := int64(0)
	if teamIDPtr != nil {
		teamID = *teamIDPtr
	}
	tid, err := s.resolveTeamIDByProject(ctx, projectID, teamID)
	if err != nil {
		return 0, 0, status.Errorf(codes.Internal, "failed to resolve team_id: %v", err)
	}

	tc, err := s.repo.GetTeamContext(ctx, tid)
	if err != nil || tc == nil || tc.LeaderUserID <= 0 {
		return 0, 0, status.Error(codes.Internal, "failed to resolve team leader")
	}
	return tc.LeaderUserID, tid, nil
}
