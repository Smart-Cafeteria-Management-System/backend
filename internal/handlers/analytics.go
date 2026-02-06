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
