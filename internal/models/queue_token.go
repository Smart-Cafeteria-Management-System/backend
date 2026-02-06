package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TokenStatus represents queue token status
type TokenStatus string

const (
	TokenWaiting TokenStatus = "waiting"
	TokenCalled  TokenStatus = "called"
	TokenServed  TokenStatus = "served"
	TokenExpired TokenStatus = "expired"
)

// QueueToken represents a token in the queue system
type QueueToken struct {
	ID                uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"_id"`
	TokenNumber       string      `gorm:"size:20;not null" json:"tokenNumber"`
	BookingID         uuid.UUID   `gorm:"type:uuid;not null;uniqueIndex" json:"bookingId"`
	UserID            uuid.UUID   `gorm:"type:uuid;not null" json:"userId"`
	Status            TokenStatus `gorm:"type:varchar(20);default:'waiting'" json:"status"`
	EstimatedWaitTime int         `gorm:"default:0" json:"estimatedWaitTime"`
	CalledAt          *time.Time  `json:"calledAt,omitempty"`
	CounterNumber     *int        `json:"counterNumber,omitempty"`
	CreatedAt         time.Time   `json:"createdAt"`
	UpdatedAt         time.Time   `json:"updatedAt"`

	// Relationships
	Booking Booking `gorm:"foreignKey:BookingID" json:"booking,omitempty"`
	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// BeforeCreate hook
func (q *QueueToken) BeforeCreate(tx *gorm.DB) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	return nil
}
