package task

import (
	"context"
	"errors"

	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

var (
	ErrAccessDenied    = errors.New("access denied")
	ErrBoardNotFound   = errors.New("board not found")
	ErrTaskNotFound    = errors.New("task not found")
	ErrColumnNotFound  = errors.New("column not found")
	ErrNotTeamMember   = errors.New("user is not a team member")
	ErrCrossUniversity = errors.New("cross-university access denied")
	ErrCrossDepartment = errors.New("cross-department access denied")
)

type AccessChecker struct {
	repo       Repository // ✅ интерфейс — Wire знает провайдера NewRepository
	teamClient teamv1.TeamServiceClient
	logger     *zap.Logger
}

func NewAccessChecker(
	repo Repository, // ✅ интерфейс вместо *repository
	teamClient teamv1.TeamServiceClient,
	logger *zap.Logger,
) *AccessChecker {
	return &AccessChecker{
		repo:       repo,
		teamClient: teamClient,
		logger:     logger,
	}
}

func (ac *AccessChecker) CheckBoardAccess(ctx context.Context, boardID int64, auth AuthContext) error {
	if IsInternalCall(auth) {
		return nil
	}
	board, err := ac.repo.GetBoard(ctx, boardID)
	if err != nil {
		ac.logger.Debug("board not found",
			zap.Int64("board_id", boardID),
			zap.Error(err),
		)
		return ErrBoardNotFound
	}
	return ac.checkTeamAccess(ctx, board.TeamID, auth)
}

func (ac *AccessChecker) CheckBoardAccessByProject(ctx context.Context, projectID int64, auth AuthContext) error {
	if IsInternalCall(auth) {
		return nil
	}
	board, err := ac.repo.GetBoardByProject(ctx, projectID)
	if err != nil {
		return ErrBoardNotFound
	}
	return ac.checkTeamAccess(ctx, board.TeamID, auth)
}

func (ac *AccessChecker) CheckBoardAccessByTeamID(ctx context.Context, teamID int64, auth AuthContext) error {
	if IsInternalCall(auth) {
		return nil
	}
	return ac.checkTeamAccess(ctx, teamID, auth)
}

func (ac *AccessChecker) CheckTaskAccess(ctx context.Context, taskID int64, auth AuthContext) error {
	if IsInternalCall(auth) {
		return nil
	}
	task, err := ac.repo.GetTask(ctx, taskID)
	if err != nil {
		return ErrTaskNotFound
	}
	return ac.CheckBoardAccess(ctx, task.BoardID, auth)
}

func (ac *AccessChecker) CheckColumnAccess(ctx context.Context, columnID int64, auth AuthContext) error {
	if IsInternalCall(auth) {
		return nil
	}
	column, err := ac.repo.GetColumn(ctx, columnID)
	if err != nil {
		return ErrColumnNotFound
	}
	return ac.CheckBoardAccess(ctx, column.BoardID, auth)
}

func (ac *AccessChecker) checkTeamAccess(ctx context.Context, teamID int64, auth AuthContext) error {
	teamInfo, err := ac.getTeamInfo(ctx, teamID)
	if err != nil {
		ac.logger.Error("failed to get team info",
			zap.Int64("team_id", teamID),
			zap.Error(err),
		)
		return ErrAccessDenied
	}

	switch auth.Role {
	case "student":
		return ac.checkStudentAccess(ctx, teamID, auth.UserID)
	case "teacher":
		return ac.checkTeacherAccess(ctx, teamID, teamInfo, auth)
	case "admin":
		return ac.checkAdminAccess(teamInfo, auth)
	default:
		ac.logger.Warn("unknown role",
			zap.String("role", auth.Role),
			zap.Int64("user_id", auth.UserID),
		)
		return ErrAccessDenied
	}
}

func (ac *AccessChecker) checkStudentAccess(ctx context.Context, teamID, userID int64) error {
	isMember, err := ac.checkMembership(ctx, teamID, userID)
	if err != nil {
		return ErrAccessDenied
	}
	if !isMember {
		return ErrNotTeamMember
	}
	return nil
}

func (ac *AccessChecker) checkTeacherAccess(
	ctx context.Context,
	teamID int64,
	teamInfo *teamv1.GetTeamInternalResponse,
	auth AuthContext,
) error {
	if teamInfo.SupervisorId != nil && *teamInfo.SupervisorId == auth.UserID {
		return nil
	}
	isMember, _ := ac.checkMembership(ctx, teamID, auth.UserID)
	if isMember {
		return nil
	}
	if teamInfo.UniversityId != auth.UniversityID {
		return ErrCrossUniversity
	}
	if teamInfo.DepartmentId != auth.DepartmentID {
		return ErrCrossDepartment
	}
	return nil
}

func (ac *AccessChecker) checkAdminAccess(
	teamInfo *teamv1.GetTeamInternalResponse,
	auth AuthContext,
) error {
	if teamInfo.UniversityId != auth.UniversityID {
		return ErrCrossUniversity
	}
	return nil
}

func (ac *AccessChecker) getTeamInfo(ctx context.Context, teamID int64) (*teamv1.GetTeamInternalResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, "x-internal-service", "task_service")
	resp, err := ac.teamClient.GetTeamInternal(ctx, &teamv1.GetTeamInternalRequest{
		TeamId: teamID,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (ac *AccessChecker) checkMembership(ctx context.Context, teamID, userID int64) (bool, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, "x-internal-service", "task_service")
	resp, err := ac.teamClient.IsMember(ctx, &teamv1.IsMemberRequest{
		TeamId: teamID,
		UserId: userID,
	})
	if err != nil {
		ac.logger.Error("IsMember gRPC call failed",
			zap.Int64("team_id", teamID),
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return false, err
	}
	return resp.IsMember, nil
}

func (ac *AccessChecker) CanModifyTask(ctx context.Context, task *Task, auth AuthContext) error {
	if IsInternalCall(auth) {
		return nil
	}
	if err := ac.CheckBoardAccess(ctx, task.BoardID, auth); err != nil {
		return err
	}
	if task.CreatedBy == auth.UserID {
		return nil
	}
	if task.AssigneeID != nil && *task.AssigneeID == auth.UserID {
		return nil
	}
	if auth.Role == "teacher" || auth.Role == "admin" {
		return nil
	}
	return ErrAccessDenied
}

func (ac *AccessChecker) CanDeleteTask(ctx context.Context, task *Task, auth AuthContext) error {
	if IsInternalCall(auth) {
		return nil
	}
	if err := ac.CheckBoardAccess(ctx, task.BoardID, auth); err != nil {
		return err
	}
	if task.CreatedBy == auth.UserID {
		return nil
	}
	if auth.Role == "teacher" || auth.Role == "admin" {
		return nil
	}
	return ErrAccessDenied
}

func (ac *AccessChecker) CanModifyColumn(ctx context.Context, columnID int64, auth AuthContext) error {
	if IsInternalCall(auth) {
		return nil
	}
	return ac.CheckColumnAccess(ctx, columnID, auth)
}
