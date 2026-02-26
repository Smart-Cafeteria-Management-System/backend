package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/smart-cafeteria/backend/internal/middleware"
	"github.com/smart-cafeteria/backend/internal/models"
)

// GetAllUsers returns all users (admin only)
func (h *Handler) GetAllUsers(c *gin.Context) {
	var users []models.User
	if err := h.DB.Order("created_at DESC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetCurrentUser returns the authenticated user's profile
func (h *Handler) GetCurrentUser(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUserRequest represents update user payload
type UpdateUserRequest struct {
	Name                string  `json:"name"`
	DietaryRestrictions *string `json:"dietaryRestrictions"`
	NotificationEnabled *bool   `json:"notificationEnabled"`
}

// UpdateCurrentUser updates the authenticated user's profile
func (h *Handler) UpdateCurrentUser(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update fields
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.DietaryRestrictions != nil {
		user.DietaryRestrictions = *req.DietaryRestrictions
	}
	if req.NotificationEnabled != nil {
		user.NotificationEnabled = *req.NotificationEnabled
	}

	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// BlockUser blocks a user (admin only)
func (h *Handler) BlockUser(c *gin.Context) {
	userID := c.Param("id")
	
	var user models.User
	if err := h.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.Blocked = true
	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to block user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User blocked successfully", "user": user})
}

// UnblockUser unblocks a user (admin only)
func (h *Handler) UnblockUser(c *gin.Context) {
	userID := c.Param("id")
	
	var user models.User
	if err := h.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.Blocked = false
	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unblock user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User unblocked successfully", "user": user})
}

// ChangeRoleRequest represents the request body for changing a user's role
type ChangeRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// ChangeUserRole changes a user's role (admin only)
func (h *Handler) ChangeUserRole(c *gin.Context) {
	userID := c.Param("id")

	var req ChangeRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role is required"})
		return
	}

	// Validate role value
	validRoles := map[string]bool{"student": true, "staff": true, "admin": true}
	if !validRoles[req.Role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role. Must be student, staff, or admin"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Prevent admin from changing their own role
	adminID, _ := middleware.GetUserID(c)
	if user.ID == adminID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot change your own role"})
		return
	}

	oldRole := user.Role
	user.Role = models.UserRole(req.Role)
	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change role"})
		return
	}

	// Log role change to audit
	h.DB.Create(&models.AuditLog{
		Action:    "role_change",
		UserID:    &adminID,
		Details:   "Changed role of " + user.Email + " from " + string(oldRole) + " to " + req.Role,
		IPAddress: c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "Role updated successfully", "user": user})
}
