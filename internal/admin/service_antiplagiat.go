package admin

import (
	"context"
	"errors"
	"time"

	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	antiplagStateName    = "ANTIPLAGIAT"
	antiplagEventApprove = "ANTIPLAGIAT_SUBMITTED" // ANTIPLAGIAT -> DEFENSE
	antiplagEventReject  = "ANTIPLAGIAT_REJECTED"  // ANTIPLAGIAT -> ANTIPLAGIAT
)

func (s *Service) AntiplagListPending(ctx context.Context, filter AntiplagCheckFilter) ([]*AntiplagCheck, int64, error) {
	return s.repo.ListAntiplagChecks(ctx, filter)
}

func (s *Service) AntiplagGetDocument(ctx context.Context, submissionID string) (*AntiplagCheck, []*AntiplagComment, error) {
	if _, err := s.assertAntiplagSubmission(ctx, submissionID); err != nil {
		return nil, nil, err
	}

	check, err := s.repo.EnsureAntiplagCheckForSubmission(ctx, submissionID)
	if err != nil {
		return nil, nil, err
	}

	comments, err := s.repo.ListAntiplagComments(ctx, submissionID)
	if err != nil {
		return nil, nil, err
	}

	return check, comments, nil
}

func (s *Service) AntiplagStartReview(ctx context.Context, submissionID string, actorID int64) (*AntiplagCheck, error) {
	if actorID <= 0 {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	sub, err := s.assertAntiplagSubmission(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	if err = s.assertProjectInAntiplagiat(ctx, sub.ProjectID); err != nil {
		return nil, err
	}
	if _, err = s.repo.EnsureAntiplagCheckForSubmission(ctx, submissionID); err != nil {
		return nil, err
	}

	r, ok := s.repo.(*repository)
	if !ok || r.db == nil {
		return nil, status.Error(codes.Internal, "repo misconfigured")
	}

	now := time.Now().UTC()

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var c AntiplagCheck
		if txErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&c, "submission_id = ?", submissionID).Error; txErr != nil {
			return txErr
		}
		if c.Status != AntiplagStatusSubmitted {
			return status.Error(codes.FailedPrecondition, "cannot start review in current status")
		}

		c.Status = AntiplagStatusInReview
		c.StartedAt = &now
		c.CheckerID = &actorID
		c.UpdatedAt = now

		if txErr := tx.Save(&c).Error; txErr != nil {
			return txErr
		}

		// sync admin_submissions.reviewer_id
		if txErr := tx.Model(&Submission{}).
			Where("id = ?", submissionID).
			Updates(map[string]interface{}{
				"reviewer_id": actorID,
				"updated_at":  now,
			}).Error; txErr != nil {
			return txErr
		}

		sid := submissionID
		act := actorID
		return tx.Create(&AntiplagHistory{
			ProjectID:    c.ProjectID,
			SubmissionID: &sid,
			Action:       "started_review",
			ActorID:      &act,
			CreatedAt:    now,
		}).Error
	})
	if err != nil {
		return nil, err
	}

	return s.repo.GetAntiplagCheck(ctx, submissionID)
}

func (s *Service) AntiplagSetScores(ctx context.Context, submissionID string, actorID int64, hasPlag bool, plag int32, hasAI bool, ai int32) (*AntiplagCheck, error) {
	if actorID <= 0 {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}
	if !hasPlag && !hasAI {
		return nil, status.Error(codes.InvalidArgument, "nothing to set: provide plagiarism_score and/or ai_score")
	}
	if hasPlag && (plag < 0 || plag > 100) {
		return nil, status.Error(codes.InvalidArgument, "plagiarism_score must be between 0 and 100")
	}
	if hasAI && (ai < 0 || ai > 100) {
		return nil, status.Error(codes.InvalidArgument, "ai_score must be between 0 and 100")
	}

	sub, err := s.assertAntiplagSubmission(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	if err = s.assertProjectInAntiplagiat(ctx, sub.ProjectID); err != nil {
		return nil, err
	}

	check, err := s.repo.GetAntiplagCheck(ctx, submissionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if _, e2 := s.repo.EnsureAntiplagCheckForSubmission(ctx, submissionID); e2 != nil {
				return nil, e2
			}
			check, err = s.repo.GetAntiplagCheck(ctx, submissionID)
		}
	}
	if err != nil {
		return nil, err
	}
	if check.Status != AntiplagStatusInReview {
		return nil, status.Error(codes.FailedPrecondition, "document is not in_review")
	}
	if check.CheckerID == nil || *check.CheckerID != actorID {
		return nil, status.Error(codes.PermissionDenied, "only assigned checker can set scores")
	}

	if hasPlag {
		v := plag
		check.PlagiarismScore = &v
	}
	if hasAI {
		v := ai
		check.AIScore = &v
	}
	if err = s.repo.UpdateAntiplagCheck(ctx, check); err != nil {
		return nil, err
	}

	sid := submissionID
	act := actorID
	_ = s.repo.AddAntiplagHistory(ctx, &AntiplagHistory{
		ProjectID:    check.ProjectID,
		SubmissionID: &sid,
		Action:       "scores_set",
		ActorID:      &act,
	})

	return check, nil
}

func (s *Service) AntiplagAddComment(ctx context.Context, submissionID string, actorID int64, text string, page *int32) (*AntiplagComment, error) {
	if actorID <= 0 {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}
	if text == "" {
		return nil, status.Error(codes.InvalidArgument, "text is required")
	}

	sub, err := s.assertAntiplagSubmission(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	if err = s.assertProjectInAntiplagiat(ctx, sub.ProjectID); err != nil {
		return nil, err
	}

	check, err := s.repo.GetAntiplagCheck(ctx, submissionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if _, e2 := s.repo.EnsureAntiplagCheckForSubmission(ctx, submissionID); e2 != nil {
				return nil, e2
			}
			check, err = s.repo.GetAntiplagCheck(ctx, submissionID)
		}
	}
	if err != nil {
		return nil, err
	}
	if check.Status != AntiplagStatusInReview {
		return nil, status.Error(codes.FailedPrecondition, "document is not in_review")
	}
	if check.CheckerID == nil || *check.CheckerID != actorID {
		return nil, status.Error(codes.PermissionDenied, "only assigned checker can add comments")
	}

	comment := &AntiplagComment{
		SubmissionID: submissionID,
		Text:         text,
		PageNumber:   page,
	}
	if err = s.repo.CreateAntiplagComment(ctx, comment); err != nil {
		return nil, err
	}

	sid := submissionID
	act := actorID
	_ = s.repo.AddAntiplagHistory(ctx, &AntiplagHistory{
		ProjectID:    check.ProjectID,
		SubmissionID: &sid,
		Action:       "comment_added",
		ActorID:      &act,
		Comment:      text,
	})

	return comment, nil
}

func (s *Service) AntiplagUpdateComment(ctx context.Context, commentID string, actorID int64, text string, page *int32) (*AntiplagComment, error) {
	if commentID == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if actorID <= 0 {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	comment, err := s.repo.GetAntiplagComment(ctx, commentID)
	if err != nil {
		return nil, err
	}

	check, err := s.repo.GetAntiplagCheck(ctx, comment.SubmissionID)
	if err != nil {
		return nil, err
	}
	if check.Status != AntiplagStatusInReview {
		return nil, status.Error(codes.FailedPrecondition, "document is not in_review")
	}
	if check.CheckerID == nil || *check.CheckerID != actorID {
		return nil, status.Error(codes.PermissionDenied, "only assigned checker can update comments")
	}

	if text != "" {
		comment.Text = text
	}
	comment.PageNumber = page
	if err = s.repo.UpdateAntiplagComment(ctx, comment); err != nil {
		return nil, err
	}
	return comment, nil
}

func (s *Service) AntiplagDeleteComment(ctx context.Context, commentID string, actorID int64) error {
	if commentID == "" {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	if actorID <= 0 {
		return status.Error(codes.Unauthenticated, "unauthorized")
	}

	comment, err := s.repo.GetAntiplagComment(ctx, commentID)
	if err != nil {
		return err
	}
	check, err := s.repo.GetAntiplagCheck(ctx, comment.SubmissionID)
	if err != nil {
		return err
	}
	if check.CheckerID == nil || *check.CheckerID != actorID {
		return status.Error(codes.PermissionDenied, "only assigned checker can delete comments")
	}
	return s.repo.DeleteAntiplagComment(ctx, commentID)
}

func (s *Service) AntiplagApprove(ctx context.Context, submissionID string, actorID int64, comment string) (*AntiplagCheck, error) {
	if submissionID == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if actorID <= 0 {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	sub, err := s.assertAntiplagSubmission(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	if err = s.assertProjectInAntiplagiat(ctx, sub.ProjectID); err != nil {
		return nil, err
	}

	r, ok := s.repo.(*repository)
	if !ok || r.db == nil {
		return nil, status.Error(codes.Internal, "repo misconfigured")
	}
	if _, err = s.repo.EnsureAntiplagCheckForSubmission(ctx, submissionID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	// 1) DB approve (без внешних вызовов внутри транзакции)
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var c AntiplagCheck
		if txErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&c, "submission_id = ?", submissionID).Error; txErr != nil {
			return txErr
		}
		if c.Status != AntiplagStatusInReview {
			return status.Error(codes.FailedPrecondition, "document is not in_review")
		}
		if c.CheckerID == nil || *c.CheckerID != actorID {
			return status.Error(codes.PermissionDenied, "only assigned checker can approve")
		}

		if c.PlagiarismScore == nil || c.AIScore == nil {
			return status.Error(codes.FailedPrecondition, "both plagiarism_score and ai_score must be set before approval")
		}
		if *c.PlagiarismScore < AntiplagDefaultMinPlagiarism {
			return status.Errorf(codes.FailedPrecondition,
				"originality %d%% is below minimum %d%%", *c.PlagiarismScore, AntiplagDefaultMinPlagiarism)
		}
		if *c.AIScore > AntiplagDefaultMaxAI {
			return status.Errorf(codes.FailedPrecondition,
				"AI content %d%% exceeds maximum %d%%", *c.AIScore, AntiplagDefaultMaxAI)
		}

		c.Status = AntiplagStatusApproved
		c.OverallComment = comment
		c.ReviewedAt = &now
		c.UpdatedAt = now
		if txErr := tx.Save(&c).Error; txErr != nil {
			return txErr
		}

		var ssub Submission
		if txErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&ssub, "id = ?", submissionID).Error; txErr != nil {
			return txErr
		}
		ssub.Status = StatusApproved
		ssub.ReviewerID = &actorID
		ssub.ReviewComment = comment
		ssub.ReviewedAt = &now
		ssub.UpdatedAt = now
		return tx.Save(&ssub).Error
	})
	if err != nil {
		return nil, err
	}

	check, err := s.repo.GetAntiplagCheck(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	// 2) Workflow transition от имени лидера команды (student)
	leaderID, teamID, err := s.antiplagTeamLeader(ctx, check.ProjectID, check.TeamID)
	if err != nil {
		s.rollbackAntiplagApprove(ctx, r, submissionID)
		return nil, err
	}

	wfErr := s.tryPerformIfAvailable(ctx, leaderID, "student", check.ProjectID, antiplagEventApprove, map[string]interface{}{
		"source":        "admin_service",
		"submission_id": submissionID,
	})
	if wfErr != nil {
		s.rollbackAntiplagApprove(ctx, r, submissionID)
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
	_ = s.repo.AddAntiplagHistory(ctx, &AntiplagHistory{
		ProjectID:    check.ProjectID,
		SubmissionID: &sid,
		Action:       "approved",
		ActorID:      &act,
		Comment:      comment,
	})

	// 4) Notify team
	s.notifyTeamBestEffort(ctx, teamID,
		"Антиплагиат пройден",
		"Проверка на антиплагиат завершена успешно.",
		"/diplom",
		"antiplagiat_approved",
	)

	return s.repo.GetAntiplagCheck(ctx, submissionID)
}

func (s *Service) AntiplagReturn(ctx context.Context, submissionID string, actorID int64, comment string) (*AntiplagCheck, error) {
	if submissionID == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if actorID <= 0 {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	sub, err := s.assertAntiplagSubmission(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	if err = s.assertProjectInAntiplagiat(ctx, sub.ProjectID); err != nil {
		return nil, err
	}

	r, ok := s.repo.(*repository)
	if !ok || r.db == nil {
		return nil, status.Error(codes.Internal, "repo misconfigured")
	}
	if _, err = s.repo.EnsureAntiplagCheckForSubmission(ctx, submissionID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var c AntiplagCheck
		if txErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&c, "submission_id = ?", submissionID).Error; txErr != nil {
			return txErr
		}
		if c.Status != AntiplagStatusInReview {
			return status.Error(codes.FailedPrecondition, "document is not in_review")
		}
		if c.CheckerID == nil || *c.CheckerID != actorID {
			return status.Error(codes.PermissionDenied, "only assigned checker can return")
		}

		c.Status = AntiplagStatusReturned
		c.OverallComment = comment
		c.ReviewedAt = &now
		c.UpdatedAt = now
		if txErr := tx.Save(&c).Error; txErr != nil {
			return txErr
		}

		return tx.Model(&Submission{}).
			Where("id = ?", submissionID).
			Updates(map[string]interface{}{
				"status":         StatusPending,
				"review_comment": comment,
				"reviewed_at":    now,
				"updated_at":     now,
			}).Error
	})
	if err != nil {
		return nil, err
	}

	check, err := s.repo.GetAntiplagCheck(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	leaderID, teamID, err := s.antiplagTeamLeader(ctx, check.ProjectID, check.TeamID)
	if err != nil {
		return nil, err
	}

	_ = s.tryPerformIfAvailable(ctx, leaderID, "student", check.ProjectID, antiplagEventReject, map[string]interface{}{
		"source":        "admin_service",
		"submission_id": submissionID,
	})

	sid := submissionID
	act := actorID
	_ = s.repo.AddAntiplagHistory(ctx, &AntiplagHistory{
		ProjectID:    check.ProjectID,
		SubmissionID: &sid,
		Action:       "returned",
		ActorID:      &act,
		Comment:      comment,
	})

	s.notifyTeamBestEffort(ctx, teamID,
		"Антиплагиат: возврат на доработку",
		"Документ возвращён на повторную загрузку.",
		"/diplom",
		"antiplagiat_returned",
	)

	return s.repo.GetAntiplagCheck(ctx, submissionID)
}

func (s *Service) AntiplagHistory(ctx context.Context, projectID int64) ([]*AntiplagHistory, error) {
	if projectID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}
	return s.repo.ListAntiplagHistory(ctx, projectID)
}

// ===== helpers (antiplagiat) =====

func (s *Service) rollbackAntiplagApprove(ctx context.Context, r *repository, submissionID string) {
	_ = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_ = tx.Model(&AntiplagCheck{}).
			Where("submission_id = ?", submissionID).
			Updates(map[string]interface{}{
				"status":          AntiplagStatusInReview,
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
}

func (s *Service) assertAntiplagSubmission(ctx context.Context, submissionID string) (*Submission, error) {
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
	if stateName != antiplagStateName {
		return nil, status.Error(codes.FailedPrecondition, "submission is not for ANTIPLAGIAT step")
	}

	return sub, nil
}

func (s *Service) assertProjectInAntiplagiat(ctx context.Context, projectID int64) error {
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
	if rt.CurrentStateName != antiplagStateName {
		return status.Error(codes.FailedPrecondition, "project is not in ANTIPLAGIAT state")
	}
	return nil
}

func (s *Service) antiplagTeamLeader(ctx context.Context, projectID int64, teamIDPtr *int64) (int64, int64, error) {
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
