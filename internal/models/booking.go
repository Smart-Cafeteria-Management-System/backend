package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BookingStatus represents booking status
type BookingStatus string

const (
	BookingConfirmed BookingStatus = "confirmed"
	BookingServed    BookingStatus = "served"
	BookingCancelled BookingStatus = "cancelled"
	BookingNoShow    BookingStatus = "no-show"
)

// Booking represents a meal booking
type Booking struct {
	ID                uuid.UUID     `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"_id"`
	UserID            uuid.UUID     `gorm:"type:uuid;not null" json:"userId"`
	SlotID            uuid.UUID     `gorm:"type:uuid;not null" json:"slotId"`
	TokenNumber       string        `gorm:"size:20;not null" json:"tokenNumber"`
	Status            BookingStatus `gorm:"type:varchar(20);default:'confirmed'" json:"status"`
	PredictedWaitTime int           `gorm:"default:0" json:"predictedWaitTime"`
	ServedAt          *time.Time    `json:"servedAt,omitempty"`
	CreatedAt         time.Time     `json:"createdAt"`
	UpdatedAt         time.Time     `json:"updatedAt"`

	// Relationships
	User        User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Slot        MealSlot      `gorm:"foreignKey:SlotID" json:"slot,omitempty"`
	Items       []BookingItem `gorm:"foreignKey:BookingID" json:"menuItems,omitempty"`
	QueueToken  *QueueToken   `gorm:"foreignKey:BookingID" json:"queueToken,omitempty"`
}

// BeforeCreate hook
func (b *Booking) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// BookingItem represents items in a booking
type BookingItem struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"_id"`
	BookingID  uuid.UUID `gorm:"type:uuid;not null" json:"bookingId"`
	MenuItemID uuid.UUID `gorm:"type:uuid;not null" json:"itemId"`
	ItemName   string    `gorm:"size:255;not null" json:"name"`
	Quantity   int       `gorm:"default:1" json:"quantity"`

	// Relationship
	MenuItem MenuItem `gorm:"foreignKey:MenuItemID" json:"menuItem,omitempty"`
}

// BeforeCreate hook
func (bi *BookingItem) BeforeCreate(tx *gorm.DB) error {
	if bi.ID == uuid.Nil {
		bi.ID = uuid.New()
	}
	return nil
}
