package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/smart-cafeteria/backend/internal/config"
	"github.com/smart-cafeteria/backend/internal/models"
)

// Claims represents JWT claims
type Claims struct {
	UserID uuid.UUID       `json:"userId"`
	Email  string          `json:"email"`
	Role   models.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// AuthRequired middleware validates JWT token
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Check for Bearer prefix
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		secret := config.GetEnv("JWT_SECRET", "your-secret-key")

		// Parse token
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// AdminOnly middleware restricts access to admin users
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		if role.(models.UserRole) != models.RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// StaffOrAdmin middleware allows both staff and admin users
func StaffOrAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		userRole := role.(models.UserRole)
		if userRole != models.RoleAdmin && userRole != models.RoleStaff {
			c.JSON(http.StatusForbidden, gin.H{"error": "Staff or admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return uuid.Nil, false
	}
	return userID.(uuid.UUID), true
}

// GetUserRole extracts user role from context
func GetUserRole(c *gin.Context) (models.UserRole, bool) {
	role, exists := c.Get("role")
	if !exists {
		return "", false
	}
	return role.(models.UserRole), true
}
