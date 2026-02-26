package models

import (
	"time"

	"gorm.io/gorm"
)

// OperatingHours Represents the specific operating window for a meal type on a given day.
// This supports US-AM-4 (Configure Operating Hours).
type OperatingHours struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	DayOfWeek string         `gorm:"type:varchar(20);not null;uniqueIndex:idx_day_meal" json:"dayOfWeek"` // e.g., "Monday"
	MealType  MealType       `gorm:"type:varchar(20);not null;uniqueIndex:idx_day_meal" json:"mealType"`  // e.g., "Breakfast", "Lunch", "Dinner"
	StartTime string         `gorm:"type:varchar(8);not null" json:"startTime"`                          // e.g., "08:00"
	EndTime   string         `gorm:"type:varchar(8);not null" json:"endTime"`                            // e.g., "10:30"
	IsClosed  bool           `gorm:"default:false" json:"isClosed"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
