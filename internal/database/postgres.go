package database

import (
	"fmt"
	"log"

	"github.com/smart-cafeteria/backend/internal/config"
	"github.com/smart-cafeteria/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect establishes a connection to PostgreSQL
func Connect() (*gorm.DB, error) {
	host := config.GetEnv("DB_HOST", "localhost")
	port := config.GetEnv("DB_PORT", "5432")
	user := config.GetEnv("DB_USER", "cafeteria")
	password := config.GetEnv("DB_PASSWORD", "cafeteria123")
	dbname := config.GetEnv("DB_NAME", "cafeteria")

	dsn := fmt.Sprintf(
		"host='%s' port='%s' user='%s' password='%s' dbname='%s' sslmode=require TimeZone=Asia/Kolkata",
		host, port, user, password, dbname,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Connected to PostgreSQL database")
	return db, nil
}

// AutoMigrate runs database migrations
func AutoMigrate(db *gorm.DB) error {
	log.Println("Running database migrations...")

	return db.AutoMigrate(
		&models.User{},
		&models.MenuItem{},
		&models.MealSlot{},
		&models.Booking{},
		&models.BookingItem{},
		&models.QueueToken{},
		&models.DemandForecast{},
		&models.Analytics{},
		&models.WasteLog{},
		// Incentive system models
		&models.IncentiveRule{},
		&models.UserPoints{},
		&models.PointTransaction{},
		&models.AttendanceLog{},
		// Addon redemption models
		&models.Addon{},
		&models.AddonRedemption{},
		// Audit log
		&models.AuditLog{},
		// Password reset
		&models.PasswordReset{},
		// Operating hours configuration
		&models.OperatingHours{},
		// Global system settings
		&models.SystemSettings{},
	)
}
