package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smart-cafeteria/backend/internal/middleware"
	"github.com/smart-cafeteria/backend/internal/models"
)

// BookingItemRequest represents a menu item in booking request
type BookingItemRequest struct {
	ItemID   string `json:"itemId" binding:"required"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

// CreateBookingRequest represents booking creation payload
type CreateBookingRequest struct {
	SlotID    string               `json:"slotId" binding:"required"`
	MenuItems []BookingItemRequest `json:"menuItems" binding:"required"`
}

// GetBookings returns all bookings (admin) or filtered bookings
func (h *Handler) GetBookings(c *gin.Context) {
	var bookings []models.Booking
	query := h.DB.Preload("User").Preload("Slot").Preload("Items")

	// Filter by status
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// Filter by date
	if date := c.Query("date"); date != "" {
		query = query.Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
			Where("meal_slots.date = ?", date)
	}

	if err := query.Order("created_at DESC").Find(&bookings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bookings"})
		return
	}

	c.JSON(http.StatusOK, bookings)
}

// GetMyBookings returns bookings for the authenticated user.
// Confirmed bookings from past slot dates are automatically treated as expired
// so they don't appear as "active" on the dashboard.
func (h *Handler) GetMyBookings(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	today := time.Now().Format("2006-01-02")

	// Auto-expire confirmed bookings whose slot date has already passed
	h.DB.Exec(`
		UPDATE bookings
		SET status = 'no-show', updated_at = NOW()
		WHERE user_id = ?
		  AND status = 'confirmed'
		  AND slot_id IN (
		      SELECT id FROM meal_slots WHERE date < ?
		  )
	`, userID, today)

	var bookings []models.Booking
	if err := h.DB.Preload("Slot").Preload("Items").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&bookings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bookings"})
		return
	}

	c.JSON(http.StatusOK, bookings)
}

// CreateBooking creates a new meal booking
func (h *Handler) CreateBooking(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check if user is blocked
	var user models.User
	if err := h.DB.First(&user, "id = ?", userID).Error; err == nil {
		if user.Blocked {
			c.JSON(http.StatusForbidden, gin.H{"error": "Your account has been blocked due to low incentive points. Please contact admin."})
			return
		}
	}

	var req CreateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slotID, err := uuid.Parse(req.SlotID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid slot ID"})
		return
	}

	// Check if slot exists and has capacity
	var slot models.MealSlot
	if err := h.DB.First(&slot, "id = ?", slotID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Slot not found"})
		return
	}

	if slot.Status == models.SlotFull || slot.BookedCount >= slot.Capacity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slot is full"})
		return
	}

	// Generate token number
	tokenNumber := generateTokenNumber(slot.MealType, slot.BookedCount+1)

	// Start transaction
	tx := h.DB.Begin()

	// Calculate preparation time for current items and fetch wait time from queue
	totalItemPrepTime := 0
	for _, reqItem := range req.MenuItems {
		var menuItem models.MenuItem
		if err := h.DB.First(&menuItem, "id = ?", reqItem.ItemID).Error; err == nil {
			qty := reqItem.Quantity
			if qty <= 0 {
				qty = 1
			}
			totalItemPrepTime += menuItem.PreparationTime * qty
		}
	}

	// Base wait time = (people ahead * 2 min) + current items prep time
	predictedWait := (slot.BookedCount * 2) + totalItemPrepTime
	if predictedWait < 2 {
		predictedWait = 2 // Minimum 2 minutes
	}

	// Create booking
	booking := models.Booking{
		UserID:            userID,
		SlotID:            slotID,
		TokenNumber:       tokenNumber,
		Status:            models.BookingConfirmed,
		PredictedWaitTime: predictedWait,
	}

	if err := tx.Create(&booking).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking"})
		return
	}

	// Create booking items
	for _, item := range req.MenuItems {
		itemID, err := uuid.Parse(item.ItemID)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid menu item ID"})
			return
		}

		quantity := item.Quantity
		if quantity <= 0 {
			quantity = 1
		}

		bookingItem := models.BookingItem{
			BookingID:  booking.ID,
			MenuItemID: itemID,
			ItemName:   item.Name,
			Quantity:   quantity,
		}

		if err := tx.Create(&bookingItem).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking item"})
			return
		}
	}

	// Create queue token
	queueToken := models.QueueToken{
		TokenNumber:       tokenNumber,
		BookingID:         booking.ID,
		UserID:            userID,
		Status:            models.TokenWaiting,
		EstimatedWaitTime: booking.PredictedWaitTime,
	}

	if err := tx.Create(&queueToken).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create queue token"})
		return
	}

	// Update slot booked count
	slot.BookedCount++
	if err := tx.Save(&slot).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update slot"})
		return
	}

	tx.Commit()

	// Reload booking with relationships
	h.DB.Preload("Slot").Preload("Items").Preload("QueueToken").First(&booking, "id = ?", booking.ID)

	c.JSON(http.StatusCreated, booking)
}

// UpdateBooking updates a booking
func (h *Handler) UpdateBooking(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	var booking models.Booking
	if err := h.DB.Preload("Slot").First(&booking, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Handle status update
	if status, ok := req["status"].(string); ok {
		oldStatus := booking.Status
		booking.Status = models.BookingStatus(status)

		if status == "served" && oldStatus != models.BookingServed {
			now := time.Now()
			booking.ServedAt = &now
			// Award attendance points
			h.AwardAttendancePoints(
				booking.UserID,
				booking.ID,
				booking.SlotID,
				booking.Slot.HasIncentive,
				booking.Slot.IncentivePoints,
			)
		}

		if status == "no-show" && oldStatus != models.BookingNoShow {
			// Record no-show penalty
			h.RecordNoShow(booking.UserID, booking.ID, booking.SlotID)
		}
	}

	if err := h.DB.Save(&booking).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update booking"})
		return
	}

	c.JSON(http.StatusOK, booking)
}

// CancelBooking cancels a booking
func (h *Handler) CancelBooking(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var booking models.Booking
	if err := h.DB.First(&booking, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	// Check ownership (unless admin)
	role, _ := middleware.GetUserRole(c)
	if booking.UserID != userID && role != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to cancel this booking"})
		return
	}

	if booking.Status != models.BookingConfirmed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can only cancel confirmed bookings"})
		return
	}

	tx := h.DB.Begin()

	// Update booking status
	booking.Status = models.BookingCancelled
	if err := tx.Save(&booking).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel booking"})
		return
	}

	// Update queue token
	tx.Model(&models.QueueToken{}).Where("booking_id = ?", id).Update("status", models.TokenExpired)

	// Decrease slot booked count
	tx.Model(&models.MealSlot{}).Where("id = ?", booking.SlotID).
		UpdateColumn("booked_count", h.DB.Raw("booked_count - 1"))

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Booking cancelled successfully"})
}

// generateTokenNumber creates a token number based on meal type
func generateTokenNumber(mealType models.MealType, sequence int) string {
	prefix := "B"
	switch mealType {
	case models.MealBreakfast:
		prefix = "B"
	case models.MealLunch:
		prefix = "L"
	case models.MealDinner:
		prefix = "D"
	}
	return fmt.Sprintf("%s%03d", prefix, sequence)
}
