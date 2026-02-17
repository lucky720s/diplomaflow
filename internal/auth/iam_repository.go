package auth

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type IAMRepository interface {
	// Roles directory
	CreateDepartmentRole(ctx context.Context, r *DepartmentRole) error
	GetDepartmentRole(ctx context.Context, roleID int64) (*DepartmentRole, error)
	ListDepartmentRoles(ctx context.Context, departmentID int64) ([]*DepartmentRole, error)
	DeleteDepartmentRole(ctx context.Context, roleID int64) error

	// Assignments
	AssignDepartmentRole(ctx context.Context, a *UserRoleAssignment) error
	RevokeDepartmentRole(ctx context.Context, userID, departmentID, roleID int64, revokedBy int64, comment string) (bool, error)
	ListUserDepartmentRoleSlugs(ctx context.Context, userID, departmentID int64) ([]string, error)
}

type iamRepository struct {
	db *gorm.DB
}

func NewIAMRepository(db *gorm.DB) IAMRepository {
	return &iamRepository{db: db}
}

func (r *iamRepository) CreateDepartmentRole(ctx context.Context, role *DepartmentRole) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *iamRepository) GetDepartmentRole(ctx context.Context, roleID int64) (*DepartmentRole, error) {
	var role DepartmentRole
	if err := r.db.WithContext(ctx).First(&role, roleID).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *iamRepository) ListDepartmentRoles(ctx context.Context, departmentID int64) ([]*DepartmentRole, error) {
	var roles []*DepartmentRole
	q := r.db.WithContext(ctx).Model(&DepartmentRole{}).
		Where("department_id = ?", departmentID).
		Order("id ASC")
	if err := q.Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *iamRepository) DeleteDepartmentRole(ctx context.Context, roleID int64) error {
	res := r.db.WithContext(ctx).Delete(&DepartmentRole{}, roleID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *iamRepository) AssignDepartmentRole(ctx context.Context, a *UserRoleAssignment) error {
	if a.AssignedAt.IsZero() {
		a.AssignedAt = time.Now()
	}
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *iamRepository) RevokeDepartmentRole(ctx context.Context, userID, departmentID, roleID int64, revokedBy int64, comment string) (bool, error) {
	now := time.Now()

	// revoke only active assignment (revoked_at IS NULL) for exact scope dept_id and workflow_id IS NULL
	res := r.db.WithContext(ctx).
		Model(&UserRoleAssignment{}).
		Where("user_id = ? AND role_id = ? AND department_id = ? AND workflow_id IS NULL AND revoked_at IS NULL",
			userID, roleID, departmentID).
		Updates(map[string]interface{}{
			"revoked_at": now,
			"comment":    comment,
		})

	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *iamRepository) ListUserDepartmentRoleSlugs(ctx context.Context, userID, departmentID int64) ([]string, error) {
	if userID <= 0 || departmentID <= 0 {
		return []string{}, nil
	}

	type row struct {
		Slug string `gorm:"column:slug"`
	}

	var rows []row
	err := r.db.WithContext(ctx).
		Table("user_role_assignments ura").
		Select("roles.name as slug").
		Joins("JOIN roles ON roles.id = ura.role_id").
		Where("ura.user_id = ? AND ura.department_id = ? AND ura.workflow_id IS NULL AND ura.revoked_at IS NULL AND roles.deleted_at IS NULL",
			userID, departmentID).
		Order("roles.name ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	slugs := make([]string, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		if r.Slug == "" {
			continue
		}
		if _, ok := seen[r.Slug]; ok {
			continue
		}
		seen[r.Slug] = struct{}{}
		slugs = append(slugs, r.Slug)
	}
	return slugs, nil
}

var ErrIAMForbidden = errors.New("forbidden")
