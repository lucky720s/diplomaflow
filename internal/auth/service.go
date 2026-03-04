package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lucky720s/diplomaflow/pkg/logger"
	universityv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/university/v1"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// =====================================================
// Error definitions
// =====================================================

var (
	ErrUserNotFound = errors.New("user not found")
	ErrSelfDelete   = errors.New("cannot delete yourself")
	ErrEmailTaken   = errors.New("email already taken")
)

type Service struct {
	repo       Repository
	iamRepo    IAMRepository
	jwtWrapper JwtWrapper

	univClient universityv1.UniversityServiceClient

	logger *logger.Logger
}

func NewService(
	repo Repository,
	iamRepo IAMRepository,
	jwtWrapper JwtWrapper,
	univClient universityv1.UniversityServiceClient,
	log *logger.Logger,
) *Service {
	return &Service{
		repo:       repo,
		iamRepo:    iamRepo,
		jwtWrapper: jwtWrapper,
		univClient: univClient,
		logger:     log,
	}
}

func (s *Service) Register(ctx context.Context, email, password, firstName, lastName, role string, universityID, departmentID int64) (int64, error) {
	existingUser, err := s.repo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return 0, errors.New("user already exists")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	hashedPassword, err := HashPassword(password)
	if err != nil {
		return 0, err
	}

	user := &User{
		Email:        email,
		Password:     hashedPassword,
		FirstName:    firstName,
		LastName:     lastName,
		Role:         role,
		UniversityID: universityID,
		DepartmentID: departmentID,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return 0, err
	}

	return user.ID, nil
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (string, string, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", errors.New("invalid credentials")
	}
	if !CheckPasswordHash(password, user.Password) {
		return "", "", errors.New("invalid credentials")
	}

	deptRoles, err := s.iamRepo.ListUserDepartmentRoleSlugs(ctx, user.ID, user.DepartmentID)
	if err != nil {
		s.logger.Warn("failed to load dept roles; continue without them", zap.Error(err))
		deptRoles = []string{}
	}

	accessToken, err := s.jwtWrapper.GenerateAccessToken(*user, deptRoles)
	if err != nil {
		return "", "", err
	}

	rawUUID := s.jwtWrapper.GenerateRefreshTokenSecret()
	tokenHash, err := HashToken(rawUUID)
	if err != nil {
		return "", "", err
	}

	refreshTokenModel := &RefreshToken{
		UserID:    user.ID,
		Token:     tokenHash,
		UserAgent: userAgent,
		ClientIP:  ip,
		ExpiresAt: time.Now().Add(s.jwtWrapper.RefreshTokenTTL),
		Revoked:   false,
	}
	if err := s.repo.CreateRefreshToken(ctx, refreshTokenModel); err != nil {
		return "", "", err
	}
	clientRefreshToken := FormatRefreshToken(refreshTokenModel.ID, rawUUID)

	return accessToken, clientRefreshToken, nil
}

func (s *Service) RefreshToken(ctx context.Context, clientToken, userAgent, ip string) (string, string, error) {
	id, rawUUID, err := ParseRefreshToken(clientToken)
	if err != nil {
		return "", "", errors.New("invalid refresh token format")
	}

	storedToken, err := s.repo.GetRefreshTokenByID(ctx, id)
	if err != nil {
		return "", "", errors.New("refresh token not found")
	}
	if storedToken.UserAgent != userAgent {
		s.logger.Warn("Refresh token used with different User-Agent",
			zap.String("old", storedToken.UserAgent),
			zap.String("new", userAgent))
	}
	if !CheckTokenHash(rawUUID, storedToken.Token) {
		return "", "", errors.New("invalid refresh token")
	}

	if storedToken.Revoked {
		_ = s.repo.RevokeAllUserTokens(ctx, storedToken.UserID)
		return "", "", errors.New("security alert: token reuse detected")
	}

	if storedToken.ExpiresAt.Before(time.Now()) {
		return "", "", errors.New("refresh token expired")
	}

	user, err := s.repo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return "", "", err
	}

	if err := s.repo.RevokeRefreshToken(ctx, storedToken.ID); err != nil {
		return "", "", err
	}

	deptRoles, err := s.iamRepo.ListUserDepartmentRoleSlugs(ctx, user.ID, user.DepartmentID)
	if err != nil {
		s.logger.Warn("failed to load dept roles; continue without them", zap.Error(err))
		deptRoles = []string{}
	}

	newAccessToken, _ := s.jwtWrapper.GenerateAccessToken(*user, deptRoles)

	newRawUUID := s.jwtWrapper.GenerateRefreshTokenSecret()
	newHash, _ := HashToken(newRawUUID)

	newStoredToken := &RefreshToken{
		UserID:    user.ID,
		Token:     newHash,
		UserAgent: userAgent,
		ClientIP:  ip,
		ExpiresAt: time.Now().Add(s.jwtWrapper.RefreshTokenTTL),
	}

	if err := s.repo.CreateRefreshToken(ctx, newStoredToken); err != nil {
		return "", "", err
	}

	newClientToken := FormatRefreshToken(newStoredToken.ID, newRawUUID)

	return newAccessToken, newClientToken, nil
}

func (s *Service) Validate(ctx context.Context, token string) (*JwtClaims, error) {
	return s.jwtWrapper.ValidateToken(token)
}

func (s *Service) ListUsers(ctx context.Context, universityID int64, departmentID int64, role string, page, pageSize int32, excludeUserID int64) ([]*User, int64, error) {
	if pageSize <= 0 {
		pageSize = 10
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	filter := UserFilter{
		UniversityID:  universityID,
		DepartmentID:  departmentID,
		Role:          role,
		ExcludeUserID: excludeUserID,
		Limit:         int(pageSize),
		Offset:        int(offset),
	}

	return s.repo.ListUsers(ctx, filter)
}

func (s *Service) ListSessions(ctx context.Context, userID int64) ([]*RefreshToken, error) {
	return s.repo.ListActiveSessions(ctx, userID)
}

func (s *Service) RevokeSession(ctx context.Context, userID int64, sessionID uint64) error {
	token, err := s.repo.GetRefreshTokenByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if token.UserID != userID {
		return errors.New("unauthorized")
	}
	return s.repo.RevokeRefreshToken(ctx, sessionID)
}

// Base role change (single role in users.role)
func (s *Service) AssignRole(ctx context.Context, userID int64, role string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	validRoles := map[string]bool{
		"student": true,
		"teacher": true,
		"admin":   true,
	}
	if !validRoles[role] {
		return fmt.Errorf("invalid role: %s", role)
	}

	user.Role = role
	return s.repo.Update(ctx, user)
}

func (s *Service) BatchGetUserPreviews(ctx context.Context, ids []int64) ([]*User, error) {
	uniq := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return []*User{}, nil
	}
	if len(uniq) > 200 {
		return nil, fmt.Errorf("too many ids (max 200)")
	}

	return s.repo.GetByIDs(ctx, uniq)
}

func (s *Service) GetUser(ctx context.Context, userID int64) (*User, error) {
	if userID <= 0 {
		return nil, ErrUserNotFound
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		s.logger.Error("GetUser failed", zap.Int64("user_id", userID), zap.Error(err))
		return nil, err
	}

	return user, nil
}

func (s *Service) UpdateUser(ctx context.Context, userID int64, email, firstName, lastName, role string, universityID, departmentID int64, requesterRole string) (*User, error) {
	if userID <= 0 {
		return nil, ErrUserNotFound
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		s.logger.Error("UpdateUser: failed to get user", zap.Int64("user_id", userID), zap.Error(err))
		return nil, err
	}

	if email != "" && email != user.Email {
		existingUser, err := s.repo.GetByEmail(ctx, email)
		if err == nil && existingUser != nil && existingUser.ID != userID {
			return nil, ErrEmailTaken
		}
		user.Email = email
	}

	if firstName != "" {
		user.FirstName = firstName
	}
	if lastName != "" {
		user.LastName = lastName
	}

	if requesterRole == "admin" {
		if role != "" {
			validRoles := map[string]bool{
				"student": true,
				"teacher": true,
				"admin":   true,
			}
			if !validRoles[role] {
				return nil, fmt.Errorf("invalid role: %s", role)
			}
			user.Role = role
		}
		if universityID > 0 {
			user.UniversityID = universityID
		}
		if departmentID > 0 {
			user.DepartmentID = departmentID
		}
	} else {
		if role != "" || universityID > 0 || departmentID > 0 {
			s.logger.Warn("Non-admin tried to change restricted fields",
				zap.Int64("user_id", userID),
				zap.String("requester_role", requesterRole),
			)
		}
	}

	if err := s.repo.Update(ctx, user); err != nil {
		s.logger.Error("UpdateUser failed", zap.Int64("user_id", userID), zap.Error(err))
		return nil, err
	}

	s.logger.Info("User updated",
		zap.Int64("user_id", userID),
		zap.String("updated_by_role", requesterRole),
	)

	return user, nil
}

func (s *Service) DeleteUser(ctx context.Context, userID, requesterID int64) error {
	if userID <= 0 {
		return ErrUserNotFound
	}

	if userID == requesterID {
		s.logger.Warn("Self-delete attempted",
			zap.Int64("user_id", userID),
			zap.Int64("requester_id", requesterID),
		)
		return ErrSelfDelete
	}

	_, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		s.logger.Error("DeleteUser: failed to get user", zap.Int64("user_id", userID), zap.Error(err))
		return err
	}

	if err := s.repo.RevokeAllUserTokens(ctx, userID); err != nil {
		s.logger.Warn("Failed to revoke user tokens before delete",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
	}

	if err := s.repo.Delete(ctx, userID); err != nil {
		s.logger.Error("DeleteUser failed", zap.Int64("user_id", userID), zap.Error(err))
		return err
	}

	s.logger.Info("User deleted",
		zap.Int64("user_id", userID),
		zap.Int64("deleted_by", requesterID),
	)

	return nil
}
