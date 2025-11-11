package project

import (
	"context"
	"os"

	rkpostgres "github.com/rookie-ninja/rk-db/postgres"
	"gorm.io/gorm"
)

type Project struct {
	ID           int64 `gorm:"primaryKey"`
	Topic        string
	SupervisorID int64
	DepartmentID int64
}

type ProjectMember struct {
	ID        uint  `gorm:"primaryKey"`
	ProjectID int64 `gorm:"index"`
	UserID    int64 `gorm:"index"`
}
type ProjectRepository struct {
	Db *gorm.DB
}

func NewProjectRepository() *ProjectRepository {
	pgEntry := rkpostgres.GetPostgresEntry("project-conn")
	dbName := os.Getenv("PROJECT_DB_NAME")
	db := pgEntry.GetDB(dbName)
	db.AutoMigrate(&Project{}, &ProjectMember{})
	return &ProjectRepository{Db: db}
}
func (r *ProjectRepository) CreateProject(ctx context.Context, topic string, supervisorID int64, studentIDs []int64, departmentID int64) (*Project, error) {
	project := &Project{
		Topic:        topic,
		SupervisorID: supervisorID,
		DepartmentID: departmentID,
	}
	err := r.Db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(project).Error; err != nil {
			return err
		}
		if len(studentIDs) > 0 {
			var members []*ProjectMember
			for _, studentID := range studentIDs {
				members = append(members, &ProjectMember{
					ProjectID: project.ID,
					UserID:    studentID,
				})
			}
			if err := tx.Create(members).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return project, err
}

func (r *ProjectRepository) GetProjectByID(ctx context.Context, projectID int64) (*Project, []int64, error) {
	var project Project
	if err := r.Db.WithContext(ctx).First(&project, projectID).Error; err != nil {
		return nil, nil, err
	}
	var members []*ProjectMember
	if err := r.Db.WithContext(ctx).Where("project_id = ?", project.ID).Find(&members).Error; err != nil {
		return nil, nil, err
	}
	var studentIDs []int64
	for _, member := range members {
		studentIDs = append(studentIDs, member.UserID)
	}
	return &project, studentIDs, nil
}

func (r *ProjectRepository) UpdateProject(ctx context.Context, projectID int64, topic string) (*Project, error) {
	var project Project
	if err := r.Db.WithContext(ctx).First(&project, projectID).Error; err != nil {
		return nil, err
	}
	project.Topic = topic
	err := r.Db.WithContext(ctx).Save(&project).Error
	return &project, err
}

func (r *ProjectRepository) DeleteProject(ctx context.Context, projectID int64) error {
	return r.Db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", projectID).Delete(&ProjectMember{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&ProjectMember{ProjectID: projectID}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&Project{ID: projectID}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *ProjectRepository) ListProjects(ctx context.Context) ([]Project, error) {
	var projects []Project
	err := r.Db.WithContext(ctx).Find(&projects).Error
	return projects, err
}
