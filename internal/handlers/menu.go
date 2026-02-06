package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smart-cafeteria/backend/internal/models"
)

// GetMenuItems returns all available menu items
func (h *Handler) GetMenuItems(c *gin.Context) {
	var items []models.MenuItem
	
	query := h.DB
	
	// Filter by category if provided
	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}
	
	// Filter by availability
	if available := c.Query("available"); available == "true" {
		query = query.Where("available = ?", true)
	}

	if err := query.Order("category, name").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch menu items"})
		return
	}

	// Transform to frontend expected format with nested nutritionInfo
	var response []gin.H
	for _, item := range items {
		calories := 0
		protein := 0
		carbs := 0
		fat := 0
		if item.Calories != nil {
			calories = *item.Calories
		}
		if item.Protein != nil {
			protein = *item.Protein
		}
		if item.Carbs != nil {
			carbs = *item.Carbs
		}
		if item.Fat != nil {
			fat = *item.Fat
		}

		response = append(response, gin.H{
			"_id":      item.ID,
			"name":     item.Name,
			"category": item.Category,
			"price":    item.Price,
			"nutritionInfo": gin.H{
				"calories": calories,
				"protein":  protein,
				"carbs":    carbs,
				"fat":      fat,
			},
			"available":           item.Available,
			"sustainabilityScore": item.SustainabilityScore,
			"preparationTime":     item.PreparationTime,
			"imageUrl":            item.ImageURL,
			"createdAt":           item.CreatedAt,
			"updatedAt":           item.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, response)
}

// NutritionInfo represents nested nutrition data from frontend
type NutritionInfo struct {
	Calories int `json:"calories"`
	Protein  int `json:"protein"`
	Carbs    int `json:"carbs"`
	Fat      int `json:"fat"`
}

// CreateMenuItemRequest represents menu item creation payload
type CreateMenuItemRequest struct {
	Name                string              `json:"name" binding:"required"`
	Category            models.ItemCategory `json:"category" binding:"required"`
	Price               float64             `json:"price" binding:"required"`
	NutritionInfo       *NutritionInfo      `json:"nutritionInfo"`
	SustainabilityScore int                 `json:"sustainabilityScore"`
	PreparationTime     int                 `json:"preparationTime"`
	ImageURL            *string             `json:"imageUrl"`
	Available           *bool               `json:"available"`
}

// CreateMenuItem creates a new menu item (admin only)
func (h *Handler) CreateMenuItem(c *gin.Context) {
	var req CreateMenuItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item := models.MenuItem{
		Name:                req.Name,
		Category:            req.Category,
		Price:               req.Price,
		Available:           true,
		SustainabilityScore: req.SustainabilityScore,
		PreparationTime:     req.PreparationTime,
		ImageURL:            req.ImageURL,
	}

	// Extract nutrition info from nested object
	if req.NutritionInfo != nil {
		item.Calories = &req.NutritionInfo.Calories
		item.Protein = &req.NutritionInfo.Protein
		item.Carbs = &req.NutritionInfo.Carbs
		item.Fat = &req.NutritionInfo.Fat
	}

	if item.SustainabilityScore == 0 {
		item.SustainabilityScore = 3
	}
	if item.PreparationTime == 0 {
		item.PreparationTime = 5
	}

	if err := h.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create menu item"})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// UpdateMenuItem updates an existing menu item (admin only)
func (h *Handler) UpdateMenuItem(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid menu item ID"})
		return
	}

	var item models.MenuItem
	if err := h.DB.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Menu item not found"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.DB.Model(&item).Updates(req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update menu item"})
		return
	}

	c.JSON(http.StatusOK, item)
}

// UpdateMenuAvailability toggles menu item availability (staff or admin)
func (h *Handler) UpdateMenuAvailability(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid menu item ID"})
		return
	}

	var item models.MenuItem
	if err := h.DB.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Menu item not found"})
		return
	}

	var req struct {
		Available bool `json:"available"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item.Available = req.Available
	if err := h.DB.Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update availability"})
		return
	}

	c.JSON(http.StatusOK, item)
}

// DeleteMenuItem deletes a menu item (admin only)
func (h *Handler) DeleteMenuItem(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid menu item ID"})
		return
	}

	result := h.DB.Delete(&models.MenuItem{}, "id = ?", id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete menu item"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Menu item not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Menu item deleted successfully"})
}

