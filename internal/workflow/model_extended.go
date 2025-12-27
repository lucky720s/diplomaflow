package workflow

import (
	"time"

	"gorm.io/datatypes"
)

type ActionRegistryEntry struct {
	ID           string         `gorm:"primaryKey;type:varchar(50)"`
	Name         string         `gorm:"not null"`
	Description  string         `gorm:"type:text"`
	Category     string         `gorm:"type:varchar(50)"`
	ConfigSchema datatypes.JSON `gorm:"type:jsonb;not null"`
	IsSystem     bool           `gorm:"default:false"`
	IsEnabled    bool           `gorm:"default:true"`
	CreatedAt    time.Time
}

func (ActionRegistryEntry) TableName() string {
	return "action_registry"
}
