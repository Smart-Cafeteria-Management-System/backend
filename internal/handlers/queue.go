package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smart-cafeteria/backend/internal/middleware"
	"github.com/smart-cafeteria/backend/internal/models"
)

// QueueStatusResponse represents queue status
type QueueStatusResponse struct {
	CurrentlyServing *models.QueueToken  `json:"currentlyServing"`
	WaitingCount     int64               `json:"waitingCount"`
	AvgWaitTime      int                 `json:"avgWaitTime"`
	WaitingTokens    []models.QueueToken `json:"waitingTokens"`
	RecentlyCalled   []models.QueueToken `json:"recentlyCalled"`
}

// GetQueueStatus returns the current queue status
func (h *Handler) GetQueueStatus(c *gin.Context) {
	today := time.Now().Truncate(24 * time.Hour)

	// Fetch waiting tokens
	var waitingTokens []models.QueueToken
	if err := h.DB.Preload("User").Preload("Booking").
		Joins("JOIN bookings ON bookings.id = queue_tokens.booking_id").
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date = ?", today).
		Where("queue_tokens.status = ?", "waiting").
		Order("queue_tokens.created_at").
		Find(&waitingTokens).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch queue status"})
		return
	}

	// Fetch recently called tokens (called but not yet served)
	var recentlyCalled []models.QueueToken
	if err := h.DB.Preload("User").Preload("Booking").
		Joins("JOIN bookings ON bookings.id = queue_tokens.booking_id").
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date = ?", today).
		Where("queue_tokens.status = ?", "called").
		Order("queue_tokens.called_at DESC").
		Find(&recentlyCalled).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch called tokens"})
		return
	}

	waitingCount := int64(len(waitingTokens))

	var currentlyServing *models.QueueToken
	if len(recentlyCalled) > 0 {
		currentlyServing = &recentlyCalled[0]
	}

	response := QueueStatusResponse{
		CurrentlyServing: currentlyServing,
		WaitingCount:     waitingCount,
		AvgWaitTime:      int(waitingCount) * 2, // 2 min per person estimate
		WaitingTokens:    waitingTokens,
		RecentlyCalled:   recentlyCalled,
	}

	c.JSON(http.StatusOK, response)
}

// GetMyToken returns the current user's queue token
func (h *Handler) GetMyToken(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	today := time.Now().Truncate(24 * time.Hour)

	var token models.QueueToken
	err := h.DB.Preload("Booking").Preload("Booking.Slot").
		Joins("JOIN bookings ON bookings.id = queue_tokens.booking_id").
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("queue_tokens.user_id = ?", userID).
		Where("meal_slots.date = ?", today).
		Where("queue_tokens.status IN ?", []string{"waiting", "called"}).
		First(&token).Error

	if err != nil {
		// No active token found
		c.JSON(http.StatusOK, gin.H{"token": nil, "position": 0, "estimatedWait": 0})
		return
	}

	// Calculate position in queue
	var position int64
	h.DB.Model(&models.QueueToken{}).
		Joins("JOIN bookings ON bookings.id = queue_tokens.booking_id").
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date = ?", today).
		Where("queue_tokens.status = ?", "waiting").
		Where("queue_tokens.created_at < ?", token.CreatedAt).
		Count(&position)

	c.JSON(http.StatusOK, gin.H{
		"token":         token,
		"position":      position + 1,
		"estimatedWait": (position + 1) * 2, // 2 min per person
	})
}

// GetQueueHistory returns the queue history for today
func (h *Handler) GetQueueHistory(c *gin.Context) {
	today := time.Now().Truncate(24 * time.Hour)

	var tokens []models.QueueToken
	if err := h.DB.Preload("User").Preload("Booking").
		Joins("JOIN bookings ON bookings.id = queue_tokens.booking_id").
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date = ?", today).
		Order("queue_tokens.created_at DESC").
		Limit(50).
		Find(&tokens).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch queue history"})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// CallNextToken calls the next token in queue (admin only)
func (h *Handler) CallNextToken(c *gin.Context) {
	today := time.Now().Truncate(24 * time.Hour)

	// Mark any currently called tokens as waiting (reset)
	h.DB.Model(&models.QueueToken{}).
		Joins("JOIN bookings ON bookings.id = queue_tokens.booking_id").
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date = ?", today).
		Where("queue_tokens.status = ?", "called").
		Update("status", "waiting")

	// Get the next waiting token
	var token models.QueueToken
	if err := h.DB.Preload("User").Preload("Booking.Items").
		Joins("JOIN bookings ON bookings.id = queue_tokens.booking_id").
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date = ?", today).
		Where("queue_tokens.status = ?", "waiting").
		Order("queue_tokens.created_at").
		First(&token).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No waiting tokens in queue"})
		return
	}

	// Update token status
	now := time.Now()
	token.Status = models.TokenCalled
	token.CalledAt = &now

	// Get counter number from request if provided
	var req struct {
		CounterNumber int `json:"counterNumber"`
	}
	if err := c.ShouldBindJSON(&req); err == nil && req.CounterNumber > 0 {
		token.CounterNumber = &req.CounterNumber
	}

	if err := h.DB.Save(&token).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update token"})
		return
	}

	c.JSON(http.StatusOK, token)
}

// ServeToken marks a token as served (admin only)
func (h *Handler) ServeToken(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID"})
		return
	}

	var token models.QueueToken
	if err := h.DB.First(&token, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	if token.Status != models.TokenCalled && token.Status != models.TokenWaiting {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token cannot be served"})
		return
	}

	tx := h.DB.Begin()

	// Update token
	token.Status = models.TokenServed
	if err := tx.Save(&token).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update token"})
		return
	}

	// Update booking
	now := time.Now()
	tx.Model(&models.Booking{}).Where("id = ?", token.BookingID).Updates(map[string]interface{}{
		"status":    models.BookingServed,
		"served_at": now,
	})

	tx.Commit()

	c.JSON(http.StatusOK, token)
}

