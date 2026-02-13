package team

import (
	"context"
	"strconv"

	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Handler struct {
	teamv1.UnimplementedTeamServiceServer
	service *Service
	logger  *zap.Logger
}

func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// ---- metadata helpers ----

func getRequesterID(ctx context.Context) int64 {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-user-id"); len(vals) > 0 {
			id, _ := strconv.ParseInt(vals[0], 10, 64)
			return id
		}
	}
	return 0
}

func getUserRole(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-user-role"); len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

func getUniversityID(ctx context.Context) int64 {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-university-id"); len(vals) > 0 {
			id, _ := strconv.ParseInt(vals[0], 10, 64)
			return id
		}
	}
	return 0
}

func getDepartmentID(ctx context.Context) int64 {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-department-id"); len(vals) > 0 {
			id, _ := strconv.ParseInt(vals[0], 10, 64)
			return id
		}
	}
	return 0
}

func getInternalService(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-internal-service"); len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

func requireAuth(ctx context.Context) (userID int64, role string, univID int64, deptID int64, err error) {
	userID = getRequesterID(ctx)
	if userID == 0 {
		return 0, "", 0, 0, status.Error(codes.Unauthenticated, "unauthorized")
	}
	role = getUserRole(ctx)
	univID = getUniversityID(ctx)
	deptID = getDepartmentID(ctx)
	if univID == 0 || deptID == 0 {
		return 0, "", 0, 0, status.Error(codes.InvalidArgument, "university_id and department_id are required in metadata")
	}
	return userID, role, univID, deptID, nil
}

// ---- RPCs ----

func (h *Handler) CreateTeam(ctx context.Context, req *teamv1.CreateTeamRequest) (*teamv1.CreateTeamResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	userID, role, univID, deptID, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	// Обычно команды создают студенты; admin тоже можно (если нужно) — оставим.
	if role == "" {
		return nil, status.Error(codes.PermissionDenied, "forbidden")
	}

	// НЕ доверяем req.LeaderId: лидер = requester
	teamID, err := h.service.CreateTeam(ctx, req.Name, userID, req.MemberIds, deptID, univID)
	if err != nil {
		h.logger.Error("CreateTeam failed", zap.Error(err))
		return nil, mapError(err)
	}
	return &teamv1.CreateTeamResponse{TeamId: teamID}, nil
}

func (h *Handler) GetTeam(ctx context.Context, req *teamv1.GetTeamRequest) (*teamv1.GetTeamResponse, error) {
	if req.TeamId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id is required")
	}
	team, members, err := h.service.GetTeam(ctx, req.TeamId)
	if err != nil {
		return nil, mapError(err)
	}

	pbMembers := make([]*teamv1.TeamMember, 0, len(members))
	for _, m := range members {
		pbMembers = append(pbMembers, &teamv1.TeamMember{
			UserId: m.UserID,
			Role:   m.Role,
		})
	}

	return &teamv1.GetTeamResponse{
		TeamId:  team.ID,
		Name:    team.Name,
		Members: pbMembers,
	}, nil
}

func (h *Handler) GetMyTeam(ctx context.Context, req *teamv1.GetMyTeamRequest) (*teamv1.GetMyTeamResponse, error) {
	userID, _, _, _, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	// игнорируем req.UserId (или можно строго запрещать mismatch)
	if req.UserId != 0 && req.UserId != userID {
		return nil, status.Error(codes.PermissionDenied, "forbidden")
	}

	team, membership, members, pendingCount, err := h.service.GetMyTeam(ctx, userID)
	if err != nil {
		return nil, mapError(err)
	}
	if team == nil {
		return &teamv1.GetMyTeamResponse{HasTeam: false}, nil
	}

	// collect ids
	ids := make([]int64, 0, len(members))
	seen := map[int64]struct{}{}
	for _, m := range members {
		if _, ok := seen[m.UserID]; ok {
			continue
		}
		seen[m.UserID] = struct{}{}
		ids = append(ids, m.UserID)
	}

	// internal call to auth_service
	authCtx := metadata.AppendToOutgoingContext(ctx, "x-internal-service", "team_service")
	au, err := h.service.authClient.BatchGetUserPreviews(authCtx, &authv1.BatchGetUserPreviewsRequest{Ids: ids})
	if err != nil {
		h.logger.Warn("BatchGetUserPreviews failed", zap.Error(err))
	}

	users := map[int64]*authv1.UserPreview{}
	if au != nil {
		for _, u := range au.Users {
			users[u.Id] = u
		}
	}

	pbMembers := make([]*teamv1.TeamMember, 0, len(members))
	for _, m := range members {
		tm := &teamv1.TeamMember{UserId: m.UserID, Role: m.Role}
		if u := users[m.UserID]; u != nil {
			tm.FirstName = u.FirstName
			tm.LastName = u.LastName
			tm.Email = u.Email
		}
		pbMembers = append(pbMembers, tm)
	}

	return &teamv1.GetMyTeamResponse{
		HasTeam: true,
		Team: &teamv1.TeamInfo{
			TeamId:              team.ID,
			Name:                team.Name,
			Role:                membership.Role,
			Members:             pbMembers,
			MemberCount:         int32(len(members)),
			PendingInvitesCount: int32(pendingCount),
			InviteCode:          team.InviteCode,
			CompositionLocked:   team.CompositionLocked,
		},
	}, nil
}

func (h *Handler) UpdateTeam(ctx context.Context, req *teamv1.UpdateTeamRequest) (*teamv1.UpdateTeamResponse, error) {
	if req.Team == nil {
		return nil, status.Error(codes.InvalidArgument, "team is required")
	}
	userID, _, _, _, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}

	team, err := h.service.UpdateTeam(ctx, req.Team.Id, req.Team.Name, userID)
	if err != nil {
		h.logger.Error("UpdateTeam failed", zap.Error(err))
		return nil, mapError(err)
	}

	members, _ := h.service.repo.GetMembers(ctx, team.ID)
	pbMembers := make([]*teamv1.TeamMember, 0, len(members))
	for _, m := range members {
		pbMembers = append(pbMembers, &teamv1.TeamMember{UserId: m.UserID, Role: m.Role})
	}

	return &teamv1.UpdateTeamResponse{
		Team: &teamv1.Team{
			Id:        team.ID,
			Name:      team.Name,
			Members:   pbMembers,
			CreatedAt: team.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		},
	}, nil
}

func (h *Handler) DeleteTeam(ctx context.Context, req *teamv1.DeleteTeamRequest) (*emptypb.Empty, error) {
	userID, _, _, _, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if req.TeamId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id is required")
	}

	if err := h.service.DeleteTeam(ctx, req.TeamId, userID); err != nil {
		h.logger.Error("DeleteTeam failed", zap.Error(err))
		return nil, mapError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) LeaveTeam(ctx context.Context, req *teamv1.LeaveTeamRequest) (*teamv1.LeaveTeamResponse, error) {
	userID, _, _, _, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if req.TeamId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id is required")
	}
	// user_id only self
	if req.UserId != 0 && req.UserId != userID {
		return nil, status.Error(codes.PermissionDenied, "forbidden")
	}

	result, err := h.service.LeaveTeam(ctx, req.TeamId, userID)
	if err != nil {
		h.logger.Error("LeaveTeam failed", zap.Error(err))
		return nil, mapError(err)
	}

	return &teamv1.LeaveTeamResponse{
		Success:     result.Success,
		Message:     result.Message,
		TeamDeleted: result.TeamDeleted,
		NewLeaderId: result.NewLeaderID,
	}, nil
}

func (h *Handler) TransferLeadership(ctx context.Context, req *teamv1.TransferLeadershipRequest) (*teamv1.TransferLeadershipResponse, error) {
	userID, _, _, _, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if req.TeamId <= 0 || req.NewLeaderId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id and new_leader_id are required")
	}
	// current leader must be requester
	if req.CurrentLeaderId != 0 && req.CurrentLeaderId != userID {
		return nil, status.Error(codes.PermissionDenied, "forbidden")
	}

	team, members, err := h.service.TransferLeadership(ctx, req.TeamId, userID, req.NewLeaderId)
	if err != nil {
		h.logger.Error("TransferLeadership failed", zap.Error(err))
		return nil, mapError(err)
	}

	pbMembers := make([]*teamv1.TeamMember, 0, len(members))
	for _, m := range members {
		pbMembers = append(pbMembers, &teamv1.TeamMember{UserId: m.UserID, Role: m.Role})
	}

	return &teamv1.TransferLeadershipResponse{
		Success: true,
		Message: "Лидерство успешно передано",
		UpdatedTeam: &teamv1.TeamInfo{
			TeamId:      team.ID,
			Name:        team.Name,
			Members:     pbMembers,
			MemberCount: int32(len(members)),
		},
	}, nil
}

func (h *Handler) AddMember(ctx context.Context, req *teamv1.AddMemberRequest) (*teamv1.AddMemberResponse, error) {
	userID, _, _, _, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if req.TeamId <= 0 || req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id and user_id are required")
	}

	if err := h.service.AddMember(ctx, req.TeamId, req.UserId, req.Role, userID); err != nil {
		return nil, mapError(err)
	}
	return &teamv1.AddMemberResponse{Success: true, Message: "Member added successfully"}, nil
}

func (h *Handler) RemoveMember(ctx context.Context, req *teamv1.RemoveMemberRequest) (*emptypb.Empty, error) {
	userID, _, _, _, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if req.TeamId <= 0 || req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id and user_id are required")
	}

	if err := h.service.RemoveMember(ctx, req.TeamId, req.UserId, userID); err != nil {
		return nil, mapError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) GetMyInvites(ctx context.Context, req *teamv1.GetMyInvitesRequest) (*teamv1.GetMyInvitesResponse, error) {
	userID, _, _, _, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if req.UserId != 0 && req.UserId != userID {
		return nil, status.Error(codes.PermissionDenied, "forbidden")
	}

	invites, err := h.service.GetMyInvites(ctx, userID)
	if err != nil {
		return nil, mapError(err)
	}

	pbInvites := make([]*teamv1.Invite, 0, len(invites))
	for _, inv := range invites {
		team, _ := h.service.repo.GetByID(ctx, inv.TeamID)
		teamName := ""
		if team != nil {
			teamName = team.Name
		}
		pbInvites = append(pbInvites, &teamv1.Invite{
			Id:        inv.ID,
			TeamId:    inv.TeamID,
			TeamName:  teamName,
			InviterId: inv.InviterID,
			Status:    inv.Status,
		})
	}

	return &teamv1.GetMyInvitesResponse{Invites: pbInvites}, nil
}

func (h *Handler) RespondToInvite(ctx context.Context, req *teamv1.RespondToInviteRequest) (*teamv1.RespondToInviteResponse, error) {
	userID, _, _, _, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if req.InviteId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invite_id is required")
	}
	if req.UserId != 0 && req.UserId != userID {
		return nil, status.Error(codes.PermissionDenied, "forbidden")
	}

	if err := h.service.RespondToInvite(ctx, req.InviteId, userID, req.Accept); err != nil {
		return nil, mapError(err)
	}

	msg := "Приглашение отклонено"
	if req.Accept {
		msg = "Вы присоединились к команде"
	}
	return &teamv1.RespondToInviteResponse{Success: true, Message: msg}, nil
}

func (h *Handler) GetAvailableStudents(ctx context.Context, req *teamv1.GetAvailableStudentsRequest) (*teamv1.GetAvailableStudentsResponse, error) {
	userID, role, univID, deptID, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	_ = userID

	// По логике платформы — это для студентов/лидера команды.
	// Если хочешь разрешить admin — добавь role == "admin".
	if role != "student" && role != "admin" {
		return nil, status.Error(codes.PermissionDenied, "forbidden")
	}

	// НЕ доверяем req.UniversityId/DepartmentId: берём из metadata
	users, err := h.service.GetAvailableStudents(ctx, univID, deptID, req.ExcludeUserId)
	if err != nil {
		return nil, mapError(err)
	}

	students := make([]*teamv1.StudentPreview, 0, len(users))
	for _, u := range users {
		students = append(students, &teamv1.StudentPreview{
			Id:       u.Id,
			FullName: u.FirstName + " " + u.LastName,
			Email:    u.Email,
		})
	}

	return &teamv1.GetAvailableStudentsResponse{Students: students}, nil
}

func (h *Handler) ListTeams(ctx context.Context, req *teamv1.ListTeamsRequest) (*teamv1.ListTeamsResponse, error) {
	_, role, _, deptID, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	// чтобы студент не сканировал все команды: по умолчанию ограничиваем своим dept
	departmentID := req.DepartmentId
	if role != "admin" {
		departmentID = deptID
	}

	teams, total, err := h.service.ListTeams(ctx, departmentID, req.Page, req.PageSize)
	if err != nil {
		return nil, mapError(err)
	}

	pbTeams := make([]*teamv1.Team, 0, len(teams))
	for _, t := range teams {
		members, _ := h.service.repo.GetMembers(ctx, t.ID)
		pbMembers := make([]*teamv1.TeamMember, 0, len(members))
		for _, m := range members {
			pbMembers = append(pbMembers, &teamv1.TeamMember{UserId: m.UserID, Role: m.Role})
		}
		pbTeams = append(pbTeams, &teamv1.Team{
			Id:        t.ID,
			Name:      t.Name,
			Members:   pbMembers,
			CreatedAt: t.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}

	return &teamv1.ListTeamsResponse{Teams: pbTeams, TotalCount: total}, nil
}

func (h *Handler) JoinTeamByCode(ctx context.Context, req *teamv1.JoinTeamByCodeRequest) (*teamv1.JoinTeamByCodeResponse, error) {
	userID, role, univID, deptID, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if req.InviteCode == "" {
		return nil, status.Error(codes.InvalidArgument, "invite_code is required")
	}

	teamID, err := h.service.JoinTeamByCode(ctx, req.InviteCode, userID, univID, deptID, role)
	if err != nil {
		return nil, mapError(err)
	}
	return &teamv1.JoinTeamByCodeResponse{Success: true, Message: "joined", TeamId: teamID}, nil
}

func (h *Handler) RegenerateInviteCode(ctx context.Context, req *teamv1.RegenerateInviteCodeRequest) (*teamv1.RegenerateInviteCodeResponse, error) {
	userID, _, _, _, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if req.TeamId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id is required")
	}

	code, err := h.service.RegenerateInviteCode(ctx, req.TeamId, userID)
	if err != nil {
		return nil, mapError(err)
	}
	return &teamv1.RegenerateInviteCodeResponse{InviteCode: code}, nil
}

func (h *Handler) LockTeamComposition(ctx context.Context, req *teamv1.LockTeamCompositionRequest) (*teamv1.LockTeamCompositionResponse, error) {
	// internal-only allow-list
	switch getInternalService(ctx) {
	case "admin_service", "workflow_service":
		// ok
	default:
		return nil, status.Error(codes.PermissionDenied, "forbidden")
	}

	if req.TeamId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id is required")
	}

	if err := h.service.LockTeamComposition(ctx, req.TeamId); err != nil {
		return nil, mapError(err)
	}
	return &teamv1.LockTeamCompositionResponse{Success: true}, nil
}

// ---- error mapping ----

func mapError(err error) error {
	switch err {
	case ErrTeamNotFound:
		return status.Error(codes.NotFound, "team not found")
	case ErrMemberNotFound:
		return status.Error(codes.NotFound, "member not found")
	case ErrNotTeamMember:
		return status.Error(codes.PermissionDenied, "user is not a team member")
	case ErrNotLeader:
		return status.Error(codes.PermissionDenied, "only team leader can perform this action")
	case ErrAlreadyInTeam:
		return status.Error(codes.AlreadyExists, "user is already in a team")
	case ErrCannotRemoveLeader:
		return status.Error(codes.FailedPrecondition, "cannot remove team leader")
	case ErrNewLeaderNotInTeam:
		return status.Error(codes.InvalidArgument, "new leader must be a team member")
	case ErrUnauthorized:
		return status.Error(codes.PermissionDenied, "unauthorized")
	case ErrInvalidInviteCode:
		return status.Error(codes.InvalidArgument, "invalid invite code")
	case ErrTeamCompositionLocked:
		return status.Error(codes.FailedPrecondition, "team composition is locked")
	case ErrTeamFull:
		return status.Error(codes.FailedPrecondition, "team is full")
	case ErrForbiddenDepartment, ErrForbiddenRole:
		return status.Error(codes.PermissionDenied, "forbidden")
	default:
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}
}
