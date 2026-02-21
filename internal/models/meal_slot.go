package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MealType represents the type of meal
type MealType string

const (
	MealBreakfast MealType = "breakfast"
	MealLunch     MealType = "lunch"
	MealSnacks    MealType = "snacks"
	MealDinner    MealType = "dinner"
)

// SlotStatus represents meal slot status
type SlotStatus string

const (
	SlotAvailable SlotStatus = "available"
	SlotFull      SlotStatus = "full"
	SlotClosed    SlotStatus = "closed"
)

// MealSlot represents a bookable meal time slot
type MealSlot struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"_id"`
	Date            time.Time  `gorm:"type:date;not null" json:"date"`
	MealType        MealType   `gorm:"type:varchar(20);not null" json:"mealType"`
	StartTime       string     `gorm:"size:10;not null" json:"startTime"`
	EndTime         string     `gorm:"size:10;not null" json:"endTime"`
	Capacity        int        `gorm:"default:50;not null" json:"capacity"`
	BookedCount     int        `gorm:"default:0" json:"bookedCount"`
	Status          SlotStatus `gorm:"type:varchar(20);default:'available'" json:"status"`
	HasIncentive    bool       `gorm:"default:false" json:"hasIncentive"`
	IncentivePoints int        `gorm:"default:0" json:"incentivePoints"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`

	// Relationships
	Bookings []Booking `gorm:"foreignKey:SlotID" json:"bookings,omitempty"`
}

// BeforeCreate hook
func (s *MealSlot) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// BeforeSave hook to update status based on capacity
func (s *MealSlot) BeforeSave(tx *gorm.DB) error {
	if s.BookedCount >= s.Capacity {
		s.Status = SlotFull
	} else if s.Status == SlotFull && s.BookedCount < s.Capacity {
		s.Status = SlotAvailable
	}
	return nil
}
