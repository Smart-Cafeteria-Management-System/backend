package handlers

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"github.com/smart-cafeteria/backend/internal/config"
	"github.com/smart-cafeteria/backend/internal/middleware"
	"github.com/smart-cafeteria/backend/internal/models"
)

// SetupTOTPResponse holds the response for TOTP setup
type SetupTOTPResponse struct {
	Secret     string `json:"secret"`
	QRCode     string `json:"qrCode"` // base64 PNG data URL
	OTPAuthURL string `json:"otpAuthURL"`
}

// ConfirmTOTPRequest is the body for confirming a TOTP code
type ConfirmTOTPRequest struct {
	Code string `json:"code" binding:"required"`
}

// DisableTOTPRequest requires password + OTP to disable
type DisableTOTPRequest struct {
	Password string `json:"password" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

// FirstSetupTOTPRequest accepts a temp token (issued at login) instead of a full JWT
type FirstSetupTOTPRequest struct {
	TempToken string `json:"tempToken" binding:"required"`
}

// FirstConfirmTOTPRequest accepts temp token + OTP, enabling TOTP and issuing the real JWT
type FirstConfirmTOTPRequest struct {
	TempToken string `json:"tempToken" binding:"required"`
	Code      string `json:"code" binding:"required"`
}

// --- Helpers ---

func generateQRBase64(key interface {
	Image(int, int) (interface{ Bounds() interface{} }, error)
}) (string, error) {
	return "", nil // unused placeholder; real QR generation is inline below
}

func userQRCodeBase64(user models.User) (string, string, string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Smart Cafeteria",
		AccountName: user.Email,
	})
	if err != nil {
		return "", "", "", err
	}

	img, err := key.Image(200, 200)
	if err != nil {
		return "", "", "", err
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", "", "", err
	}

	qrBase64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	return key.Secret(), qrBase64, key.URL(), nil
}

// --- Standard (already-authenticated) TOTP endpoints ---

// SetupTOTP generates a new TOTP secret for an authenticated user (re-setup or voluntary setup).
func (h *Handler) SetupTOTP(c *gin.Context) {
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

	secret, qrBase64, otpURL, err := userQRCodeBase64(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate TOTP key"})
		return
	}

	if err := h.DB.Model(&user).Update("totp_secret", secret).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save TOTP secret"})
		return
	}

	LogActivity(h.DB, c, &userID, user.Email, "TOTP_SETUP_INITIATED", "auth", map[string]interface{}{
		"email": user.Email,
	}, true)

	c.JSON(http.StatusOK, SetupTOTPResponse{
		Secret:     secret,
		QRCode:     qrBase64,
		OTPAuthURL: otpURL,
	})
}

// ConfirmTOTP validates the first OTP code and enables TOTP for an authenticated user.
func (h *Handler) ConfirmTOTP(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ConfirmTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.TOTPSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No TOTP setup in progress. Call /totp/setup first."})
		return
	}

	valid, vErr := totp.ValidateCustom(req.Code, user.TOTPSecret, time.Now().UTC(), totp.ValidateOpts{
		Period: 30, Skew: 1, Digits: 6, Algorithm: 0,
	})
	if vErr != nil || !valid {
		LogActivity(h.DB, c, &userID, user.Email, "TOTP_CONFIRM_FAILED", "auth", nil, false)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid OTP code"})
		return
	}

	if err := h.DB.Model(&user).Update("totp_enabled", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enable TOTP"})
		return
	}

	LogActivity(h.DB, c, &userID, user.Email, "TOTP_ENABLED", "auth", nil, true)
	c.JSON(http.StatusOK, gin.H{"message": "Two-factor authentication has been enabled successfully"})
}

// DisableTOTP requires the user's password and a valid OTP to turn off 2FA.
func (h *Handler) DisableTOTP(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req DisableTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if !user.TOTPEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Two-factor authentication is not enabled for this account"})
		return
	}

	if !user.CheckPassword(req.Password) {
		LogActivity(h.DB, c, &userID, user.Email, "TOTP_DISABLE_FAILED", "auth", map[string]interface{}{"reason": "invalid password"}, false)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	valid2, vErr2 := totp.ValidateCustom(req.Code, user.TOTPSecret, time.Now().UTC(), totp.ValidateOpts{
		Period: 30, Skew: 1, Digits: 6, Algorithm: 0,
	})
	if vErr2 != nil || !valid2 {
		LogActivity(h.DB, c, &userID, user.Email, "TOTP_DISABLE_FAILED", "auth", map[string]interface{}{"reason": "invalid OTP"}, false)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid OTP code"})
		return
	}

	if err := h.DB.Model(&user).Updates(map[string]interface{}{
		"totp_enabled": false,
		"totp_secret":  "",
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disable TOTP"})
		return
	}

	LogActivity(h.DB, c, &userID, user.Email, "TOTP_DISABLED", "auth", nil, true)
	c.JSON(http.StatusOK, gin.H{"message": "Two-factor authentication has been disabled"})
}

// GetTOTPStatus returns whether TOTP is enabled for the current user.
func (h *Handler) GetTOTPStatus(c *gin.Context) {
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

	c.JSON(http.StatusOK, gin.H{"totpEnabled": user.TOTPEnabled})
}

// --- Mandatory first-time setup endpoints (accept temp token, no full JWT required) ---

// parseTempTokenFromBody validates a temp token from the request body and returns the user.
func (h *Handler) parseTempTokenFromBody(tempToken string) (*models.User, error) {
	secret := config.GetEnv("JWT_SECRET", "your-secret-key")
	claims := &TempClaims{}
	token, err := jwt.ParseWithClaims(tempToken, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid || !claims.IsTempToken {
		return nil, err
	}

	var user models.User
	if err := h.DB.First(&user, "id = ?", claims.UserID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FirstSetupTOTP generates a TOTP secret for a user who hasn't set up 2FA yet.
// Accepts a temp token (issued during login) instead of a full JWT.
func (h *Handler) FirstSetupTOTP(c *gin.Context) {
	var req FirstSetupTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.parseTempTokenFromBody(req.TempToken)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired session. Please log in again."})
		return
	}

	secret, qrBase64, otpURL, err := userQRCodeBase64(*user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate TOTP key"})
		return
	}

	if err := h.DB.Model(user).Update("totp_secret", secret).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save TOTP secret"})
		return
	}

	LogActivity(h.DB, c, &user.ID, user.Email, "TOTP_FIRST_SETUP_INITIATED", "auth", nil, true)

	c.JSON(http.StatusOK, SetupTOTPResponse{
		Secret:     secret,
		QRCode:     qrBase64,
		OTPAuthURL: otpURL,
	})
}

// FirstConfirmTOTP validates the OTP for a mandatory first-time setup,
// enables TOTP, and returns the full JWT so the user can proceed.
func (h *Handler) FirstConfirmTOTP(c *gin.Context) {
	var req FirstConfirmTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.parseTempTokenFromBody(req.TempToken)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired session. Please log in again."})
		return
	}

	if user.TOTPSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TOTP setup not initiated. Call /auth/totp/first-setup first."})
		return
	}

	valid3, vErr3 := totp.ValidateCustom(req.Code, user.TOTPSecret, time.Now().UTC(), totp.ValidateOpts{
		Period: 30, Skew: 1, Digits: 6, Algorithm: 0,
	})
	if vErr3 != nil || !valid3 {
		LogActivity(h.DB, c, &user.ID, user.Email, "TOTP_FIRST_CONFIRM_FAILED", "auth", nil, false)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid OTP code"})
		return
	}

	if err := h.DB.Model(user).Update("totp_enabled", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enable TOTP"})
		return
	}

	// Issue the real JWT now that 2FA is set up and verified
	authToken, err := generateToken(*user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Reload user to get updated field
	h.DB.First(user, "id = ?", user.ID)

	LogActivity(h.DB, c, &user.ID, user.Email, "TOTP_ENABLED", "auth", nil, true)
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
		"totpEnabled":         true,
		"createdAt":           user.CreatedAt,
		"updatedAt":           user.UpdatedAt,
	})
}
