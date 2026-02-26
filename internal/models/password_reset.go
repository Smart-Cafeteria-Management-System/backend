package models

import (
	"time"

	"github.com/google/uuid"
)

// PasswordReset stores password reset tokens
type PasswordReset struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;index;not null"                        json:"userId"`
	Token     string    `gorm:"size:255;uniqueIndex;not null"                   json:"token"`
	Used      bool      `gorm:"default:false"                                   json:"used"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

// IsExpired returns true if the reset token has passed its expiry time
func (p *PasswordReset) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}
