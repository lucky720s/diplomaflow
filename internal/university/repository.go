package university

import (
	"context"
	"fmt"
	"os"

	rkpostgres "github.com/rookie-ninja/rk-db/postgres"
	"gorm.io/gorm"
)

type University struct {
	ID        int64 `gorm:"primaryKey"`
	Name      string
	ShortName string `gorm:"uniqueIndex"`
}
type Department struct {
	ID           int64 `gorm:"primaryKey"`
	Name         string
	UniversityID int64
	University   University `gorm:"foreignKey:UniversityID"`
}
type Repository interface {
	CreateUniversity(ctx context.Context, name, shortName string) (*University, error)
	GetUniversityByShortName(ctx context.Context, shortName string) (*University, error)
	GetUniversityByID(ctx context.Context, id int64) (*University, error)
	CreateDepartment(ctx context.Context, name string, universityID int64) (*Department, error)
	GetDepartmentByID(ctx context.Context, departmentID int64) (*Department, error)
}
type repository struct {
	db *gorm.DB
}

func NewRepository() (Repository, error) {
	pgEntry := rkpostgres.GetPostgresEntry("university-conn")
	dbName := os.Getenv("UNIVERSITY_DB_NAME")
	db := pgEntry.GetDB(dbName)
	if db == nil {
		panic("Database not found")
	}
	if err := db.AutoMigrate(&University{}, &Department{}); err != nil {
		return nil, fmt.Errorf("auto migrate university err: %w", err)
	}
	return &repository{db: db}, nil
}
func (r *repository) CreateUniversity(ctx context.Context, name, shortName string) (*University, error) {
	university := &University{Name: name, ShortName: shortName}
	res := r.db.WithContext(ctx).Create(university)
	if res.Error != nil {
		return nil, res.Error
	}
	return university, nil
}
func (r *repository) GetUniversityByShortName(ctx context.Context, shortName string) (*University, error) {
	var university University
	res := r.db.WithContext(ctx).First(&university, "short_name = ?", shortName)
	if res.Error != nil {
		return nil, res.Error
	}
	return &university, nil
}
func (r *repository) GetUniversityByID(ctx context.Context, id int64) (*University, error) {
	var university University
	res := r.db.WithContext(ctx).First(&university, id)
	if res.Error != nil {
		return nil, res.Error
	}
	return &university, nil
}

func (r *repository) CreateDepartment(ctx context.Context, name string, universityID int64) (*Department, error) {
	department := &Department{Name: name, UniversityID: universityID}
	res := r.db.WithContext(ctx).Create(&department)
	if res.Error != nil {
		return nil, res.Error
	}
	return department, nil
}
func (r *repository) GetDepartmentByID(ctx context.Context, departmentID int64) (*Department, error) {
	var department Department
	res := r.db.WithContext(ctx).First(&department, departmentID)
	if res.Error != nil {
		return nil, res.Error
	}
	return &department, nil
}
