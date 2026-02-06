package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IncentiveRule defines rules for incentivizing off-peak slots
type IncentiveRule struct {
	ID               uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name             string    `gorm:"size:100;not null" json:"name"`
	Description      string    `gorm:"type:text" json:"description"`
	SlotType         MealType  `gorm:"type:varchar(20)" json:"slotType"`  // breakfast, lunch, dinner, or empty for all
	MaxOccupancyPct  int       `gorm:"default:50" json:"maxOccupancyPct"` // slots below this % occupancy get incentive
	BonusPoints      int       `gorm:"default:10" json:"bonusPoints"`
	BaseAttendPoints int       `gorm:"default:5" json:"baseAttendPoints"` // base points for attending
	NoShowPenalty    int       `gorm:"default:10" json:"noShowPenalty"`   // points deducted for no-show
	IsActive         bool      `gorm:"default:true" json:"isActive"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// BeforeCreate hook
func (r *IncentiveRule) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// UserPoints tracks total points for a user
type UserPoints struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"userId"`
	TotalPoints int       `gorm:"default:0" json:"totalPoints"`
	UpdatedAt   time.Time `json:"updatedAt"`

	// Relationship
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// BeforeCreate hook
func (p *UserPoints) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// PointTransactionType represents the type of point transaction
type PointTransactionType string

const (
	PointsAttendance    PointTransactionType = "attendance"
	PointsBonus         PointTransactionType = "bonus_offpeak"
	PointsNoShowPenalty PointTransactionType = "penalty_noshow"
	PointsRedemption    PointTransactionType = "redemption"
	PointsManual        PointTransactionType = "manual"
)

// PointTransaction records individual point changes
type PointTransaction struct {
	ID        uuid.UUID            `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID            `gorm:"type:uuid;not null;index" json:"userId"`
	BookingID *uuid.UUID           `gorm:"type:uuid" json:"bookingId,omitempty"`
	Points    int                  `gorm:"not null" json:"points"` // positive = earned, negative = spent/penalty
	Type      PointTransactionType `gorm:"type:varchar(30);not null" json:"type"`
	Reason    string               `gorm:"size:255" json:"reason"`
	CreatedAt time.Time            `json:"createdAt"`

	// Relationships
	User    User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Booking *Booking `gorm:"foreignKey:BookingID" json:"booking,omitempty"`
}

// BeforeCreate hook
func (t *PointTransaction) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
