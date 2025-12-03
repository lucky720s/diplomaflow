package role

import "gorm.io/gorm"

type Role struct {
	ID           int64          `gorm:"primaryKey"`
	Name         string         `gorm:"not null"`
	DepartmentID int64          `gorm:"index"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}
