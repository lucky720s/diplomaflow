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
	Supervisor   int64
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
		Supervisor:   supervisorID,
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
