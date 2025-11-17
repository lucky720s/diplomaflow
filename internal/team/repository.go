package team

import (
	"context"
	"fmt"
	"os"

	rkpostgres "github.com/rookie-ninja/rk-db/postgres"
	"gorm.io/gorm"
)

type Team struct {
	ID           int64 `gorm:"primaryKey"`
	Name         string
	DepartmentID int64 `gorm:"index"`
}

type TeamMember struct {
	ID     int64 `gorm:"primaryKey"`
	TeamID int64 `gorm:"index"`
	UserID int64 `gorm:"index"`
}
type Repository interface {
	CreateTeam(ctx context.Context, name string, departmentID int64, memberIDs []int64) (*Team, error)
	GetTeamByID(ctx context.Context, teamID int64) (*Team, []int64, error)
	AddMember(ctx context.Context, teamID int64, userID int64) error
	RemoveMember(ctx context.Context, teamID int64, userID int64) error
	ListTeams(ctx context.Context, departmentID int64) ([]*Team, error)
	DeleteTeam(ctx context.Context, teamID int64) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository() (Repository, error) {
	pgEntry := rkpostgres.GetPostgresEntry("team-conn")
	dbName := os.Getenv("TEAM_DB_NAME")
	db := pgEntry.GetDB(dbName)
	if db == nil {
		panic("Database not found")
	}
	if err := db.AutoMigrate(&Team{}, &TeamMember{}); err != nil {
		return nil, fmt.Errorf("auto migrate team err: %w", err)
	}
	return &repository{db: db}, nil
}
func (r *repository) CreateTeam(ctx context.Context, name string, departmentID int64, memberIDs []int64) (*Team, error) {
	team := &Team{Name: name, DepartmentID: departmentID}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(team).Error; err != nil {
			return err
		}
		if len(memberIDs) > 0 {
			var members []*TeamMember
			for _, id := range memberIDs {
				members = append(members, &TeamMember{TeamID: team.ID, UserID: id})
			}
			if err := tx.Create(&members).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return team, err
}

func (r *repository) GetTeamByID(ctx context.Context, teamID int64) (*Team, []int64, error) {
	var team Team
	if err := r.db.WithContext(ctx).First(&team, teamID).Error; err != nil {
		return nil, nil, err
	}
	var members []*TeamMember
	if err := r.db.WithContext(ctx).Where("team_id = ?", teamID).Find(&members).Error; err != nil {
		return nil, nil, err
	}
	var memberIDs []int64
	for _, member := range members {
		memberIDs = append(memberIDs, member.UserID)
	}
	return &team, memberIDs, nil
}
func (r *repository) AddMember(ctx context.Context, teamID int64, userID int64) error {
	member := &TeamMember{TeamID: teamID, UserID: userID}
	return r.db.WithContext(ctx).Create(member).Error
}
func (r *repository) RemoveMember(ctx context.Context, teamID int64, userID int64) error {
	return r.db.WithContext(ctx).Delete(&TeamMember{TeamID: teamID, UserID: userID}).Error
}
func (r *repository) ListTeams(ctx context.Context, departmentID int64) ([]*Team, error) {
	var teams []*Team
	err := r.db.WithContext(ctx).Where("department_id = ?", departmentID).Find(&teams).Error
	return teams, err
}
func (r *repository) DeleteTeam(ctx context.Context, teamID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("team_id = ?", teamID).Delete(&TeamMember{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&Team{}, teamID).Error; err != nil {
			return err
		}
		return nil
	})
}
