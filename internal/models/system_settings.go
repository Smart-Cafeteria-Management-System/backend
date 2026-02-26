package models

import (
	"time"

	"gorm.io/gorm"
)

// SystemSettings represents global configuration parameters
// This supports US-AM-6 (Manage System Settings)
type SystemSettings struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	MaintenanceMode    bool           `gorm:"default:false" json:"maintenanceMode"`
	SustainabilityGoal float64        `gorm:"default:85.0" json:"sustainabilityGoal"` // Target score
	IncentiveMultiplier float64       `gorm:"default:1.0" json:"incentiveMultiplier"`
	MaxBookingsPerUser int            `gorm:"default:3" json:"maxBookingsPerUser"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}
