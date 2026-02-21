package models

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog records user activity for audit purposes
type AuditLog struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    *uuid.UUID `gorm:"type:uuid;index"                                 json:"userId,omitempty"`
	UserEmail string     `gorm:"size:255"                                         json:"userEmail"`
	Action    string     `gorm:"size:100;index"                                   json:"action"`
	Resource  string     `gorm:"size:100"                                         json:"resource"`
	Details   string     `gorm:"type:text"                                        json:"details"`
	IPAddress string     `gorm:"size:64"                                          json:"ipAddress"`
	UserAgent string     `gorm:"size:512"                                         json:"userAgent"`
	Success   bool       `json:"success"`
	CreatedAt time.Time  `gorm:"index"                                            json:"createdAt"`
}
