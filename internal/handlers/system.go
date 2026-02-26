package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/smart-cafeteria/backend/internal/models"
)

// GetSystemSettings handles GET /api/system/settings
func (h *Handler) GetSystemSettings(c *gin.Context) {
	var settings models.SystemSettings
	if err := h.DB.First(&settings).Error; err != nil {
		// If no settings exist, create default
		settings = models.SystemSettings{
			SustainabilityGoal: 85.0,
			IncentiveMultiplier: 1.0,
			MaxBookingsPerUser: 3,
		}
		h.DB.Create(&settings)
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateSystemSettings handles PUT /api/system/settings
func (h *Handler) UpdateSystemSettings(c *gin.Context) {
	var input models.SystemSettings
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var settings models.SystemSettings
	if err := h.DB.First(&settings).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Settings not found"})
		return
	}

	// Update fields
	settings.MaintenanceMode = input.MaintenanceMode
	settings.SustainabilityGoal = input.SustainabilityGoal
	settings.IncentiveMultiplier = input.IncentiveMultiplier
	settings.MaxBookingsPerUser = input.MaxBookingsPerUser

	if err := h.DB.Save(&settings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}
