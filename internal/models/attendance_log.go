package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AttendanceStatus represents the attendance outcome
type AttendanceStatus string

const (
	AttendanceAttended  AttendanceStatus = "attended"
	AttendanceNoShow    AttendanceStatus = "no-show"
	AttendanceLate      AttendanceStatus = "late"
	AttendanceCancelled AttendanceStatus = "cancelled"
)

// AttendanceLog records attendance behavior for analysis
type AttendanceLog struct {
	ID            uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"_id"`
	UserID        uuid.UUID        `gorm:"type:uuid;not null;index" json:"userId"`
	BookingID     uuid.UUID        `gorm:"type:uuid;not null;uniqueIndex" json:"bookingId"`
	SlotID        uuid.UUID        `gorm:"type:uuid;not null" json:"slotId"`
	Status        AttendanceStatus `gorm:"type:varchar(20);not null" json:"status"`
	PointsAwarded int              `gorm:"default:0" json:"pointsAwarded"`
	CheckedAt     *time.Time       `json:"checkedAt,omitempty"`
	CreatedAt     time.Time        `json:"createdAt"`

	// Relationships
	User    User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Booking Booking  `gorm:"foreignKey:BookingID" json:"booking,omitempty"`
	Slot    MealSlot `gorm:"foreignKey:SlotID" json:"slot,omitempty"`
}

// BeforeCreate hook
func (a *AttendanceLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
