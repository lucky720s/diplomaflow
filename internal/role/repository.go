package role

import (
	"context"
	"fmt"
	"os"

	rkpostgres "github.com/rookie-ninja/rk-db/postgres"
	"gorm.io/gorm"
)

type Role struct {
	ID           int64  `gorm:"primaryKey"`
	Name         string `gorm:"index:idx_department_role_name,unique"`
	DepartmentID int64  `gorm:"index:idx_department_role_name,unique"`
}
type Repository interface {
	CreateRole(ctx context.Context, name string, departmentID int64) (*Role, error)
	GetRole(ctx context.Context, roleID int64) (*Role, error)
	ListRoles(ctx context.Context, departmentID int64) ([]Role, error)
	DeleteRole(ctx context.Context, roleID int64) error
}
type repository struct {
	db *gorm.DB
}

func NewRepository() (Repository, error) {
	pgEntry := rkpostgres.GetPostgresEntry("role-conn")
	dbName := os.Getenv("ROLE_DB_NAME")
	db := pgEntry.GetDB(dbName)
	if db == nil {
		return nil, fmt.Errorf("DB Not Found")
	}
	if err := db.AutoMigrate(&Role{}); err != nil {
		return nil, fmt.Errorf("AutoMigrate Role Error: %v", err)
	}
	return &repository{db: db}, nil
}

func (r *repository) CreateRole(ctx context.Context, name string, departmentID int64) (*Role, error) {
	role := &Role{Name: name, DepartmentID: departmentID}
	res := r.db.WithContext(ctx).Create(role)
	if res.Error != nil {
		return nil, res.Error
	}
	return role, nil
}
func (r *repository) GetRole(ctx context.Context, roleID int64) (*Role, error) {
	var role Role
	res := r.db.WithContext(ctx).First(&role, roleID)
	if res.Error != nil {
		return nil, res.Error
	}
	return &role, nil
}
func (r *repository) ListRoles(ctx context.Context, departmentID int64) ([]Role, error) {
	var roles []Role
	err := r.db.WithContext(ctx).Where("department_id = ?", departmentID).Find(&roles).Error
	return roles, err
}

func (r *repository) DeleteRole(ctx context.Context, roleID int64) error {
	return r.db.WithContext(ctx).Delete(&Role{}, roleID).Error
}
