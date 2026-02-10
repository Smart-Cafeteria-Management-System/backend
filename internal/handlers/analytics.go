package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/smart-cafeteria/backend/internal/models"
)

// DashboardResponse represents dashboard data
type DashboardResponse struct {
	Today          TodayStats         `json:"today"`
	QueueStatus    QueueInfo          `json:"queueStatus"`
	DemandByMeal   DemandByMeal       `json:"demandByMeal"`
	RecentBookings []models.Booking   `json:"recentBookings"`
}

// TodayStats represents today's statistics
type TodayStats struct {
	TotalBookings int     `json:"totalBookings"`
	TotalServed   int     `json:"totalServed"`
	AvgWaitTime   float64 `json:"avgWaitTime"`
	Revenue       float64 `json:"revenue"`
}

// QueueInfo represents queue information
type QueueInfo struct {
	CurrentlyWaiting int    `json:"currentlyWaiting"`
	CurrentToken     string `json:"currentToken"`
	AvgWaitTime      int    `json:"avgWaitTime"`
}

// DemandByMeal represents demand by meal type
type DemandByMeal struct {
	Breakfast int `json:"breakfast"`
	Lunch     int `json:"lunch"`
	Dinner    int `json:"dinner"`
}

// GetDashboard returns analytics dashboard data
func (h *Handler) GetDashboard(c *gin.Context) {
	today := time.Now().Truncate(24 * time.Hour)

	// Get today's bookings count
	var totalBookings int64
	h.DB.Model(&models.Booking{}).
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date = ?", today).
		Count(&totalBookings)

	// Get served count
	var totalServed int64
	h.DB.Model(&models.Booking{}).
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date = ?", today).
		Where("bookings.status = ?", "served").
		Count(&totalServed)

	// Get waiting count
	var waitingCount int64
	h.DB.Model(&models.QueueToken{}).
		Joins("JOIN bookings ON bookings.id = queue_tokens.booking_id").
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date = ?", today).
		Where("queue_tokens.status = ?", "waiting").
		Count(&waitingCount)

	// Get current token
	var currentToken models.QueueToken
	h.DB.Joins("JOIN bookings ON bookings.id = queue_tokens.booking_id").
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date = ?", today).
		Where("queue_tokens.status = ?", "called").
		First(&currentToken)

	// Get demand by meal type for today
	var breakfastCount, lunchCount, dinnerCount int64
	h.DB.Model(&models.Booking{}).
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date = ?", today).
		Where("meal_slots.meal_type = ?", "breakfast").
		Count(&breakfastCount)
	h.DB.Model(&models.Booking{}).
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date = ?", today).
		Where("meal_slots.meal_type = ?", "lunch").
		Count(&lunchCount)
	h.DB.Model(&models.Booking{}).
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date = ?", today).
		Where("meal_slots.meal_type = ?", "dinner").
		Count(&dinnerCount)

	// Get recent bookings
	var recentBookings []models.Booking
	h.DB.Preload("User").Preload("Slot").Preload("Items").
		Order("created_at DESC").
		Limit(10).
		Find(&recentBookings)

	response := DashboardResponse{
		Today: TodayStats{
			TotalBookings: int(totalBookings),
			TotalServed:   int(totalServed),
			AvgWaitTime:   float64(waitingCount) * 2,
			Revenue:       float64(totalServed) * 75, // Average meal price
		},
		QueueStatus: QueueInfo{
			CurrentlyWaiting: int(waitingCount),
			CurrentToken:     currentToken.TokenNumber,
			AvgWaitTime:      int(waitingCount) * 2,
		},
		DemandByMeal: DemandByMeal{
			Breakfast: int(breakfastCount),
			Lunch:     int(lunchCount),
			Dinner:    int(dinnerCount),
		},
		RecentBookings: recentBookings,
	}

	c.JSON(http.StatusOK, response)
}

// TrendsResponse represents trend data
type TrendsResponse struct {
	DailyBookings []DailyBooking `json:"dailyBookings"`
	WeeklyTrend   []WeeklyTrend  `json:"weeklyTrend"`
}

// DailyBooking represents daily booking data
type DailyBooking struct {
	Date     string `json:"date"`
	Bookings int    `json:"bookings"`
	Served   int    `json:"served"`
}

// WeeklyTrend represents weekly trend data
type WeeklyTrend struct {
	Week      string  `json:"week"`
	AvgDemand float64 `json:"avgDemand"`
}

// GetTrends returns analytics trends
func (h *Handler) GetTrends(c *gin.Context) {
	// Get last 7 days of booking data
	var dailyBookings []DailyBooking
	
	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i).Truncate(24 * time.Hour)
		
		var bookings, served int64
		h.DB.Model(&models.Booking{}).
			Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
			Where("meal_slots.date = ?", date).
			Count(&bookings)
		
		h.DB.Model(&models.Booking{}).
			Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
			Where("meal_slots.date = ?", date).
			Where("bookings.status = ?", "served").
			Count(&served)

		dailyBookings = append(dailyBookings, DailyBooking{
			Date:     date.Format("2006-01-02"),
			Bookings: int(bookings),
			Served:   int(served),
		})
	}

	response := TrendsResponse{
		DailyBookings: dailyBookings,
		WeeklyTrend:   []WeeklyTrend{}, // Can be implemented later
	}

	c.JSON(http.StatusOK, response)
}

// DailyWaste represents daily waste data
type DailyWaste struct {
	Date            string  `json:"date"`
	WastePercentage float64 `json:"wastePercentage"`
}

// WasteReportResponse represents waste report data
type WasteReportResponse struct {
	AvgWastePercentage float64      `json:"avgWastePercentage"`
	Trend              string       `json:"trend"` // "good", "moderate", "bad"
	WasteData          []DailyWaste `json:"wasteData"`
}

// GetWasteReport returns waste report for the last 7 days
func (h *Handler) GetWasteReport(c *gin.Context) {
	// Get last 7 days of waste data
	var wasteData []DailyWaste
	var totalWastePercentage float64
	var validDays int

	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i).Truncate(24 * time.Hour)

		var logs []models.WasteLog
		h.DB.Where("date = ?", date).Find(&logs)

		var totalPrepared, totalWasted int
		for _, log := range logs {
			totalPrepared += log.PreparedQuantity
			totalWasted += log.WastedQuantity
		}

		wastePercentage := 0.0
		if totalPrepared > 0 {
			wastePercentage = float64(totalWasted) / float64(totalPrepared) * 100
			totalWastePercentage += wastePercentage
			validDays++
		}

		wasteData = append(wasteData, DailyWaste{
			Date:            date.Format("2006-01-02"),
			WastePercentage: wastePercentage,
		})
	}

	// Calculate average waste percentage
	avgWastePercentage := 0.0
	if validDays > 0 {
		avgWastePercentage = totalWastePercentage / float64(validDays)
	}

	// Determine trend
	trend := "good"
	if avgWastePercentage >= 20 {
		trend = "bad"
	} else if avgWastePercentage >= 10 {
		trend = "moderate"
	}

	response := WasteReportResponse{
		AvgWastePercentage: avgWastePercentage,
		Trend:              trend,
		WasteData:          wasteData,
	}

	c.JSON(http.StatusOK, response)
}

// AnalyticsSummaryResponse represents the analytics summary data
type AnalyticsSummaryResponse struct {
	TotalUsers      int     `json:"totalUsers"`
	TotalBookings   int     `json:"totalBookings"`
	TodayBookings   int     `json:"todayBookings"`
	PeakHour        *string `json:"peakHour"`
	AvgDailyBookings int    `json:"avgDailyBookings"`
}

// GetAnalyticsSummary returns summary statistics for analytics page
func (h *Handler) GetAnalyticsSummary(c *gin.Context) {
	today := time.Now().Truncate(24 * time.Hour)

	// Get total users count
	var totalUsers int64
	h.DB.Model(&models.User{}).Count(&totalUsers)

	// Get total bookings count (all time)
	var totalBookings int64
	h.DB.Model(&models.Booking{}).Count(&totalBookings)

	// Get today's bookings count
	var todayBookings int64
	h.DB.Model(&models.Booking{}).
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date = ?", today).
		Count(&todayBookings)

	// Calculate peak hour (most bookings by hour)
	var peakHour *string
	var peakResult struct {
		Hour  int
		Count int64
	}
	h.DB.Raw(`
		SELECT EXTRACT(HOUR FROM meal_slots.start_time) as hour, COUNT(*) as count
		FROM bookings
		JOIN meal_slots ON meal_slots.id = bookings.slot_id
		WHERE meal_slots.date = ?
		GROUP BY hour
		ORDER BY count DESC
		LIMIT 1
	`, today).Scan(&peakResult)
	
	if peakResult.Count > 0 {
		hourStr := time.Date(0, 1, 1, peakResult.Hour, 0, 0, 0, time.UTC).Format("3PM")
		peakHour = &hourStr
	}

	// Calculate average daily bookings (last 30 days)
	var avgDailyBookings float64
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Truncate(24 * time.Hour)
	var recentBookings int64
	h.DB.Model(&models.Booking{}).
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date >= ?", thirtyDaysAgo).
		Count(&recentBookings)
	avgDailyBookings = float64(recentBookings) / 30.0

	response := AnalyticsSummaryResponse{
		TotalUsers:       int(totalUsers),
		TotalBookings:    int(totalBookings),
		TodayBookings:    int(todayBookings),
		PeakHour:         peakHour,
		AvgDailyBookings: int(avgDailyBookings),
	}

	c.JSON(http.StatusOK, response)
}

