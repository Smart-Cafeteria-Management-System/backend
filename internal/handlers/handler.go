package handlers

import (
	"gorm.io/gorm"
)

// Handler holds the database connection
type Handler struct {
	DB *gorm.DB
}

// New creates a new handler with database connection
func New(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}
