
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WeatherCondition for forecasting
type WeatherCondition string

const (
	WeatherSunny  WeatherCondition = "sunny"
	WeatherCloudy WeatherCondition = "cloudy"
	WeatherRainy  WeatherCondition = "rainy"
	WeatherStormy WeatherCondition = "stormy"
)

// AcademicSchedule for forecasting
type AcademicSchedule string

const (
	ScheduleRegular AcademicSchedule = "regular"
	ScheduleExams   AcademicSchedule = "exams"
	ScheduleHoliday AcademicSchedule = "holiday"
	ScheduleWeekend AcademicSchedule = "weekend"
)

// DayOfWeek for forecasting
type DayOfWeek string

const (
	Monday    DayOfWeek = "monday"
	Tuesday   DayOfWeek = "tuesday"
	Wednesday DayOfWeek = "wednesday"
	Thursday  DayOfWeek = "thursday"
	Friday    DayOfWeek = "friday"
	Saturday  DayOfWeek = "saturday"
	Sunday    DayOfWeek = "sunday"
)

// DemandForecast represents demand prediction data
type DemandForecast struct {
	ID               uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"_id"`
	Date             time.Time        `gorm:"type:date;not null" json:"date"`
	MealType         MealType         `gorm:"type:varchar(20);not null" json:"mealType"`
	PredictedDemand  int              `gorm:"not null" json:"predictedDemand"`
	ActualDemand     int              `gorm:"default:0" json:"actualDemand"`
	WeatherCondition WeatherCondition `gorm:"type:varchar(20);default:'sunny'" json:"weatherCondition"`
	AcademicSchedule AcademicSchedule `gorm:"type:varchar(20);default:'regular'" json:"academicSchedule"`
	DayOfWeek        DayOfWeek        `gorm:"type:varchar(20)" json:"dayOfWeek"`
	Confidence       int              `gorm:"default:75;check:confidence >= 0 AND confidence <= 100" json:"confidence"`
	CreatedAt        time.Time        `json:"createdAt"`
	UpdatedAt        time.Time        `json:"updatedAt"`
}

// BeforeCreate hook
func (d *DemandForecast) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
