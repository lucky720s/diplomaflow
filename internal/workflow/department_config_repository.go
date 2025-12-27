// internal/workflow/department_config_repository.go
package workflow

import (
	"context"

	"gorm.io/gorm"
)

type DepartmentConfigRepository struct {
	db *gorm.DB
}

func NewDepartmentConfigRepository(db *gorm.DB) *DepartmentConfigRepository {
	return &DepartmentConfigRepository{db: db}
}

func (r *DepartmentConfigRepository) Create(ctx context.Context, config *DepartmentWorkflowConfig) error {
	return r.db.WithContext(ctx).Create(config).Error
}

func (r *DepartmentConfigRepository) Get(ctx context.Context, id int64) (*DepartmentWorkflowConfig, error) {
	var config DepartmentWorkflowConfig
	err := r.db.WithContext(ctx).First(&config, id).Error
	return &config, err
}

func (r *DepartmentConfigRepository) GetActiveConfig(ctx context.Context, departmentID int64, academicYear string) (*DepartmentWorkflowConfig, error) {
	var config DepartmentWorkflowConfig
	err := r.db.WithContext(ctx).
		Where("department_id = ? AND academic_year = ? AND is_active = ?", departmentID, academicYear, true).
		First(&config).Error
	return &config, err
}

func (r *DepartmentConfigRepository) Update(ctx context.Context, config *DepartmentWorkflowConfig) error {
	return r.db.WithContext(ctx).Save(config).Error
}

func (r *DepartmentConfigRepository) DeactivateAll(ctx context.Context, departmentID int64, academicYear string) error {
	return r.db.WithContext(ctx).
		Model(&DepartmentWorkflowConfig{}).
		Where("department_id = ? AND academic_year = ?", departmentID, academicYear).
		Update("is_active", false).Error
}

func (r *DepartmentConfigRepository) GetCustomSteps(ctx context.Context, configID int64) ([]DepartmentCustomStep, error) {
	var steps []DepartmentCustomStep
	err := r.db.WithContext(ctx).
		Where("department_config_id = ?", configID).
		Order("id ASC").
		Find(&steps).Error
	return steps, err
}

func (r *DepartmentConfigRepository) CreateCustomStep(ctx context.Context, step *DepartmentCustomStep) error {
	return r.db.WithContext(ctx).Create(step).Error
}

func (r *DepartmentConfigRepository) List(ctx context.Context, departmentID int64, academicYear string) ([]*DepartmentWorkflowConfig, error) {
	var configs []*DepartmentWorkflowConfig
	query := r.db.WithContext(ctx).Where("department_id = ?", departmentID)
	if academicYear != "" {
		query = query.Where("academic_year = ?", academicYear)
	}
	err := query.Find(&configs).Error
	return configs, err
}
