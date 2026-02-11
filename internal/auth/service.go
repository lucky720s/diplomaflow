package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lucky720s/diplomaflow/pkg/logger"
	rolev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/role/v1"
	universityv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/university/v1"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service struct {
	repo       Repository
	jwtWrapper JwtWrapper
	univClient universityv1.UniversityServiceClient
	roleClient rolev1.RoleServiceClient
	logger     *logger.Logger
}

func NewService(
	repo Repository,
	jwtWrapper JwtWrapper,
	univClient universityv1.UniversityServiceClient,
	roleClient rolev1.RoleServiceClient,
	log *logger.Logger,
) *Service {
	return &Service{
		repo:       repo,
		jwtWrapper: jwtWrapper,
		univClient: univClient,
		roleClient: roleClient,
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
	accessToken, err := s.jwtWrapper.GenerateAccessToken(*user)
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
	newAccessToken, _ := s.jwtWrapper.GenerateAccessToken(*user)

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
	// дедуп + лимит
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
