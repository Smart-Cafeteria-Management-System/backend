package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/smart-cafeteria/backend/internal/models"
)

// GetOperatingHours handles GET /api/operating-hours
func (h *Handler) GetOperatingHours(c *gin.Context) {
	var hours []models.OperatingHours
	if err := h.DB.Order("id asc").Find(&hours).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch operating hours"})
		return
	}

	c.JSON(http.StatusOK, hours)
}

// UpdateOperatingHours handles PUT /api/operating-hours/:id
func (h *Handler) UpdateOperatingHours(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		StartTime string `json:"startTime"`
		EndTime   string `json:"endTime"`
		IsClosed  bool   `json:"isClosed"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var hours models.OperatingHours
	if err := h.DB.First(&hours, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Operating hours record not found"})
		return
	}

	// Update fields
	hours.StartTime = input.StartTime
	hours.EndTime = input.EndTime
	hours.IsClosed = input.IsClosed

	if err := h.DB.Save(&hours).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update operating hours"})
		return
	}

	// Audit log (assuming there's a LogAction method on Handler or similar)
	// h.LogAction(c, "UPDATE_OPERATING_HOURS", "Updated hours for "+hours.DayOfWeek+" "+hours.MealType)

	c.JSON(http.StatusOK, hours)
}
