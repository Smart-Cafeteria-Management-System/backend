package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ItemCategory represents menu item category
type ItemCategory string

const (
	CategoryMain     ItemCategory = "main"
	CategorySide     ItemCategory = "side"
	CategoryBeverage ItemCategory = "beverage"
	CategoryDessert  ItemCategory = "dessert"
)

// MenuItem represents a food item in the menu
type MenuItem struct {
	ID                 uuid.UUID    `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"_id"`
	Name               string       `gorm:"size:255;not null" json:"name"`
	Category           ItemCategory `gorm:"type:varchar(20);not null" json:"category"`
	Price              float64      `gorm:"type:decimal(10,2);not null" json:"price"`
	Calories           *int         `json:"calories,omitempty"`
	Protein            *int         `json:"protein,omitempty"`
	Carbs              *int         `json:"carbs,omitempty"`
	Fat                *int         `json:"fat,omitempty"`
	Available          bool         `gorm:"default:true" json:"available"`
	SustainabilityScore int         `gorm:"default:3;check:sustainability_score >= 1 AND sustainability_score <= 5" json:"sustainabilityScore"`
	PreparationTime    int          `gorm:"default:5" json:"preparationTime"`
	ImageURL           *string      `gorm:"size:500" json:"imageUrl,omitempty"`
	CreatedAt          time.Time    `json:"createdAt"`
	UpdatedAt          time.Time    `json:"updatedAt"`
}

// BeforeCreate hook
func (m *MenuItem) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
