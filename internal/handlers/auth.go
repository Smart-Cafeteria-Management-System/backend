package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/smart-cafeteria/backend/internal/config"
	"github.com/smart-cafeteria/backend/internal/middleware"
	"github.com/smart-cafeteria/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
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
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

// VerifyTOTPRequest is the body for the TOTP verification step after login
type VerifyTOTPRequest struct {
	TempToken string `json:"tempToken" binding:"required"`
	Code      string `json:"code" binding:"required"`
}

// Login authenticates a user and returns a JWT token.
// If TOTP is enabled, it returns totp_required: true and a short-lived temp_token instead.
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user by email
	var user models.User
	if err := h.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		LogActivity(h.DB, c, nil, req.Email, "LOGIN_FAILED", "auth",
			map[string]interface{}{"reason": "user not found"}, false)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Check password
	if !user.CheckPassword(req.Password) {
		LogActivity(h.DB, c, &user.ID, user.Email, "LOGIN_FAILED", "auth",
			map[string]interface{}{"reason": "invalid password"}, false)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Check if user is blocked
	if user.Blocked {
		LogActivity(h.DB, c, &user.ID, user.Email, "LOGIN_FAILED", "auth",
			map[string]interface{}{"reason": "account blocked"}, false)
		c.JSON(http.StatusForbidden, gin.H{"error": "Your account has been blocked. Please contact the administrator."})
		return
	}

	// Mandatory 2FA gate: users without TOTP configured must set it up before getting a JWT
	if !user.TOTPEnabled {
		tempToken, err := generateTempToken(user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate temp token"})
			return
		}
		LogActivity(h.DB, c, &user.ID, user.Email, "LOGIN_TOTP_SETUP_REQUIRED", "auth", nil, true)
		c.JSON(http.StatusOK, gin.H{
			"totp_setup_required": true,
			"temp_token":          tempToken,
		})
		return
	}

	// TOTP is configured — require OTP verification before issuing the JWT
	tempToken, err := generateTempToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate temp token"})
		return
	}
	LogActivity(h.DB, c, &user.ID, user.Email, "LOGIN_TOTP_REQUIRED", "auth", nil, true)
	c.JSON(http.StatusOK, gin.H{
		"totp_required": true,
		"temp_token":    tempToken,
	})
}

// VerifyTOTP handles the second factor — validates the temp token + OTP and issues the real JWT.
func (h *Handler) VerifyTOTP(c *gin.Context) {
	var req VerifyTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse temp token
	secret := config.GetEnv("JWT_SECRET", "your-secret-key")
	claims := &TempClaims{}
	token, err := jwt.ParseWithClaims(req.TempToken, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid || !claims.IsTempToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired temp token"})
		return
	}

	// Fetch user
	var user models.User
	if err := h.DB.First(&user, "id = ?", claims.UserID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Validate OTP — debug logging
	now := time.Now().UTC()
	log.Printf("[TOTP DEBUG] ServerTime=%s, UserEmail=%s, SecretPrefix=%s..., CodeSubmitted=%s",
		now.Format(time.RFC3339), user.Email, user.TOTPSecret[:4], req.Code)

	// Generate what the expected code should be for comparison
	expectedCode, _ := totp.GenerateCodeCustom(user.TOTPSecret, now, totp.ValidateOpts{
		Period:    30,
		Skew:     1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	log.Printf("[TOTP DEBUG] ExpectedCode=%s, SecretLen=%d", expectedCode, len(user.TOTPSecret))

	valid, err := totp.ValidateCustom(req.Code, user.TOTPSecret, now, totp.ValidateOpts{
		Period:    30,
		Skew:     1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	log.Printf("[TOTP DEBUG] Valid=%v, Err=%v", valid, err)

	if err != nil || !valid {
		LogActivity(h.DB, c, &user.ID, user.Email, "LOGIN_TOTP_FAILED", "auth",
			map[string]interface{}{"reason": fmt.Sprintf("invalid OTP: submitted=%s expected=%s serverTime=%s", req.Code, expectedCode, now.Format(time.RFC3339))}, false)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid OTP code"})
		return
	}

	// Issue the real JWT
	authToken, err := generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	LogActivity(h.DB, c, &user.ID, user.Email, "LOGIN_SUCCESS", "auth", nil, true)

	c.JSON(http.StatusOK, gin.H{
		"token":               authToken,
		"_id":                 user.ID,
		"name":                user.Name,
		"email":               user.Email,
		"role":                user.Role,
		"studentId":           user.StudentID,
		"dietaryRestrictions": user.DietaryRestrictions,
		"notificationEnabled": user.NotificationEnabled,
		"totpEnabled":         user.TOTPEnabled,
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

	LogActivity(h.DB, c, &user.ID, user.Email, "REGISTER", "auth", map[string]interface{}{
		"name": user.Name,
		"role": user.Role,
	}, true)

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
		"totpEnabled":         user.TOTPEnabled,
		"createdAt":           user.CreatedAt,
		"updatedAt":           user.UpdatedAt,
	})
}

// TempClaims is a short-lived JWT used as a "password verified" token before TOTP
type TempClaims struct {
	UserID      uuid.UUID `json:"userId"`
	IsTempToken bool      `json:"isTempToken"`
	jwt.RegisteredClaims
}

// generateTempToken creates a 5-minute JWT that only signals password was verified
func generateTempToken(user models.User) (string, error) {
	secret := config.GetEnv("JWT_SECRET", "your-secret-key")
	claims := TempClaims{
		UserID:      user.ID,
		IsTempToken: true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
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

// ForgotPasswordRequest represents the forgot password request
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPassword generates a password reset token
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Valid email is required"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, "email = ?", req.Email).Error; err != nil {
		// Return success even if email not found to prevent email enumeration
		c.JSON(http.StatusOK, gin.H{
			"message": "If an account with that email exists, a password reset link has been generated.",
		})
		return
	}

	// Invalidate any existing reset tokens for this user
	h.DB.Model(&models.PasswordReset{}).Where("user_id = ? AND used = false", user.ID).Update("used", true)

	// Generate a secure random reset token
	resetToken := uuid.New().String()

	// Create password reset entry (expires in 15 minutes)
	passwordReset := models.PasswordReset{
		UserID:    user.ID,
		Token:     resetToken,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	if err := h.DB.Create(&passwordReset).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reset token"})
		return
	}

	// Log the password reset request
	h.DB.Create(&models.AuditLog{
		Action:    "password_reset_requested",
		UserID:    &user.ID,
		UserEmail: user.Email,
		Details:   "Password reset token generated",
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	// In a production system, this token would be sent via email.
	// For this project, we return it directly so it can be used in the UI.
	log.Printf("[PASSWORD RESET] Token for %s: %s (expires in 15 min)", user.Email, resetToken)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Password reset token generated successfully.",
		"resetToken": resetToken,
		"expiresIn":  "15 minutes",
	})
}

// ResetPasswordRequest represents the reset password request
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

// ResetPassword resets the user's password using a valid reset token
func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token and new password (min 6 chars) are required"})
		return
	}

	// Find the reset token
	var resetEntry models.PasswordReset
	if err := h.DB.First(&resetEntry, "token = ? AND used = false", req.Token).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset token"})
		return
	}

	// Check if token is expired
	if resetEntry.IsExpired() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reset token has expired. Please request a new one."})
		return
	}

	// Find the user
	var user models.User
	if err := h.DB.First(&user, "id = ?", resetEntry.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update password — explicitly hash it here for robustness, 
	// then Save() will update the DB. This avoids potential hook bypass issues.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	
	user.Password = string(hashedPassword)
	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	// Mark token as used
	resetEntry.Used = true
	h.DB.Save(&resetEntry)

	// Log the password reset
	h.DB.Create(&models.AuditLog{
		Action:    "password_reset_completed",
		UserID:    &user.ID,
		UserEmail: user.Email,
		Details:   "Password was reset via recovery token",
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Password has been reset successfully. You can now log in with your new password."})
}
