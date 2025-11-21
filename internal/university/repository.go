package university

import (
	"context"
	"fmt"
	"os"

	universityv1 "github.com/lucky720s/diplomaflow/protobuf/university/v1"
	rkpostgres "github.com/rookie-ninja/rk-db/postgres"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
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
	ListUniversities(ctx context.Context) ([]*University, error)
	UpdateUniversity(ctx context.Context, university *universityv1.University, mask *fieldmaskpb.FieldMask) (*University, error)
	DeleteUniversity(ctx context.Context, universityID int64) error
	GetUniversityByID(ctx context.Context, universityID int64) (*University, error)
	CreateDepartment(ctx context.Context, name string, universityID int64) (*Department, error)
	GetDepartmentByID(ctx context.Context, departmentID int64) (*Department, error)
	ListDepartments(ctx context.Context, universityID int64) ([]*Department, error)
	UpdateDepartment(ctx context.Context, department *universityv1.Department, mask *fieldmaskpb.FieldMask) (*Department, error)
	DeleteDepartment(ctx context.Context, departmentID int64) error
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

func (r *repository) GetUniversityByID(ctx context.Context, universityID int64) (*University, error) {
	var university University
	res := r.db.WithContext(ctx).First(&university, universityID)
	if res.Error != nil {
		return nil, res.Error
	}
	return &university, nil

}
func (r *repository) ListUniversities(ctx context.Context) ([]*University, error) {
	var universities []*University
	err := r.db.WithContext(ctx).Find(&universities).Error
	if err != nil {
		return nil, err
	}
	return universities, nil
}
func (r *repository) UpdateUniversity(ctx context.Context, university *universityv1.University, mask *fieldmaskpb.FieldMask) (*University, error) {
	var existingUniversity University
	if err := r.db.WithContext(ctx).First(&existingUniversity, university.GetId()).Error; err != nil {
		return nil, err
	}
	updateData := make(map[string]interface{})
	for _, path := range mask.Paths {
		switch path {
		case "name":
			updateData["name"] = university.GetName()
		case "short_name":
			updateData["short_name"] = university.GetShortName()

		}
	}
	if len(updateData) > 0 {
		if err := r.db.WithContext(ctx).Model(&existingUniversity).Association("Update").Error; err != nil {
			return nil, err
		}
	}
	return &existingUniversity, nil
}
func (r *repository) DeleteUniversity(ctx context.Context, universityID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("university_id=?", universityID).Delete(&University{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&University{}).Error; err != nil {
			return err
		}
		return nil
	})
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
func (r *repository) ListDepartments(ctx context.Context, universityID int64) ([]*Department, error) {
	var departments []*Department
	err := r.db.WithContext(ctx).Find(&departments).Error
	if err != nil {
		return nil, err
	}
	return departments, nil
}
func (r *repository) UpdateDepartment(ctx context.Context, department *universityv1.Department, mask *fieldmaskpb.FieldMask) (*Department, error) {
	var existingDepartment Department
	if err := r.db.WithContext(ctx).First(&existingDepartment, department.GetId()).Error; err != nil {
		return nil, err
	}
	updateData := make(map[string]interface{})
	for _, path := range mask.Paths {
		switch path {
		case "name":
			updateData["name"] = department.GetName()
		}
	}
	if len(updateData) > 0 {
		if err := r.db.WithContext(ctx).Model(&existingDepartment).Association("Update").Error; err != nil {
			return nil, err
		}
	}
	return &existingDepartment, nil
}
func (r *repository) DeleteDepartment(ctx context.Context, departmentID int64) error {
	return r.db.WithContext(ctx).Delete(&Department{}).Error
}
