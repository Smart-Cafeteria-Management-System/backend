package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WasteCategory represents the type of food waste
type WasteCategory string

const (
	WastePrepared    WasteCategory = "prepared"     // Food prepared but not served
	WasteLeftover    WasteCategory = "leftover"     // Food left after serving
	WasteExpired     WasteCategory = "expired"      // Food expired before use
	WasteDamaged     WasteCategory = "damaged"      // Food damaged/spoiled
)

// WasteLog represents a food waste record
type WasteLog struct {
	ID               uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"_id"`
	Date             time.Time        `gorm:"type:date;not null" json:"date"`
	MealType         MealType         `gorm:"type:varchar(20);not null" json:"mealType"`
	Category         WasteCategory    `gorm:"type:varchar(20);not null" json:"category"`
	FoodItem         string           `gorm:"type:varchar(100);not null" json:"foodItem"`
	PreparedQuantity int              `gorm:"not null" json:"preparedQuantity"`        // Total prepared (in servings)
	WastedQuantity   int              `gorm:"not null" json:"wastedQuantity"`          // Wasted (in servings)
	WasteWeight      float64          `gorm:"type:decimal(10,2)" json:"wasteWeight"`   // Weight in kg (optional)
	Reason           string           `gorm:"type:varchar(255)" json:"reason"`
	RecordedBy       uuid.UUID        `gorm:"type:uuid;not null" json:"recordedBy"`
	WeatherCondition WeatherCondition `gorm:"type:varchar(20)" json:"weatherCondition"`
	AcademicSchedule AcademicSchedule `gorm:"type:varchar(20)" json:"academicSchedule"`
	Notes            string           `gorm:"type:text" json:"notes"`
	CreatedAt        time.Time        `json:"createdAt"`
	UpdatedAt        time.Time        `json:"updatedAt"`

	// Relations
	User User `gorm:"foreignKey:RecordedBy" json:"user,omitempty"`
}

// BeforeCreate hook
func (w *WasteLog) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

// WasteSummary represents aggregated waste statistics
type WasteSummary struct {
	Period            string  `json:"period"`
	TotalPrepared     int     `json:"totalPrepared"`
	TotalWasted       int     `json:"totalWasted"`
	WastePercentage   float64 `json:"wastePercentage"`
	TotalWasteWeight  float64 `json:"totalWasteWeight"`
	EstimatedCost     float64 `json:"estimatedCost"`      // Estimated cost of waste
	CO2Impact         float64 `json:"co2Impact"`          // CO2 in kg (waste * 2.5)
	ByMealType        map[string]float64 `json:"byMealType,omitempty"`
	ByCategory        map[string]float64 `json:"byCategory,omitempty"`
	TopWastedItems    []ItemWaste `json:"topWastedItems,omitempty"`
}

// ItemWaste represents waste for a specific item
type ItemWaste struct {
	FoodItem       string  `json:"foodItem"`
	TotalWasted    int     `json:"totalWasted"`
	WastePercent   float64 `json:"wastePercent"`
}
