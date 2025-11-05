package tenant

import (
	"context"
	"os"

	"github.com/rookie-ninja/rk-db/postgres"
	"gorm.io/gorm"
)

type Department struct {
	ID                 int64 `gorm:"primary_key"`
	Name               string
	WorkflowTemplateID int64
}
type TenantRepository struct {
	Db *gorm.DB
}

func NewTenantRepository() *TenantRepository {
	pgEntry := rkpostgres.GetPostgresEntry("tenant-conn")
	dbName := os.Getenv("DB_NAME")
	db := pgEntry.GetDB(dbName)
	db.AutoMigrate(&Department{})
	var count int64
	db.Model(&Department{}).Count(&count)
	if count == 0 {
		db.Create(&Department{ID: 1, Name: "IS", WorkflowTemplateID: 101})
	}
	return &TenantRepository{Db: db}
}

func (r *TenantRepository) GetTemplateID(ctx context.Context, departmentID int64) (int64, error) {
	var department Department
	res := r.Db.WithContext(ctx).First(&department, departmentID)
	return department.WorkflowTemplateID, res.Error
}
