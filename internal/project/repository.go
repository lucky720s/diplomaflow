package project

import (
	"context"
	"fmt"
	"os"
	"time"

	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	rkpostgres "github.com/rookie-ninja/rk-db/postgres"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"gorm.io/gorm"
)

type Project struct {
	ID             int64 `gorm:"primaryKey"`
	Topic          string
	SupervisorID   int64
	DepartmentID   int64 `gorm:"index"`
	TeamID         int64
	CurrentStageID int64
	CompletedAt    *time.Time
}
type Repository interface {
	CreateProject(ctx context.Context, p *Project) (*Project, error)
	GetProjectByID(ctx context.Context, projectID int64) (*Project, error)
	ListProjects(ctx context.Context, departmentID int64) ([]*Project, error)
	UpdateProject(ctx context.Context, project *projectv1.Project, mask *fieldmaskpb.FieldMask) (*Project, error)
	DeleteProject(ctx context.Context, projectID int64) error
	UpdateProjectStage(ctx context.Context, projectID, stageID int64) error
	CompleteProject(ctx context.Context, projectID int64) error
}
type repository struct {
	db *gorm.DB
}

func NewRepository() (Repository, error) {
	pgEntry := rkpostgres.GetPostgresEntry("project-conn")
	dbName := os.Getenv("PROJECT_DB_NAME")
	db := pgEntry.GetDB(dbName)
	if db == nil {
		panic("Database not found")
	}
	if err := db.AutoMigrate(&Project{}); err != nil {
		return nil, fmt.Errorf("AutoMigrate Project Error: %v", err)
	}

	return &repository{db: db}, nil
}

func (r *repository) CreateProject(ctx context.Context, p *Project) (*Project, error) {
	res := r.db.WithContext(ctx).Create(p)
	if res.Error != nil {
		return nil, res.Error
	}
	return p, nil

}
func (r *repository) GetProjectByID(ctx context.Context, projectID int64) (*Project, error) {
	var project Project
	err := r.db.WithContext(ctx).First(&project, projectID).Error
	return &project, err
}
func (r *repository) ListProjects(ctx context.Context, departmentID int64) ([]*Project, error) {
	var projects []*Project
	err := r.db.WithContext(ctx).Where("department_id=?", departmentID).Find(&projects).Error
	return projects, err
}
func (r *repository) UpdateProject(ctx context.Context, project *projectv1.Project, mask *fieldmaskpb.FieldMask) (*Project, error) {
	var existingProject Project
	if err := r.db.WithContext(ctx).First(&existingProject, project.GetId()).Error; err != nil {
		return nil, err
	}
	updateData := make(map[string]interface{})
	for _, path := range mask.GetPaths() {
		switch path {
		case "topic":
			updateData["topic"] = project.Topic
		}
	}
	if len(updateData) > 0 {
		if err := r.db.WithContext(ctx).Model(&existingProject).Updates(updateData).Error; err != nil {
			return nil, err
		}
	}
	return &existingProject, nil
}

func (r *repository) DeleteProject(ctx context.Context, projectID int64) error {
	return r.db.WithContext(ctx).Delete(&Project{}, projectID).Error
}

func (r *repository) UpdateProjectStage(ctx context.Context, projectID, stageID int64) error {
	return r.db.WithContext(ctx).Model(&Project{}).Where("id = ?", projectID).Update("current_stage_id", stageID).Error
}
func (r *repository) CompleteProject(ctx context.Context, projectID int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&Project{}).Where("id = ?", projectID).Update("completed_at", &now).Error
}
