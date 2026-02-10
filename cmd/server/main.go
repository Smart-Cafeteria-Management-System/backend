package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/smart-cafeteria/backend/internal/config"
	"github.com/smart-cafeteria/backend/internal/database"
	"github.com/smart-cafeteria/backend/internal/handlers"
	"github.com/smart-cafeteria/backend/internal/middleware"
)

func main() {
	// Load .env file if exists (for local development)
	godotenv.Load()

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
		// Basic Health Check - Verifies the server is operational
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "OK", "message": "Smart Cafeteria API is running"})
		})

		// Authentication Endpoints (Login and Registration)
		auth := api.Group("/auth")
		{
			auth.POST("/login", h.Login)
			auth.POST("/register", h.Register)
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

		// User routes (keep for backward compatibility)
		users := protected.Group("/users")
		{
			users.GET("/me", h.GetCurrentUser)
			users.PUT("/me", h.UpdateCurrentUser)
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
			waste.POST("", middleware.AdminOnly(), h.CreateWasteLog)
			waste.PUT("/:id", middleware.AdminOnly(), h.UpdateWasteLog)
			waste.DELETE("/:id", middleware.AdminOnly(), h.DeleteWasteLog)
		}

		// Sustainability routes
		sustainability := protected.Group("/sustainability")
		{
			sustainability.GET("/report", h.GetSustainabilityReport)
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
	}

	// Get port from environment or default
	port := config.GetEnv("PORT", "5000")
	log.Printf("Server starting on port %s", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
