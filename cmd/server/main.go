package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/smart-cafeteria/backend/internal/config"
	"github.com/smart-cafeteria/backend/internal/database"
	"github.com/smart-cafeteria/backend/internal/handlers"
	"github.com/smart-cafeteria/backend/internal/middleware"
)

var startTime time.Time

func init() {
	startTime = time.Now()
}

func main() {
	// Load .env file if exists (for local development)
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or error loading: %v", err)
	}

	// Initialize database connection to Supabase (PostgreSQL)
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Critical Error: Failed to connect to database: %v", err)
	}

	// Run AutoMigrations to ensure database tables match GORM models
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("Critical Error: Failed to run migrations: %v", err)
	}

	// Granular Database Seeding: Populates initial sample data if tables are empty
	// Controlled by the SEED_DB environment variable
	if os.Getenv("SEED_DB") == "true" {
		if err := database.Seed(db); err != nil {
			log.Printf("Warning: Seeding failed: %v", err)
		} else {
			log.Println("Database seeded successfully")
		}
	}

	// Initialize Gin router
	router := gin.Default()

	// Apply CORS middleware
	router.Use(middleware.CORS())

	// Initialize handlers
	h := handlers.New(db)

	// Public API Group (No authentication required)
	api := router.Group("/api")
	{
		// Enhanced Health Check - Verifies the server is operational and monitoring system health
		api.GET("/health", func(c *gin.Context) {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			uptime := time.Since(startTime).String()

			// Check DB connection
			dbStatus := "OK"
			sqlDB, err := h.DB.DB()
			if err != nil || sqlDB.Ping() != nil {
				dbStatus = "Error"
			}

			c.JSON(200, gin.H{
				"status":     "OK",
				"message":    "Smart Cafeteria API is running",
				"uptime":     uptime,
				"dbStatus":   dbStatus,
				"memory":     fmt.Sprintf("%.2f MB", float64(m.Alloc)/1024/1024),
				"goroutines": runtime.NumGoroutine(),
			})
		})

		// Authentication Endpoints (Login and Registration)
		auth := api.Group("/auth")
		{
			auth.POST("/login", h.Login)
			auth.POST("/register", h.Register)
			auth.POST("/verify-totp", h.VerifyTOTP)              // Step 2: user with 2FA already set up
			auth.POST("/totp/first-setup", h.FirstSetupTOTP)     // Mandatory: generate QR for new user
			auth.POST("/totp/first-confirm", h.FirstConfirmTOTP) // Mandatory: confirm + issue real JWT
			auth.POST("/forgot-password", h.ForgotPassword)       // Password recovery
			auth.POST("/reset-password", h.ResetPassword)         // Reset with token
		}

		// Public Menu access allows students to view items before logging in
		api.GET("/menu", h.GetMenuItems)
	}

	// Protected API Group (JWT Authentication Required)
	protected := api.Group("")
	protected.Use(middleware.AuthRequired())
	{
		// Auth profile routes (frontend expects /auth/me)
		protected.GET("/auth/me", h.GetCurrentUser)
		protected.PUT("/auth/me", h.UpdateCurrentUser)

		// TOTP / 2FA routes
		totpGroup := protected.Group("/auth/totp")
		{
			totpGroup.GET("/status", h.GetTOTPStatus)
			totpGroup.POST("/setup", h.SetupTOTP)
			totpGroup.POST("/confirm", h.ConfirmTOTP)
			totpGroup.DELETE("/disable", h.DisableTOTP)
		}

		// User routes (keep for backward compatibility)
		users := protected.Group("/users")
		{
			users.GET("", middleware.AdminOnly(), h.GetAllUsers)
			users.GET("/me", h.GetCurrentUser)
			users.PUT("/me", h.UpdateCurrentUser)
			// Admin user management
			users.PUT("/:id/block", middleware.AdminOnly(), h.BlockUser)
			users.PUT("/:id/unblock", middleware.AdminOnly(), h.UnblockUser)
			users.PUT("/:id/role", middleware.AdminOnly(), h.ChangeUserRole)
		}

		// Menu management (admin only)
		menu := protected.Group("/menu")
		{
			menu.POST("", middleware.AdminOnly(), h.CreateMenuItem)
			menu.PUT("/:id", middleware.AdminOnly(), h.UpdateMenuItem)
			menu.PATCH("/:id/availability", h.UpdateMenuAvailability) // Staff can toggle availability
			menu.DELETE("/:id", middleware.AdminOnly(), h.DeleteMenuItem)
		}

		// Slots routes
		slots := protected.Group("/slots")
		{
			slots.GET("/today", h.GetTodaySlots)
			slots.GET("", h.GetSlots)
			slots.POST("", middleware.AdminOnly(), h.CreateSlot)
			slots.POST("/generate", middleware.AdminOnly(), h.GenerateSlots)
			slots.PUT("/:id", middleware.AdminOnly(), h.UpdateSlot)
			slots.DELETE("/:id", middleware.AdminOnly(), h.DeleteSlot)
		}

		// Bookings routes
		bookings := protected.Group("/bookings")
		{
			bookings.GET("", h.GetMyBookings) // Default to user's bookings
			bookings.GET("/all", middleware.AdminOnly(), h.GetBookings)
			bookings.GET("/my", h.GetMyBookings)
			bookings.POST("", h.CreateBooking)
			bookings.PUT("/:id", h.UpdateBooking)
			bookings.DELETE("/:id", h.CancelBooking)
		}

		// Queue routes
		queue := protected.Group("/queue")
		{
			queue.GET("/status", h.GetQueueStatus)
			queue.GET("/my-token", h.GetMyToken)
			queue.GET("/history", h.GetQueueHistory)
			queue.GET("/fairness", h.GetFairnessIndicators)
			queue.POST("/call-next", middleware.StaffOrAdmin(), h.CallNextToken)
			queue.PUT("/:id/serve", middleware.StaffOrAdmin(), h.ServeToken)
		}

		// Forecasts routes
		forecasts := protected.Group("/forecasts")
		{
			forecasts.GET("", h.GetForecasts)
			forecasts.GET("/today", h.GetTodayForecasts)
			forecasts.GET("/week", h.GetWeekForecasts)
			forecasts.GET("/accuracy", h.GetForecastAccuracy)
			forecasts.POST("/predict", h.GetPrediction)
			forecasts.PUT("/:id/actual", middleware.AdminOnly(), h.UpdateActualDemand)
			forecasts.POST("/record-actual", middleware.AdminOnly(), h.RecordActualFromBookings)
		}

		// Waste tracking routes
		waste := protected.Group("/waste")
		{
			waste.GET("", h.GetWasteLogs)
			waste.GET("/summary", h.GetWasteSummary)
			waste.POST("", middleware.StaffOrAdmin(), h.CreateWasteLog)
			waste.PUT("/:id", middleware.StaffOrAdmin(), h.UpdateWasteLog)
			waste.DELETE("/:id", middleware.StaffOrAdmin(), h.DeleteWasteLog)
		}

		// Sustainability routes
		sustainability := protected.Group("/sustainability")
		{
			sustainability.GET("/report", h.GetSustainabilityReport)
			sustainability.GET("/report/csv", middleware.AdminOnly(), h.DownloadSustainabilityCSV)
			sustainability.GET("/metrics", h.GetSustainabilityMetrics)
		}

		// Preparation recommendations routes
		preparation := protected.Group("/preparation")
		{
			preparation.GET("/recommendations", h.GetPreparationRecommendations)
		}

		// Analytics routes
		analytics := protected.Group("/analytics")
		{
			analytics.GET("/dashboard", h.GetDashboard)
			analytics.GET("/trends", h.GetTrends)
			analytics.GET("/demand-trends", h.GetTrends)           // Alias for frontend
			analytics.GET("/summary", h.GetAnalyticsSummary)       // Analytics summary
			analytics.GET("/waste-report", h.GetWasteReport)       // Waste report for frontend
		}

		// Incentive routes
		incentives := protected.Group("/incentives")
		{
			// User routes
			incentives.GET("/my-points", h.GetMyPoints)
			incentives.GET("/my-history", h.GetPointsHistory)
			incentives.GET("/status", h.GetIncentiveStatus)

			// Admin routes
			incentives.GET("/rules", middleware.AdminOnly(), h.GetIncentiveRules)
			incentives.POST("/rules", middleware.AdminOnly(), h.CreateIncentiveRule)
			incentives.PUT("/rules/:id", middleware.AdminOnly(), h.UpdateIncentiveRule)
			incentives.DELETE("/rules/:id", middleware.AdminOnly(), h.DeleteIncentiveRule)
			incentives.GET("/abuse-report", middleware.AdminOnly(), h.GetAbuseReport)
			incentives.GET("/behavior-trends", middleware.AdminOnly(), h.GetBehaviorTrends)
			incentives.POST("/apply-to-slots", middleware.AdminOnly(), h.ApplyIncentivesToSlots)
		}

		// Addon routes
		addons := protected.Group("/addons")
		{
			// User routes
			addons.GET("", h.GetAddons)
			addons.POST("/:id/redeem", h.RedeemAddon)
			addons.GET("/my-redemptions", h.GetMyRedemptions)

			// Admin routes
			addons.POST("", middleware.AdminOnly(), h.CreateAddon)
			addons.PUT("/:id", middleware.AdminOnly(), h.UpdateAddon)
			addons.DELETE("/:id", middleware.AdminOnly(), h.DeleteAddon)
			addons.POST("/claim", middleware.AdminOnly(), h.ClaimRedemption) // Staff verifies code
		}

		// Audit log routes (admin only)
		protected.GET("/audit-logs", middleware.AdminOnly(), h.GetAuditLogs)

		// System Settings routes
		protected.GET("/system/settings", h.GetSystemSettings)
		protected.PUT("/system/settings", middleware.AdminOnly(), h.UpdateSystemSettings)

		// Operating Hours routes (US-AM-4)
		protected.GET("/operating-hours", h.GetOperatingHours)
		protected.PUT("/operating-hours/:id", middleware.AdminOnly(), h.UpdateOperatingHours)
	}

	// Get port from environment or default
	port := config.GetEnv("PORT", "5000")
	log.Printf("Server starting on port %s", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
