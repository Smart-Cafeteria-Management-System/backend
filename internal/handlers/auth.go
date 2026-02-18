package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/smart-cafeteria/backend/internal/config"
	"github.com/smart-cafeteria/backend/internal/middleware"
	"github.com/smart-cafeteria/backend/internal/models"
)

// LoginRequest represents login payload
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest represents registration payload
type RegisterRequest struct {
	Name      string `json:"name" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=6"`
	StudentID string `json:"studentId"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Token string       `json:"token"`
	User  models.User  `json:"user"`
}

// Login authenticates a user and returns a JWT token
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user by email
	var user models.User
	if err := h.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Check password
	if !user.CheckPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Check if user is blocked
	if user.Blocked {
		c.JSON(http.StatusForbidden, gin.H{"error": "Your account has been blocked. Please contact the administrator."})
		return
	}

	// Generate JWT token
	token, err := generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Return flattened response for frontend compatibility
	c.JSON(http.StatusOK, gin.H{
		"token":               token,
		"_id":                 user.ID,
		"name":                user.Name,
		"email":               user.Email,
		"role":                user.Role,
		"studentId":           user.StudentID,
		"dietaryRestrictions": user.DietaryRestrictions,
		"notificationEnabled": user.NotificationEnabled,
		"createdAt":           user.CreatedAt,
		"updatedAt":           user.UpdatedAt,
	})
}

// Register creates a new user account
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if email already exists
	var existingUser models.User
	if err := h.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	// Create new user
	user := models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Role:     models.RoleStudent,
	}

	if req.StudentID != "" {
		user.StudentID = &req.StudentID
	}

	if err := h.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate JWT token
	token, err := generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Return flattened response for frontend compatibility
	c.JSON(http.StatusCreated, gin.H{
		"token":               token,
		"_id":                 user.ID,
		"name":                user.Name,
		"email":               user.Email,
		"role":                user.Role,
		"studentId":           user.StudentID,
		"dietaryRestrictions": user.DietaryRestrictions,
		"notificationEnabled": user.NotificationEnabled,
		"createdAt":           user.CreatedAt,
		"updatedAt":           user.UpdatedAt,
	})
}

// generateToken creates a JWT token for the user
func generateToken(user models.User) (string, error) {
	secret := config.GetEnv("JWT_SECRET", "your-secret-key")
	
	claims := middleware.Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
