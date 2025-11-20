package auth

import (
	"context"
	"fmt"
	"os"

	rkpostgres "github.com/rookie-ninja/rk-db/postgres"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID           int64  `gorm:"primary_key"`
	Email        string `gorm:"uniqueIndex"`
	PasswordHash string
	UniversityID int64
	DepartmentID int64
}
type RoleAssignment struct {
	ID     int64 `gorm:"primaryKey"`
	UserID int64 `gorm:"index:idx_user_role,unique"`
	RoleID int64 `gorm:"index:idx_user_role,unique"`
}

type Repository interface {
	CreateUser(ctx context.Context, email, password string, universityID int64) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id int64) (*User, error)
	AssignRole(ctx context.Context, userID, roleID int64) error
	GetUserRoleIDs(ctx context.Context, userID int64) ([]int64, error)
	SetDepartment(ctx context.Context, userID, departmentID int64) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository() (Repository, error) {
	pgEntry := rkpostgres.GetPostgresEntry("auth-conn")
	dbName := os.Getenv("AUTH_DB_NAME")
	db := pgEntry.GetDB(dbName)
	if db == nil {
		panic("Database not found")
	}
	if err := db.AutoMigrate(&User{}, &RoleAssignment{}); err != nil {
		return nil, fmt.Errorf("AutoMigrate User Error: %v", err)
	}
	return &repository{db: db}, nil
}

func (r *repository) CreateUser(ctx context.Context, email, password string, universityID int64) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &User{
		Email:        email,
		PasswordHash: string(hashedPassword),
		UniversityID: universityID,
	}
	res := r.db.WithContext(ctx).Create(user)
	return user, res.Error
}
func (r *repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	res := r.db.WithContext(ctx).Where("email = ?", email).First(&user)
	return &user, res.Error
}
func (r *repository) GetUserByID(ctx context.Context, id int64) (*User, error) {
	var user User
	res := r.db.WithContext(ctx).Where("id = ?", id).First(&user)
	return &user, res.Error
}

func (r *repository) AssignRole(ctx context.Context, userID, roleID int64) error {
	assignment := &RoleAssignment{
		UserID: userID,
		RoleID: roleID,
	}
	return r.db.WithContext(ctx).Create(assignment).Error
}
func (r *repository) GetUserRoleIDs(ctx context.Context, userID int64) ([]int64, error) {
	var assignments []RoleAssignment
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&assignments).Error
	if err != nil {
		return nil, err
	}
	roleIDs := make([]int64, 0, len(assignments))
	for _, a := range assignments {
		roleIDs = append(roleIDs, a.RoleID)
	}
	return roleIDs, nil
}
func (r *repository) SetDepartment(ctx context.Context, userID, departmentID int64) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Update("department_id", departmentID).Error
}
