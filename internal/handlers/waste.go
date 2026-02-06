package handlers

import (
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smart-cafeteria/backend/internal/middleware"
	"github.com/smart-cafeteria/backend/internal/models"
)

// CreateWasteLogRequest represents waste log creation payload
type CreateWasteLogRequest struct {
	Date             string                  `json:"date" binding:"required"`
	MealType         models.MealType         `json:"mealType" binding:"required"`
	Category         models.WasteCategory    `json:"category" binding:"required"`
	FoodItem         string                  `json:"foodItem" binding:"required"`
	PreparedQuantity int                     `json:"preparedQuantity" binding:"required,min=0"`
	WastedQuantity   int                     `json:"wastedQuantity" binding:"required,min=0"`
	WasteWeight      float64                 `json:"wasteWeight"`
	Reason           string                  `json:"reason"`
	WeatherCondition models.WeatherCondition `json:"weatherCondition"`
	AcademicSchedule models.AcademicSchedule `json:"academicSchedule"`
	Notes            string                  `json:"notes"`
}

// GetWasteLogs returns waste logs with optional filters
func (h *Handler) GetWasteLogs(c *gin.Context) {
	var logs []models.WasteLog
	query := h.DB.Preload("User")

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

	// Filter by category
	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}

	if err := query.Order("date DESC, meal_type").Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch waste logs"})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// CreateWasteLog creates a new waste log entry
func (h *Handler) CreateWasteLog(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CreateWasteLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
		return
	}

	wasteLog := models.WasteLog{
		Date:             date,
		MealType:         req.MealType,
		Category:         req.Category,
		FoodItem:         req.FoodItem,
		PreparedQuantity: req.PreparedQuantity,
		WastedQuantity:   req.WastedQuantity,
		WasteWeight:      req.WasteWeight,
		Reason:           req.Reason,
		RecordedBy:       userID,
		WeatherCondition: req.WeatherCondition,
		AcademicSchedule: req.AcademicSchedule,
		Notes:            req.Notes,
	}

	if err := h.DB.Create(&wasteLog).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create waste log"})
		return
	}

	// Load user relationship
	h.DB.Preload("User").First(&wasteLog, "id = ?", wasteLog.ID)

	c.JSON(http.StatusCreated, wasteLog)
}

// UpdateWasteLog updates an existing waste log
func (h *Handler) UpdateWasteLog(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid waste log ID"})
		return
	}

	var wasteLog models.WasteLog
	if err := h.DB.First(&wasteLog, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Waste log not found"})
		return
	}

	var req CreateWasteLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
		return
	}

	// Update fields
	wasteLog.Date = date
	wasteLog.MealType = req.MealType
	wasteLog.Category = req.Category
	wasteLog.FoodItem = req.FoodItem
	wasteLog.PreparedQuantity = req.PreparedQuantity
	wasteLog.WastedQuantity = req.WastedQuantity
	wasteLog.WasteWeight = req.WasteWeight
	wasteLog.Reason = req.Reason
	wasteLog.WeatherCondition = req.WeatherCondition
	wasteLog.AcademicSchedule = req.AcademicSchedule
	wasteLog.Notes = req.Notes

	if err := h.DB.Save(&wasteLog).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update waste log"})
		return
	}

	c.JSON(http.StatusOK, wasteLog)
}

// DeleteWasteLog deletes a waste log entry
func (h *Handler) DeleteWasteLog(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid waste log ID"})
		return
	}

	var wasteLog models.WasteLog
	if err := h.DB.First(&wasteLog, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Waste log not found"})
		return
	}

	if err := h.DB.Delete(&wasteLog).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete waste log"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Waste log deleted successfully"})
}

// GetWasteSummary returns aggregated waste statistics
func (h *Handler) GetWasteSummary(c *gin.Context) {
	// Default to last 30 days
	days := 30
	if d := c.Query("days"); d != "" {
		// Parse days parameter
		var parsed int
		if _, err := time.ParseDuration(d + "h"); err == nil {
			// If it parses as duration, convert to days
			days = 30 // fallback
		}
		if parsed > 0 {
			days = parsed
		}
	}

	startDate := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)
	endDate := time.Now()

	var logs []models.WasteLog
	if err := h.DB.Where("date >= ?", startDate).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch waste logs"})
		return
	}

	// Calculate summary statistics
	var totalPrepared, totalWasted int
	var totalWeight float64
	byMealType := make(map[string]float64)
	byCategory := make(map[string]float64)
	itemWaste := make(map[string]struct {
		prepared int
		wasted   int
	})

	for _, log := range logs {
		totalPrepared += log.PreparedQuantity
		totalWasted += log.WastedQuantity
		totalWeight += log.WasteWeight

		byMealType[string(log.MealType)] += float64(log.WastedQuantity)
		byCategory[string(log.Category)] += float64(log.WastedQuantity)

		item := itemWaste[log.FoodItem]
		item.prepared += log.PreparedQuantity
		item.wasted += log.WastedQuantity
		itemWaste[log.FoodItem] = item
	}

	// Calculate waste percentage
	wastePercentage := 0.0
	if totalPrepared > 0 {
		wastePercentage = float64(totalWasted) / float64(totalPrepared) * 100
	}

	// Calculate top wasted items
	var topWasted []models.ItemWaste
	for item, data := range itemWaste {
		percent := 0.0
		if data.prepared > 0 {
			percent = float64(data.wasted) / float64(data.prepared) * 100
		}
		topWasted = append(topWasted, models.ItemWaste{
			FoodItem:     item,
			TotalWasted:  data.wasted,
			WastePercent: math.Round(percent*100) / 100,
		})
	}

	// Sort by total wasted (simple bubble sort for small list)
	for i := 0; i < len(topWasted); i++ {
		for j := i + 1; j < len(topWasted); j++ {
			if topWasted[j].TotalWasted > topWasted[i].TotalWasted {
				topWasted[i], topWasted[j] = topWasted[j], topWasted[i]
			}
		}
	}

	// Limit to top 10
	if len(topWasted) > 10 {
		topWasted = topWasted[:10]
	}

	// Cost estimation (assuming $5 per unit average)
	estimatedCost := float64(totalWasted) * 5.0
	// CO2 impact (2.5 kg CO2 per kg food waste)
	co2Impact := totalWeight * 2.5

	summary := models.WasteSummary{
		Period:           startDate.Format("2006-01-02") + " to " + endDate.Format("2006-01-02"),
		TotalPrepared:    totalPrepared,
		TotalWasted:      totalWasted,
		WastePercentage:  math.Round(wastePercentage*100) / 100,
		TotalWasteWeight: math.Round(totalWeight*100) / 100,
		EstimatedCost:    math.Round(estimatedCost*100) / 100,
		CO2Impact:        math.Round(co2Impact*100) / 100,
		ByMealType:       byMealType,
		ByCategory:       byCategory,
		TopWastedItems:   topWasted,
	}

	c.JSON(http.StatusOK, summary)
}
