package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smart-cafeteria/backend/internal/models"
	"gorm.io/gorm"
)

// LogActivity writes an audit log entry to the database.
// It is safe to call with a nil userID (for anonymous events).
func LogActivity(db *gorm.DB, c *gin.Context, userID *uuid.UUID, userEmail, action, resource string, details map[string]interface{}, success bool) {
	detailsJSON := "{}"
	if details != nil {
		if b, err := json.Marshal(details); err == nil {
			detailsJSON = string(b)
		}
	}

	entry := models.AuditLog{
		UserID:    userID,
		UserEmail: userEmail,
		Action:    action,
		Resource:  resource,
		Details:   detailsJSON,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Success:   success,
	}
	// Intentionally ignore error — audit log failure should not break the request
	db.Create(&entry)
}

// GetAuditLogs returns paginated audit log entries (admin only)
func (h *Handler) GetAuditLogs(c *gin.Context) {
	var logs []models.AuditLog

	query := h.DB.Order("created_at DESC")

	// Optional filters
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}
	if email := c.Query("email"); email != "" {
		query = query.Where("user_email ILIKE ?", "%"+email+"%")
	}
	if from := c.Query("from"); from != "" {
		query = query.Where("created_at >= ?", from)
	}
	if to := c.Query("to"); to != "" {
		query = query.Where("created_at <= ?", to)
	}
	if success := c.Query("success"); success != "" {
		query = query.Where("success = ?", success == "true")
	}

	// Count
	var total int64
	query.Model(&models.AuditLog{}).Count(&total)

	// Pagination
	page := 1
	pageSize := 50
	offset := (page - 1) * pageSize

	if err := query.Limit(pageSize).Offset(offset).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":     logs,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}
