package auth

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type AuthService interface {
	Register(ctx context.Context, email, password, firstName, lastName, role string, universityID, departmentID int64) (int64, error)
	Login(ctx context.Context, email, password, userAgent, ip string) (string, string, error)
	Validate(ctx context.Context, token string) (*JwtClaims, error)
	ListUsers(ctx context.Context, universityID int64, departmentID int64, role string, page, pageSize int32, excludeUserID int64) ([]*User, int64, error)
	AssignRole(ctx context.Context, userID int64, role string) error
	BatchGetUserPreviews(ctx context.Context, ids []int64) ([]*User, error)
	CreateDepartmentRole(ctx context.Context, departmentID int64, slug string) (*DepartmentRole, error)
	ListDepartmentRoles(ctx context.Context, departmentID int64) ([]*DepartmentRole, error)
	GetDepartmentRole(ctx context.Context, roleID int64) (*DepartmentRole, error)
	DeleteDepartmentRole(ctx context.Context, roleID int64) error
	AssignDepartmentRole(ctx context.Context, userID, departmentID, roleID int64, assignedBy int64, comment string) error
	RevokeDepartmentRole(ctx context.Context, userID, departmentID, roleID int64, revokedBy int64, comment string) error
	ListUserDepartmentRoleSlugs(ctx context.Context, userID, departmentID int64) ([]string, error)
	GetUser(ctx context.Context, userID int64) (*User, error)
	UpdateUser(ctx context.Context, userID int64, email, firstName, lastName, role string, universityID, departmentID int64, isActive *bool, requesterRole string) (*User, error)
	DeleteUser(ctx context.Context, userID, requesterID int64) error
	RefreshToken(ctx context.Context, clientToken, userAgent, ip string) (string, string, error)
	ListSessions(ctx context.Context, userID int64) ([]*RefreshToken, error)
	RevokeSession(ctx context.Context, userID int64, sessionID uint64) error
}

type Handler struct {
	authv1.UnimplementedAuthServiceServer
	service AuthService
}

func NewHandler(service AuthService) *Handler { return &Handler{service: service} }

func getClientInfo(ctx context.Context) (string, string) {
	ip := ""
	userAgent := ""
	if p, ok := peer.FromContext(ctx); ok {
		ip = p.Addr.String()
	}
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if ua := md.Get("user-agent"); len(ua) > 0 {
			userAgent = ua[0]
		}
	}
	return ip, userAgent
}
func pickClientInfo(ctx context.Context, reqIP, reqUA string) (ip string, ua string) {
	ip = strings.TrimSpace(reqIP)
	ua = strings.TrimSpace(reqUA)

	if ip != "" && ua != "" {
		return ip, ua
	}

	fallbackIP, fallbackUA := getClientInfo(ctx)
	if ip == "" {
		ip = fallbackIP
	}
	if ua == "" {
		ua = fallbackUA
	}
	return ip, ua
}

func requireInternal(ctx context.Context, expected string) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || md == nil {
		return status.Error(codes.PermissionDenied, "missing metadata")
	}
	v := md.Get("x-internal-service")
	if len(v) == 0 || v[0] != expected {
		return status.Error(codes.PermissionDenied, "forbidden")
	}
	return nil
}

func requireAdmin(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || md == nil {
		return status.Error(codes.PermissionDenied, "missing metadata")
	}
	v := md.Get("x-user-role")
	if len(v) == 0 || v[0] != "admin" {
		return status.Error(codes.PermissionDenied, "admin only")
	}
	return nil
}

func mdInt64(md metadata.MD, key string) int64 {
	if md == nil {
		return 0
	}
	v := md.Get(key)
	if len(v) == 0 {
		return 0
	}
	n, _ := strconv.ParseInt(v[0], 10, 64)
	return n
}

func requireInternalOneOf(ctx context.Context, expected ...string) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || md == nil {
		return status.Error(codes.PermissionDenied, "missing metadata")
	}
	v := md.Get("x-internal-service")
	if len(v) == 0 {
		return status.Error(codes.PermissionDenied, "forbidden")
	}
	got := v[0]
	for _, e := range expected {
		if got == e {
			return nil
		}
	}
	return status.Error(codes.PermissionDenied, "forbidden")
}

func (h *Handler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
	}

	userID, err := h.service.Register(
		ctx,
		req.Email,
		req.Password,
		req.FirstName,
		req.LastName,
		req.Role,
		req.UniversityId,
		req.DepartmentId,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register: %v", err)
	}

	return &authv1.RegisterResponse{UserId: userID}, nil
}

func (h *Handler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	if req == nil || strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	ip, userAgent := pickClientInfo(ctx, req.IpAddress, req.UserAgent)

	accessToken, refreshToken, err := h.service.Login(ctx, req.Email, req.Password, userAgent, ip)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	return &authv1.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (h *Handler) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	if req.Token == "" {
		return &authv1.ValidateTokenResponse{
			Status: http.StatusBadRequest,
			Error:  "token is required",
		}, nil
	}

	claims, err := h.service.Validate(ctx, req.Token)
	if err != nil {
		return &authv1.ValidateTokenResponse{
			Status: http.StatusUnauthorized,
			Error:  err.Error(),
		}, nil
	}

	return &authv1.ValidateTokenResponse{
		Status:       http.StatusOK,
		UserId:       claims.Id,
		Role:         claims.Role,
		UniversityId: claims.UniversityID,
		DepartmentId: claims.DepartmentID,
	}, nil
}

func (h *Handler) ListUsers(ctx context.Context, req *authv1.ListUsersRequest) (*authv1.ListUsersResponse, error) {
	if err := requireInternalOneOf(ctx, "api_gateway", "admin_service", "team_service"); err != nil {
		return nil, err
	}

	md, _ := metadata.FromIncomingContext(ctx)
	callerRole := ""
	if md != nil {
		if v := md.Get("x-user-role"); len(v) > 0 {
			callerRole = v[0]
		}
	}
	switch callerRole {
	case "admin":
	case "teacher":
		dept := mdInt64(md, "x-department-id")
		univ := mdInt64(md, "x-university-id")
		req.DepartmentId = dept
		req.UniversityId = univ
	default:
		dept := mdInt64(md, "x-department-id")
		univ := mdInt64(md, "x-university-id")
		req.DepartmentId = dept
		req.UniversityId = univ
	}

	users, total, err := h.service.ListUsers(
		ctx,
		req.UniversityId,
		req.DepartmentId,
		req.Role,
		req.Page,
		req.PageSize,
		req.ExcludeUserId,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}

	pbUsers := make([]*authv1.UserPreview, 0, len(users))
	for _, u := range users {
		pbUsers = append(pbUsers, &authv1.UserPreview{
			Id:           u.ID,
			Email:        u.Email,
			FirstName:    u.FirstName,
			LastName:     u.LastName,
			Role:         u.Role,
			UniversityId: u.UniversityID,
			DepartmentId: u.DepartmentID,
			IsActive:     &u.IsActive,
		})
	}

	return &authv1.ListUsersResponse{
		Users:      pbUsers,
		TotalCount: total,
	}, nil
}

func (h *Handler) AssignRole(ctx context.Context, req *authv1.AssignRoleRequest) (*authv1.AssignRoleResponse, error) {
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Role == "" {
		return nil, status.Error(codes.InvalidArgument, "role is required")
	}

	if err := requireInternal(ctx, "api_gateway"); err != nil {
		return nil, err
	}
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	if err := h.service.AssignRole(ctx, req.UserId, req.Role); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to assign role: %v", err)
	}

	return &authv1.AssignRoleResponse{
		Success: true,
		Message: "Role assigned successfully",
	}, nil
}

func (h *Handler) BatchGetUserPreviews(ctx context.Context, req *authv1.BatchGetUserPreviewsRequest) (*authv1.BatchGetUserPreviewsResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	internal := ""
	if md != nil {
		if v := md.Get("x-internal-service"); len(v) > 0 {
			internal = v[0]
		}
	}
	// Internal-only RPC: обогащение карточек данными пользователей.
	// Разрешаем сервисам, которым нужно резолвить имена/аватары по id.
	switch internal {
	case "team_service", "task_service":
		// ok
	default:
		return nil, status.Error(codes.PermissionDenied, "forbidden")
	}

	users, err := h.service.BatchGetUserPreviews(ctx, req.Ids)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get users: %v", err)
	}

	pb := make([]*authv1.UserPreview, 0, len(users))
	for _, u := range users {
		pb = append(pb, &authv1.UserPreview{
			Id:           u.ID,
			Email:        u.Email,
			FirstName:    u.FirstName,
			LastName:     u.LastName,
			Role:         u.Role,
			UniversityId: u.UniversityID,
			DepartmentId: u.DepartmentID,
			IsActive:     &u.IsActive,
		})
	}

	return &authv1.BatchGetUserPreviewsResponse{Users: pb}, nil
}

func (h *Handler) CreateDepartmentRole(ctx context.Context, req *authv1.CreateDepartmentRoleRequest) (*authv1.CreateDepartmentRoleResponse, error) {
	if err := requireInternal(ctx, "api_gateway"); err != nil {
		return nil, err
	}
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if req.DepartmentId == 0 || req.Slug == "" {
		return nil, status.Error(codes.InvalidArgument, "department_id and slug are required")
	}

	role, err := h.service.CreateDepartmentRole(ctx, req.DepartmentId, req.Slug)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create department role: %v", err)
	}

	return &authv1.CreateDepartmentRoleResponse{
		Role: &authv1.DepartmentRole{
			Id:           role.ID,
			Slug:         role.Slug,
			DepartmentId: role.DepartmentID,
		},
	}, nil
}

func (h *Handler) ListDepartmentRoles(ctx context.Context, req *authv1.ListDepartmentRolesRequest) (*authv1.ListDepartmentRolesResponse, error) {
	if err := requireInternal(ctx, "api_gateway"); err != nil {
		return nil, err
	}
	if req.DepartmentId == 0 {
		return nil, status.Error(codes.InvalidArgument, "department_id is required")
	}

	roles, err := h.service.ListDepartmentRoles(ctx, req.DepartmentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list department roles: %v", err)
	}

	out := make([]*authv1.DepartmentRole, 0, len(roles))
	for _, r := range roles {
		out = append(out, &authv1.DepartmentRole{
			Id:           r.ID,
			Slug:         r.Slug,
			DepartmentId: r.DepartmentID,
		})
	}

	return &authv1.ListDepartmentRolesResponse{Roles: out}, nil
}

func (h *Handler) GetDepartmentRole(ctx context.Context, req *authv1.GetDepartmentRoleRequest) (*authv1.GetDepartmentRoleResponse, error) {
	if err := requireInternal(ctx, "api_gateway"); err != nil {
		return nil, err
	}
	if req.RoleId == 0 {
		return nil, status.Error(codes.InvalidArgument, "role_id is required")
	}

	role, err := h.service.GetDepartmentRole(ctx, req.RoleId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "role not found")
	}

	return &authv1.GetDepartmentRoleResponse{
		Role: &authv1.DepartmentRole{
			Id:           role.ID,
			Slug:         role.Slug,
			DepartmentId: role.DepartmentID,
		},
	}, nil
}

func (h *Handler) DeleteDepartmentRole(ctx context.Context, req *authv1.DeleteDepartmentRoleRequest) (*authv1.DeleteDepartmentRoleResponse, error) {
	if err := requireInternal(ctx, "api_gateway"); err != nil {
		return nil, err
	}
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if req.RoleId == 0 {
		return nil, status.Error(codes.InvalidArgument, "role_id is required")
	}

	if err := h.service.DeleteDepartmentRole(ctx, req.RoleId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete role: %v", err)
	}

	return &authv1.DeleteDepartmentRoleResponse{Success: true}, nil
}

func (h *Handler) AssignDepartmentRole(ctx context.Context, req *authv1.AssignDepartmentRoleRequest) (*authv1.AssignDepartmentRoleResponse, error) {
	if err := requireInternal(ctx, "api_gateway"); err != nil {
		return nil, err
	}
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if req.UserId == 0 || req.DepartmentId == 0 || req.RoleId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id, department_id, role_id are required")
	}

	md, _ := metadata.FromIncomingContext(ctx)
	assignedBy := mdInt64(md, "x-user-id")

	if err := h.service.AssignDepartmentRole(ctx, req.UserId, req.DepartmentId, req.RoleId, assignedBy, req.Comment); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to assign department role: %v", err)
	}

	return &authv1.AssignDepartmentRoleResponse{Success: true}, nil
}

func (h *Handler) RevokeDepartmentRole(ctx context.Context, req *authv1.RevokeDepartmentRoleRequest) (*authv1.RevokeDepartmentRoleResponse, error) {
	if err := requireInternal(ctx, "api_gateway"); err != nil {
		return nil, err
	}
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if req.UserId == 0 || req.DepartmentId == 0 || req.RoleId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id, department_id, role_id are required")
	}

	md, _ := metadata.FromIncomingContext(ctx)
	revokedBy := mdInt64(md, "x-user-id")

	if err := h.service.RevokeDepartmentRole(ctx, req.UserId, req.DepartmentId, req.RoleId, revokedBy, req.Comment); err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "assignment not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to revoke department role: %v", err)
	}

	return &authv1.RevokeDepartmentRoleResponse{Success: true}, nil
}

func (h *Handler) ListUserDepartmentRoles(ctx context.Context, req *authv1.ListUserDepartmentRolesRequest) (*authv1.ListUserDepartmentRolesResponse, error) {
	if err := requireInternal(ctx, "api_gateway"); err != nil {
		return nil, err
	}
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if req.UserId == 0 || req.DepartmentId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id and department_id are required")
	}

	slugs, err := h.service.ListUserDepartmentRoleSlugs(ctx, req.UserId, req.DepartmentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list user roles: %v", err)
	}

	return &authv1.ListUserDepartmentRolesResponse{Slugs: slugs}, nil
}

func (h *Handler) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.GetUserResponse, error) {
	// Валидация
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	if err := requireInternalOneOf(ctx, "api_gateway", "admin_service"); err != nil {
		return nil, err
	}

	user, err := h.service.GetUser(ctx, req.UserId)
	if err != nil {
		if err == ErrUserNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	return &authv1.GetUserResponse{
		User: &authv1.UserPreview{
			Id:           user.ID,
			Email:        user.Email,
			FirstName:    user.FirstName,
			LastName:     user.LastName,
			Role:         user.Role,
			UniversityId: user.UniversityID,
			DepartmentId: user.DepartmentID,
			IsActive:     &user.IsActive,
		},
	}, nil
}

func (h *Handler) UpdateUser(ctx context.Context, req *authv1.UpdateUserRequest) (*authv1.UpdateUserResponse, error) {
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	if err := requireInternalOneOf(ctx, "api_gateway", "admin_service"); err != nil {
		return nil, err
	}

	md, _ := metadata.FromIncomingContext(ctx)
	requesterRole := ""
	if md != nil {
		if v := md.Get("x-user-role"); len(v) > 0 {
			requesterRole = v[0]
		}
	}
	if req.IsActive != nil && requesterRole != "admin" {
		return nil, status.Error(codes.PermissionDenied, "only admin can change is_active")
	}

	user, err := h.service.UpdateUser(
		ctx,
		req.UserId,
		req.Email,
		req.FirstName,
		req.LastName,
		req.Role,
		req.UniversityId,
		req.DepartmentId,
		req.IsActive,
		requesterRole,
	)
	if err != nil {
		switch err {
		case ErrUserNotFound:
			return nil, status.Error(codes.NotFound, "user not found")
		case ErrEmailTaken:
			return nil, status.Error(codes.AlreadyExists, "email already taken")
		default:
			return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
		}
	}

	return &authv1.UpdateUserResponse{
		User: &authv1.UserPreview{
			Id:           user.ID,
			Email:        user.Email,
			FirstName:    user.FirstName,
			LastName:     user.LastName,
			Role:         user.Role,
			UniversityId: user.UniversityID,
			DepartmentId: user.DepartmentID,
			IsActive:     &user.IsActive,
		},
	}, nil
}

func (h *Handler) DeleteUser(ctx context.Context, req *authv1.DeleteUserRequest) (*authv1.DeleteUserResponse, error) {
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.RequesterId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "requester_id is required")
	}

	if err := requireInternalOneOf(ctx, "api_gateway", "admin_service"); err != nil {
		return nil, err
	}

	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	err := h.service.DeleteUser(ctx, req.UserId, req.RequesterId)
	if err != nil {
		switch err {
		case ErrSelfDelete:
			return nil, status.Error(codes.FailedPrecondition, "cannot delete yourself")
		case ErrUserNotFound:
			return nil, status.Error(codes.NotFound, "user not found")
		default:
			return nil, status.Errorf(codes.Internal, "failed to delete user: %v", err)
		}
	}

	return &authv1.DeleteUserResponse{Success: true}, nil
}
func (h *Handler) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	if err := requireInternal(ctx, "api_gateway"); err != nil {
		return nil, err
	}

	if req == nil || strings.TrimSpace(req.RefreshToken) == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}

	ip, userAgent := pickClientInfo(ctx, req.IpAddress, req.UserAgent)

	accessToken, refreshToken, err := h.service.RefreshToken(ctx, req.RefreshToken, userAgent, ip)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}

	return &authv1.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
func (h *Handler) ListSessions(ctx context.Context, req *authv1.ListSessionsRequest) (*authv1.ListSessionsResponse, error) {
	if err := requireInternal(ctx, "api_gateway"); err != nil {
		return nil, err
	}

	if req == nil || req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	sessions, err := h.service.ListSessions(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list sessions: %v", err)
	}

	out := make([]*authv1.Session, 0, len(sessions))
	for _, s := range sessions {
		if s == nil {
			continue
		}
		out = append(out, &authv1.Session{
			Id:        s.ID,
			UserAgent: s.UserAgent,
			IpAddress: s.ClientIP,
			CreatedAt: s.CreatedAt.UTC().Format(time.RFC3339),
			ExpiresAt: s.ExpiresAt.UTC().Format(time.RFC3339),
			IsCurrent: false, // current session id мы сейчас не знаем
		})
	}

	return &authv1.ListSessionsResponse{Sessions: out}, nil
}
func (h *Handler) RevokeSession(ctx context.Context, req *authv1.RevokeSessionRequest) (*authv1.RevokeSessionResponse, error) {
	if err := requireInternal(ctx, "api_gateway"); err != nil {
		return nil, err
	}

	if req == nil || req.UserId <= 0 || req.SessionId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id and session_id are required")
	}

	err := h.service.RevokeSession(ctx, req.UserId, req.SessionId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		// в service у тебя unauthorized = errors.New("unauthorized")
		if strings.Contains(strings.ToLower(err.Error()), "unauthorized") {
			return nil, status.Error(codes.PermissionDenied, "forbidden")
		}
		return nil, status.Errorf(codes.Internal, "failed to revoke session: %v", err)
	}

	return &authv1.RevokeSessionResponse{Success: true}, nil
}
