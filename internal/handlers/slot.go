package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smart-cafeteria/backend/internal/models"
)

// GetTodaySlots returns meal slots for today
func (h *Handler) GetTodaySlots(c *gin.Context) {
	// Get today's date at midnight in local timezone
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.AddDate(0, 0, 1)

	var slots []models.MealSlot
	if err := h.DB.Where("date >= ? AND date < ?", todayStart, todayEnd).Order("start_time").Find(&slots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch slots"})
		return
	}

	c.JSON(http.StatusOK, slots)
}

// GetSlots returns meal slots with optional filters
func (h *Handler) GetSlots(c *gin.Context) {
	var slots []models.MealSlot
	query := h.DB

	// Filter by date range
	if startDate := c.Query("startDate"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("date >= ?", t)
		}
	}
	if endDate := c.Query("endDate"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			query = query.Where("date <= ?", t)
		}
	}

	// Filter by meal type
	if mealType := c.Query("mealType"); mealType != "" {
		query = query.Where("meal_type = ?", mealType)
	}

	// Filter by status
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("date, start_time").Find(&slots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch slots"})
		return
	}

	c.JSON(http.StatusOK, slots)
}

// CreateSlotRequest represents slot creation payload
type CreateSlotRequest struct {
	Date      string          `json:"date" binding:"required"`
	MealType  models.MealType `json:"mealType" binding:"required"`
	StartTime string          `json:"startTime" binding:"required"`
	EndTime   string          `json:"endTime" binding:"required"`
	Capacity  int             `json:"capacity"`
}

// CreateSlot creates a new meal slot (admin only)
func (h *Handler) CreateSlot(c *gin.Context) {
	var req CreateSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse date in local timezone
	loc := time.Now().Location()
	date, err := time.ParseInLocation("2006-01-02", req.Date, loc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format, use YYYY-MM-DD"})
		return
	}

	slot := models.MealSlot{
		Date:      date,
		MealType:  req.MealType,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Capacity:  req.Capacity,
		Status:    models.SlotAvailable,
	}

	if slot.Capacity == 0 {
		slot.Capacity = 50
	}

	if err := h.DB.Create(&slot).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create slot"})
		return
	}

	c.JSON(http.StatusCreated, slot)
}

// UpdateSlot updates an existing meal slot (admin only)
func (h *Handler) UpdateSlot(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid slot ID"})
		return
	}

	var slot models.MealSlot
	if err := h.DB.First(&slot, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Slot not found"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.DB.Model(&slot).Updates(req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update slot"})
		return
	}

	c.JSON(http.StatusOK, slot)
}

// DeleteSlot deletes a meal slot (admin only)
func (h *Handler) DeleteSlot(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid slot ID"})
		return
	}

	result := h.DB.Delete(&models.MealSlot{}, "id = ?", id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete slot"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Slot not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Slot deleted successfully"})
}

// GenerateSlotsRequest represents bulk slot generation payload
type GenerateSlotsRequest struct {
	StartDate string `json:"startDate" binding:"required"`
	EndDate   string `json:"endDate" binding:"required"`
	Capacity  int    `json:"capacity"`
}

// GenerateSlots creates slots for a date range (admin only)
func (h *Handler) GenerateSlots(c *gin.Context) {
	var req GenerateSlotsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse dates in local timezone
	loc := time.Now().Location()
	startDate, err := time.ParseInLocation("2006-01-02", req.StartDate, loc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
		return
	}

	endDate, err := time.ParseInLocation("2006-01-02", req.EndDate, loc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format"})
		return
	}

	capacity := req.Capacity
	useMLForecasts := capacity == 0 // If no capacity specified, use ML forecasts

	// Load forecasts for this date range if using ML
	forecastMap := make(map[string]int)
	if useMLForecasts {
		var forecasts []models.DemandForecast
		h.DB.Where("date >= ? AND date <= ?", startDate, endDate).Find(&forecasts)

		for _, f := range forecasts {
			key := f.Date.Format("2006-01-02") + "_" + string(f.MealType)
			// Use 120% of predicted demand as capacity to provide buffer
			forecastMap[key] = int(float64(f.PredictedDemand) * 1.2)
		}
	}

	// Define standard slots
	slotTemplates := []struct {
		MealType  models.MealType
		StartTime string
		EndTime   string
	}{
		{models.MealBreakfast, "07:00", "09:00"},
		{models.MealLunch, "12:00", "14:00"},
		{models.MealDinner, "18:00", "20:00"},
	}

	var createdSlots []models.MealSlot
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		for _, template := range slotTemplates {
			// Check if slot already exists
			var existing models.MealSlot
			if h.DB.Where("date = ? AND meal_type = ?", d, template.MealType).First(&existing).Error == nil {
				continue // Skip if exists
			}

			// Determine capacity
			slotCapacity := capacity
			if useMLForecasts {
				key := d.Format("2006-01-02") + "_" + string(template.MealType)
				if forecastCapacity, exists := forecastMap[key]; exists && forecastCapacity > 0 {
					slotCapacity = forecastCapacity
				} else {
					// No forecast found, use default
					slotCapacity = 50
				}
			}

			// Ensure minimum capacity
			if slotCapacity == 0 {
				slotCapacity = 50
			}

			slot := models.MealSlot{
				Date:      d,
				MealType:  template.MealType,
				StartTime: template.StartTime,
				EndTime:   template.EndTime,
				Capacity:  slotCapacity,
				Status:    models.SlotAvailable,
			}
			if err := h.DB.Create(&slot).Error; err == nil {
				createdSlots = append(createdSlots, slot)
			}
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "Slots generated successfully",
		"count":         len(createdSlots),
		"usedForecasts": useMLForecasts,
		"slots":         createdSlots,
	})
}
