package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Analytics represents daily analytics data
type Analytics struct {
	ID                 uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"_id"`
	Date               time.Time `gorm:"type:date;uniqueIndex;not null" json:"date"`
	TotalBookings      int       `gorm:"default:0" json:"totalBookings"`
	TotalServed        int       `gorm:"default:0" json:"totalServed"`
	AvgWaitTime        float64   `gorm:"type:decimal(5,2);default:0" json:"avgWaitTime"`
	PeakHour           *string   `gorm:"size:10" json:"peakHour,omitempty"`
	WastePercentage    float64   `gorm:"type:decimal(5,2);default:0" json:"wastePercentage"`
	SatisfactionScore  float64   `gorm:"type:decimal(3,2);default:4.0;check:satisfaction_score >= 0 AND satisfaction_score <= 5" json:"satisfactionScore"`
	BreakfastDemand    int       `gorm:"default:0" json:"breakfastDemand"`
	LunchDemand        int       `gorm:"default:0" json:"lunchDemand"`
	DinnerDemand       int       `gorm:"default:0" json:"dinnerDemand"`
	StaffEfficiency    *float64  `gorm:"type:decimal(5,2)" json:"staffEfficiency,omitempty"`
	CounterUtilization *float64  `gorm:"type:decimal(5,2)" json:"counterUtilization,omitempty"`
	FoodUtilization    *float64  `gorm:"type:decimal(5,2)" json:"foodUtilization,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// BeforeCreate hook
func (a *Analytics) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
