package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Addon represents a free add-on item that can be redeemed with points
type Addon struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	Description string    `gorm:"size:255" json:"description"`
	PointsCost  int       `gorm:"not null;default:5" json:"pointsCost"`
	Available   bool      `gorm:"default:true" json:"available"`
	ImageURL    string    `gorm:"size:255" json:"imageUrl,omitempty"`
	Category    string    `gorm:"size:50" json:"category"` // e.g., "beverage", "snack", "dessert"
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// AddonRedemption represents a user's redemption of an addon
type AddonRedemption struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null" json:"userId"`
	AddonID     uuid.UUID  `gorm:"type:uuid;not null" json:"addonId"`
	PointsSpent int        `gorm:"not null" json:"pointsSpent"`
	Status      string     `gorm:"type:varchar(20);default:'pending'" json:"status"` // pending, claimed, expired
	Code        string     `gorm:"size:10;not null" json:"code"`                     // Redemption code to show staff
	ExpiresAt   time.Time  `json:"expiresAt"`
	ClaimedAt   *time.Time `json:"claimedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`

	// Relationships
	User  User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Addon Addon `gorm:"foreignKey:AddonID" json:"addon,omitempty"`
}

// BeforeCreate hook for Addon
func (a *Addon) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook for AddonRedemption
func (r *AddonRedemption) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
