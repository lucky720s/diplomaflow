package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.uber.org/zap"
)

var (
	ErrReviewNotAllowed    = errors.New("reviewer role not allowed for this state")
	ErrReviewAlreadyExists = errors.New("reviewer already submitted a review for this state")
	ErrInvalidScore        = errors.New("score must be between 0 and 100")
	ErrInvalidDecision     = errors.New("decision must be 'approved' or 'rejected'")
	ErrNotAllVoted         = errors.New("not all required reviewers have voted yet")
)

// SubmitReviewRequest — запрос на выставление оценки/допуска
type SubmitReviewRequest struct {
	ProjectID  int64
	StateID    int64
	ReviewerID int64
	RoleSlug   string
	// CallerRoles are the caller's department roles (from JWT DeptRoles). The
	// review is allowed if RoleSlug OR any CallerRole is in reviewer_roles, so a
	// `teacher` carrying the `commission` department role can review a
	// commission-only state. The recorded role is the matched reviewer role.
	CallerRoles []string
	// Для grade_type = "admission"
	Decision string // "approved" | "rejected"
	// Для grade_type = "score"
	Score   *int32
	Comment string
}

// SubmitReviewResponse — результат после выставления оценки
type SubmitReviewResponse struct {
	AllVoted        bool
	ReadyToFinalize bool
	Summary         ReviewSummary
}

// ReviewService — сервис управления голосами преподов
type ReviewService struct {
	reviewRepo ReviewRepository
	repo       Repository
	logger     *zap.Logger
}

func NewReviewService(reviewRepo ReviewRepository, repo Repository, logger *zap.Logger) *ReviewService {
	return &ReviewService{
		reviewRepo: reviewRepo,
		repo:       repo,
		logger:     logger,
	}
}

// SubmitReview — препод выставляет оценку/допуск.
// Проверяет что роль разрешена, тип данных соответствует grade_type стэйта.
func (s *ReviewService) SubmitReview(ctx context.Context, req *SubmitReviewRequest) (*SubmitReviewResponse, error) {
	state, err := s.repo.GetState(ctx, req.StateID)
	if err != nil {
		return nil, fmt.Errorf("get state: %w", err)
	}

	rc, err := s.parseReviewConfig(state)
	if err != nil {
		return nil, fmt.Errorf("parse review_config: %w", err)
	}
	if rc == nil {
		return nil, errors.New("state has no review_config")
	}

	// Проверяем что роль разрешена. Кандидаты: базовая роль + департаментские
	// роли из JWT. Записываем именно совпавшую reviewer-роль (например
	// "commission"), а не базовую "teacher".
	matchedRole, ok := s.matchReviewerRole(req.RoleSlug, req.CallerRoles, rc.ReviewerRoles)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrReviewNotAllowed, req.RoleSlug)
	}

	// Валидируем payload по grade_type
	review := &StateReview{
		ProjectID:  req.ProjectID,
		StateID:    req.StateID,
		ReviewerID: req.ReviewerID,
		RoleSlug:   matchedRole,
		Comment:    req.Comment,
	}

	switch rc.GradeType {
	case "score":
		if req.Score == nil {
			return nil, errors.New("score is required for score-type state")
		}
		if *req.Score < 0 || *req.Score > 100 {
			return nil, ErrInvalidScore
		}
		review.Score = req.Score

	case "admission":
		if req.Decision != "approved" && req.Decision != "rejected" {
			return nil, ErrInvalidDecision
		}
		review.Decision = req.Decision

	default:
		return nil, fmt.Errorf("unknown grade_type: %q", rc.GradeType)
	}

	if err = s.reviewRepo.UpsertReview(ctx, review); err != nil {
		return nil, fmt.Errorf("upsert review: %w", err)
	}

	summary, err := s.ComputeSummary(ctx, req.ProjectID, req.StateID, rc)
	if err != nil {
		return nil, fmt.Errorf("compute summary: %w", err)
	}

	s.logger.Info("review submitted",
		zap.Int64("project_id", req.ProjectID),
		zap.Int64("state_id", req.StateID),
		zap.Int64("reviewer_id", req.ReviewerID),
		zap.String("grade_type", rc.GradeType),
		zap.Bool("all_voted", summary.AllVoted),
	)

	return &SubmitReviewResponse{
		AllVoted:        summary.AllVoted,
		ReadyToFinalize: summary.AllVoted,
		Summary:         *summary,
	}, nil
}

// ComputeSummary — считает итог голосования: AllVoted, среднее/majority, результат.
func (s *ReviewService) ComputeSummary(ctx context.Context, projectID, stateID int64, rc *ReviewConfig) (*ReviewSummary, error) {
	reviews, err := s.reviewRepo.GetReviews(ctx, projectID, stateID)
	if err != nil {
		return nil, err
	}

	summary := &ReviewSummary{
		TotalReviews: len(reviews),
		Result:       "pending",
	}

	// Определяем нужное количество голосов
	required := int(rc.MinReviewers)
	if required <= 0 {
		// 0 означает "все из reviewer_roles" — берём количество ролей как минимум 1 голос на роль
		required = len(rc.ReviewerRoles)
	}

	switch rc.GradeType {
	case "score":
		var total int32
		var count int
		for _, r := range reviews {
			if r.Score != nil {
				total += *r.Score
				count++
			}
		}
		summary.AllVoted = count >= required
		if count > 0 {
			avg := float64(total) / float64(count)
			summary.AverageScore = avg
			summary.FinalScore = &avg
			if summary.AllVoted {
				if avg >= float64(rc.PassingScore) {
					summary.Result = "passed"
				} else {
					summary.Result = "failed"
				}
			}
		}

	case "admission":
		for _, r := range reviews {
			if r.Decision == "approved" {
				summary.ApprovedCount++
			} else if r.Decision == "rejected" {
				summary.RejectedCount++
			}
		}
		voted := summary.ApprovedCount + summary.RejectedCount
		summary.AllVoted = voted >= required
		if summary.AllVoted {
			if summary.ApprovedCount >= summary.RejectedCount {
				summary.Result = "approved"
			} else {
				summary.Result = "rejected"
			}
		}
	}

	return summary, nil
}

// GetSummary — получить текущий итог по стэйту (без изменений)
func (s *ReviewService) GetSummary(ctx context.Context, projectID, stateID int64) (*ReviewSummary, error) {
	state, err := s.repo.GetState(ctx, stateID)
	if err != nil {
		return nil, err
	}
	rc, err := s.parseReviewConfig(state)
	if err != nil || rc == nil {
		return &ReviewSummary{Result: "pending"}, nil
	}
	return s.ComputeSummary(ctx, projectID, stateID, rc)
}

// GetReviews — список голосов по стэйту
func (s *ReviewService) GetReviews(ctx context.Context, projectID, stateID int64) ([]StateReview, error) {
	return s.reviewRepo.GetReviews(ctx, projectID, stateID)
}

// DeleteReviewsForState — очищает голоса при откате стэйта
func (s *ReviewService) DeleteReviewsForState(ctx context.Context, projectID, stateID int64) error {
	return s.reviewRepo.DeleteReviews(ctx, projectID, stateID)
}

func (s *ReviewService) parseReviewConfig(state *State) (*ReviewConfig, error) {
	var sc StateConfig
	if err := json.Unmarshal(state.Config, &sc); err != nil {
		return nil, err
	}
	return sc.ReviewConfig, nil
}

func (s *ReviewService) roleAllowed(roleSlug string, allowed []string) bool {
	for _, r := range allowed {
		if r == roleSlug {
			return true
		}
	}
	return false
}

// matchReviewerRole returns the first of the caller's roles (base RoleSlug plus
// department roles) that appears in the allowed reviewer_roles set. The matched
// role is what gets recorded on the review, so audit reflects the capability
// actually exercised (e.g. "commission") rather than the base role.
func (s *ReviewService) matchReviewerRole(roleSlug string, callerRoles, allowed []string) (string, bool) {
	if s.roleAllowed(roleSlug, allowed) {
		return roleSlug, true
	}
	for _, r := range callerRoles {
		if s.roleAllowed(r, allowed) {
			return r, true
		}
	}
	return "", false
}
