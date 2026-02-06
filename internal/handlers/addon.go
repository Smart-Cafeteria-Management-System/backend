package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smart-cafeteria/backend/internal/middleware"
	"github.com/smart-cafeteria/backend/internal/models"
)

// === Addon Management (Admin) ===

// GetAddons returns all available addons
func (h *Handler) GetAddons(c *gin.Context) {
	var addons []models.Addon
	query := h.DB.Order("category, name")

	// For non-admin, only show available addons
	role, _ := middleware.GetUserRole(c)
	if role != models.RoleAdmin {
		query = query.Where("available = ?", true)
	}

	if err := query.Find(&addons).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch addons"})
		return
	}

	c.JSON(http.StatusOK, addons)
}

// CreateAddonRequest represents addon creation payload
type CreateAddonRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	PointsCost  int    `json:"pointsCost"`
	Category    string `json:"category"`
	ImageURL    string `json:"imageUrl"`
	Available   bool   `json:"available"`
}

// CreateAddon creates a new addon (admin only)
func (h *Handler) CreateAddon(c *gin.Context) {
	var req CreateAddonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	addon := models.Addon{
		Name:        req.Name,
		Description: req.Description,
		PointsCost:  req.PointsCost,
		Category:    req.Category,
		ImageURL:    req.ImageURL,
		Available:   req.Available,
	}

	if addon.PointsCost == 0 {
		addon.PointsCost = 5 // Default
	}

	if err := h.DB.Create(&addon).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create addon"})
		return
	}

	c.JSON(http.StatusCreated, addon)
}

// UpdateAddon updates an existing addon (admin only)
func (h *Handler) UpdateAddon(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid addon ID"})
		return
	}

	var addon models.Addon
	if err := h.DB.First(&addon, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Addon not found"})
		return
	}

	var req CreateAddonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	addon.Name = req.Name
	addon.Description = req.Description
	addon.PointsCost = req.PointsCost
	addon.Category = req.Category
	addon.ImageURL = req.ImageURL
	addon.Available = req.Available

	if err := h.DB.Save(&addon).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update addon"})
		return
	}

	c.JSON(http.StatusOK, addon)
}

// DeleteAddon deletes an addon (admin only)
func (h *Handler) DeleteAddon(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid addon ID"})
		return
	}

	if err := h.DB.Delete(&models.Addon{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete addon"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Addon deleted successfully"})
}

// === Addon Redemption (User) ===

// generateRedemptionCode generates a random 6-character code
func generateRedemptionCode() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 6)
	for i := range code {
		code[i] = chars[rand.Intn(len(chars))]
	}
	return string(code)
}

// RedeemAddon allows a user to redeem an addon with points
func (h *Handler) RedeemAddon(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	addonID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid addon ID"})
		return
	}

	// Get addon
	var addon models.Addon
	if err := h.DB.First(&addon, "id = ? AND available = ?", addonID, true).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Addon not found or unavailable"})
		return
	}

	// Check user points
	var userPoints models.UserPoints
	if err := h.DB.Where("user_id = ?", userID).First(&userPoints).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No points balance found"})
		return
	}

	if userPoints.TotalPoints < addon.PointsCost {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Insufficient points",
			"required":       addon.PointsCost,
			"currentBalance": userPoints.TotalPoints,
		})
		return
	}

	tx := h.DB.Begin()

	// Deduct points
	userPoints.TotalPoints -= addon.PointsCost
	if err := tx.Save(&userPoints).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct points"})
		return
	}

	// Create point transaction (negative)
	transaction := models.PointTransaction{
		UserID: userID,
		Points: -addon.PointsCost,
		Type:   models.PointsRedemption,
		Reason: fmt.Sprintf("Redeemed: %s", addon.Name),
	}
	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record transaction"})
		return
	}

	// Create redemption record
	redemption := models.AddonRedemption{
		UserID:      userID,
		AddonID:     addonID,
		PointsSpent: addon.PointsCost,
		Status:      "pending",
		Code:        generateRedemptionCode(),
		ExpiresAt:   time.Now().Add(24 * time.Hour), // Valid for 24 hours
	}
	if err := tx.Create(&redemption).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create redemption"})
		return
	}

	tx.Commit()

	// Load addon details for response
	redemption.Addon = addon

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Addon redeemed successfully!",
		"redemption": redemption,
		"newBalance": userPoints.TotalPoints,
	})
}

// GetMyRedemptions returns user's addon redemptions
func (h *Handler) GetMyRedemptions(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var redemptions []models.AddonRedemption
	if err := h.DB.Preload("Addon").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(20).
		Find(&redemptions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch redemptions"})
		return
	}

	c.JSON(http.StatusOK, redemptions)
}

// ClaimRedemption marks a redemption as claimed (staff verifies code)
func (h *Handler) ClaimRedemption(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code is required"})
		return
	}

	var redemption models.AddonRedemption
	if err := h.DB.Preload("Addon").Preload("User").
		Where("code = ? AND status = ?", req.Code, "pending").
		First(&redemption).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid or expired redemption code"})
		return
	}

	// Check if expired
	if time.Now().After(redemption.ExpiresAt) {
		redemption.Status = "expired"
		h.DB.Save(&redemption)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Redemption code has expired"})
		return
	}

	// Mark as claimed
	now := time.Now()
	redemption.Status = "claimed"
	redemption.ClaimedAt = &now

	if err := h.DB.Save(&redemption).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to claim redemption"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Redemption claimed successfully!",
		"redemption": redemption,
	})
}
