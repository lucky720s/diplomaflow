package auth

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

var slugRe = regexp.MustCompile(`^[a-z][a-z0-9_]{2,31}$`)

func normalizeSlug(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

func validateSlug(s string) error {
	if !slugRe.MatchString(s) {
		return fmt.Errorf("invalid slug: must match %s", slugRe.String())
	}
	return nil
}

func (s *Service) CreateDepartmentRole(ctx context.Context, departmentID int64, slug string) (*DepartmentRole, error) {
	if departmentID <= 0 {
		return nil, fmt.Errorf("department_id is required")
	}
	slug = normalizeSlug(slug)
	if err := validateSlug(slug); err != nil {
		return nil, err
	}

	role := &DepartmentRole{
		Slug:         slug,
		DepartmentID: departmentID,
	}
	if err := s.iamRepo.CreateDepartmentRole(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *Service) ListDepartmentRoles(ctx context.Context, departmentID int64) ([]*DepartmentRole, error) {
	if departmentID <= 0 {
		return nil, fmt.Errorf("department_id is required")
	}
	return s.iamRepo.ListDepartmentRoles(ctx, departmentID)
}

func (s *Service) GetDepartmentRole(ctx context.Context, roleID int64) (*DepartmentRole, error) {
	if roleID <= 0 {
		return nil, fmt.Errorf("role_id is required")
	}
	return s.iamRepo.GetDepartmentRole(ctx, roleID)
}

func (s *Service) DeleteDepartmentRole(ctx context.Context, roleID int64) error {
	if roleID <= 0 {
		return fmt.Errorf("role_id is required")
	}
	return s.iamRepo.DeleteDepartmentRole(ctx, roleID)
}

func (s *Service) AssignDepartmentRole(ctx context.Context, userID, departmentID, roleID int64, assignedBy int64, comment string) error {
	if userID <= 0 {
		return fmt.Errorf("user_id is required")
	}
	if departmentID <= 0 {
		return fmt.Errorf("department_id is required")
	}
	if roleID <= 0 {
		return fmt.Errorf("role_id is required")
	}

	// ensure user exists
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if u.DepartmentID != 0 && u.DepartmentID != departmentID {
		return fmt.Errorf("user is in another department (user.department_id=%d)", u.DepartmentID)
	}

	// ensure role exists and belongs to department
	r, err := s.iamRepo.GetDepartmentRole(ctx, roleID)
	if err != nil {
		return fmt.Errorf("role not found: %w", err)
	}
	if r.DepartmentID != departmentID {
		return fmt.Errorf("role belongs to another department (role.department_id=%d)", r.DepartmentID)
	}

	a := &UserRoleAssignment{
		UserID:       userID,
		RoleID:       roleID,
		DepartmentID: &departmentID,
		WorkflowID:   nil,
		Comment:      comment,
	}
	if assignedBy > 0 {
		a.AssignedBy = &assignedBy
	}

	// UNIQUE index already exists in DB on your schema (ux_user_role_scope) for active assignments
	if err := s.iamRepo.AssignDepartmentRole(ctx, a); err != nil {
		return err
	}
	return nil
}

func (s *Service) RevokeDepartmentRole(ctx context.Context, userID, departmentID, roleID int64, revokedBy int64, comment string) error {
	if userID <= 0 || departmentID <= 0 || roleID <= 0 {
		return fmt.Errorf("user_id, department_id, role_id are required")
	}
	ok, err := s.iamRepo.RevokeDepartmentRole(ctx, userID, departmentID, roleID, revokedBy, comment)
	if err != nil {
		return err
	}
	if !ok {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Service) ListUserDepartmentRoleSlugs(ctx context.Context, userID, departmentID int64) ([]string, error) {
	return s.iamRepo.ListUserDepartmentRoleSlugs(ctx, userID, departmentID)
}
