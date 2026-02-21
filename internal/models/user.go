package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserRole represents the role of a user
type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RoleStudent UserRole = "student"
	RoleStaff   UserRole = "staff"
)

// User represents a system user
type User struct {
	ID                  uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"_id"`
	Name                string    `gorm:"size:255;not null" json:"name"`
	Email               string    `gorm:"size:255;uniqueIndex;not null" json:"email"`
	Password            string    `gorm:"size:255;not null" json:"-"`
	Role                UserRole  `gorm:"type:varchar(20);default:'student'" json:"role"`
	StudentID           *string   `gorm:"size:50" json:"studentId,omitempty"`
	DietaryRestrictions string    `gorm:"type:text;default:'[]'" json:"dietaryRestrictions"`
	NotificationEnabled bool      `gorm:"default:true" json:"notificationEnabled"`
	Blocked             bool      `gorm:"default:false" json:"blocked"`
	TOTPSecret          string    `gorm:"size:64" json:"-"`
	TOTPEnabled         bool      `gorm:"default:false" json:"totpEnabled"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`

	// Relationships
	Bookings    []Booking    `gorm:"foreignKey:UserID" json:"bookings,omitempty"`
	QueueTokens []QueueToken `gorm:"foreignKey:UserID" json:"queueTokens,omitempty"`
}

// BeforeCreate hook to hash password
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return u.hashPassword()
}

// BeforeUpdate hook to hash password if changed
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	if tx.Statement.Changed("Password") {
		return u.hashPassword()
	}
	return nil
}

func (u *User) hashPassword() error {
	if len(u.Password) > 0 && len(u.Password) != 60 { // 60 is bcrypt hash length
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashedPassword)
	}
	return nil
}

// CheckPassword compares password with hash
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}
