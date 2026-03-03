package handlers

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smart-cafeteria/backend/internal/middleware"
	"github.com/smart-cafeteria/backend/internal/models"
)

// QueueStatusResponse represents queue status
type QueueStatusResponse struct {
	CurrentlyServing *models.QueueToken  `json:"currentlyServing"`
	WaitingCount     int64               `json:"waitingCount"`
	AvgWaitTime      int                 `json:"avgWaitTime"`
	WaitingTokens    []models.QueueToken `json:"waitingTokens"`
	RecentlyCalled   []models.QueueToken `json:"recentlyCalled"`
}

// getTodayRange returns start and end times for today in local timezone
func getTodayRange() (time.Time, time.Time) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 0, 1)
	return start, end
}

// GetQueueStatus returns the current queue status
func (h *Handler) GetQueueStatus(c *gin.Context) {
	todayStart, todayEnd := getTodayRange()

	// Fetch waiting tokens
	var waitingTokens []models.QueueToken
	if err := h.DB.Preload("User").Preload("Booking.Items.MenuItem").
		Joins("JOIN bookings ON bookings.id = queue_tokens.booking_id").
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date >= ? AND meal_slots.date < ?", todayStart, todayEnd).
		Where("queue_tokens.status = ?", models.TokenWaiting).
		Order("queue_tokens.created_at").
		Find(&waitingTokens).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch queue status"})
		return
	}

	// Fetch recently called tokens (called but not yet served)
	var recentlyCalled []models.QueueToken
	if err := h.DB.Preload("User").Preload("Booking.Items.MenuItem").
		Joins("JOIN bookings ON bookings.id = queue_tokens.booking_id").
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date >= ? AND meal_slots.date < ?", todayStart, todayEnd).
		Where("queue_tokens.status = ?", models.TokenCalled).
		Order("queue_tokens.called_at DESC").
		Find(&recentlyCalled).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch called tokens"})
		return
	}

	waitingCount := int64(len(waitingTokens))

	var currentlyServing *models.QueueToken
	if len(recentlyCalled) > 0 {
		currentlyServing = &recentlyCalled[0]
	}

	// Calculate accurate average wait time for the queue
	totalPrepTime := 0
	for _, t := range waitingTokens {
		for _, item := range t.Booking.Items {
			totalPrepTime += item.MenuItem.PreparationTime * item.Quantity
		}
		totalPrepTime += 1 // Handover buffer
	}

	avgWait := 0
	if waitingCount > 0 {
		avgWait = totalPrepTime
	}

	response := QueueStatusResponse{
		CurrentlyServing: currentlyServing,
		WaitingCount:     waitingCount,
		AvgWaitTime:      avgWait,
		WaitingTokens:    waitingTokens,
		RecentlyCalled:   recentlyCalled,
	}

	c.JSON(http.StatusOK, response)
}

// GetMyToken returns the authenticated student's current queue information.
// CRITICAL ETHICS FEATURE: Calculates 'Dynamic Wait Time' by summing the 
// preparation times of all items ordered by students ahead in the queue.
func (h *Handler) GetMyToken(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	todayStart, todayEnd := getTodayRange()

	var token models.QueueToken
	err := h.DB.Preload("Booking").Preload("Booking.Slot").
		Joins("JOIN bookings ON bookings.id = queue_tokens.booking_id").
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("queue_tokens.user_id = ?", userID).
		Where("meal_slots.date >= ? AND meal_slots.date < ?", todayStart, todayEnd).
		Where("queue_tokens.status IN ?", []string{"waiting", "called"}).
		First(&token).Error

	if err != nil {
		// No active token found
		c.JSON(http.StatusOK, gin.H{"token": nil, "position": 0, "estimatedWait": 0})
		return
	}

	// Calculate position in queue
	var position int64
	h.DB.Model(&models.QueueToken{}).
		Joins("JOIN bookings ON bookings.id = queue_tokens.booking_id").
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date >= ? AND meal_slots.date < ?", todayStart, todayEnd).
		Where("queue_tokens.status = ?", "waiting").
		Where("queue_tokens.created_at < ?", token.CreatedAt).
		Count(&position)

	// Calculate dynamic wait time based on preparation times of people ahead
	var aheadTokens []models.QueueToken
	h.DB.Preload("Booking.Items.MenuItem").
		Joins("JOIN bookings ON bookings.id = queue_tokens.booking_id").
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date >= ? AND meal_slots.date < ?", todayStart, todayEnd).
		Where("queue_tokens.status = ?", models.TokenWaiting).
		Where("queue_tokens.created_at < ?", token.CreatedAt).
		Find(&aheadTokens)

	dynamicWait := 0
	for _, t := range aheadTokens {
		tokenPrepTime := 0
		for _, item := range t.Booking.Items {
			tokenPrepTime += item.MenuItem.PreparationTime * item.Quantity
		}
		dynamicWait += tokenPrepTime + 1 // Add 1 min handover buffer per token
	}

	// Add current user's items prep time
	var currentBooking models.Booking
	h.DB.Preload("Items.MenuItem").First(&currentBooking, "id = ?", token.BookingID)
	for _, item := range currentBooking.Items {
		dynamicWait += item.MenuItem.PreparationTime * item.Quantity
	}

	if dynamicWait < 2 {
		dynamicWait = 2 // Minimum 2 minutes
	}

	c.JSON(http.StatusOK, gin.H{
		"token":         token,
		"position":      len(aheadTokens) + 1,
		"estimatedWait": dynamicWait,
	})
}

// GetQueueHistory returns the queue history for today
func (h *Handler) GetQueueHistory(c *gin.Context) {
	todayStart, todayEnd := getTodayRange()

	var tokens []models.QueueToken
	if err := h.DB.Preload("User").Preload("Booking").
		Joins("JOIN bookings ON bookings.id = queue_tokens.booking_id").
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date >= ? AND meal_slots.date < ?", todayStart, todayEnd).
		Order("queue_tokens.created_at DESC").
		Limit(50).
		Find(&tokens).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch queue history"})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// CallNextToken marks the next token in queue as 'called' (Staff Only).
// ENFORCED FIFO: The system automatically picks the oldest waiting token.
func (h *Handler) CallNextToken(c *gin.Context) {
	todayStart, todayEnd := getTodayRange()

	// Get the next waiting token
	var token models.QueueToken
	if err := h.DB.Preload("User").Preload("Booking.Items").
		Joins("JOIN bookings ON bookings.id = queue_tokens.booking_id").
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date >= ? AND meal_slots.date < ?", todayStart, todayEnd).
		Where("queue_tokens.status = ?", "waiting").
		Order("queue_tokens.created_at").
		First(&token).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No waiting tokens in queue"})
		return
	}

	// Update token status
	now := time.Now()
	token.Status = models.TokenCalled
	token.CalledAt = &now

	// Get counter number from request if provided
	var req struct {
		CounterNumber int `json:"counterNumber"`
	}
	if err := c.ShouldBindJSON(&req); err == nil && req.CounterNumber > 0 {
		token.CounterNumber = &req.CounterNumber
	}

	if err := h.DB.Save(&token).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update token"})
		return
	}

	c.JSON(http.StatusOK, token)
}

// ServeToken marks a token as served and awards attendance points
func (h *Handler) ServeToken(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID"})
		return
	}

	var token models.QueueToken
	if err := h.DB.Preload("Booking").Preload("Booking.Slot").First(&token, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	if token.Status != models.TokenCalled && token.Status != models.TokenWaiting {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token cannot be served"})
		return
	}

	tx := h.DB.Begin()

	// Update token
	token.Status = models.TokenServed
	if err := tx.Save(&token).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update token"})
		return
	}

	// Update booking
	now := time.Now()
	tx.Model(&models.Booking{}).Where("id = ?", token.BookingID).Updates(map[string]interface{}{
		"status":    models.BookingServed,
		"served_at": now,
	})

	tx.Commit()

	// Award points (after commit) - check if booking was preloaded using ID check
	if token.Booking.ID != uuid.Nil && token.Booking.Slot.ID != uuid.Nil {
		h.AwardAttendancePoints(
			token.UserID,
			token.BookingID,
			token.Booking.SlotID,
			token.Booking.Slot.HasIncentive,
			token.Booking.Slot.IncentivePoints,
		)
	}

	c.JSON(http.StatusOK, token)
}

// === Fairness Indicators (US-ET-8) ===

// SlotFairnessMetric represents fairness metrics for a single slot
type SlotFairnessMetric struct {
	SlotID       string  `json:"slotId"`
	Date         string  `json:"date"`
	MealType     string  `json:"mealType"`
	TokensServed int     `json:"tokensServed"`
	AvgWaitMin   float64 `json:"avgWaitMinutes"`
	MaxWaitMin   float64 `json:"maxWaitMinutes"`
	MinWaitMin   float64 `json:"minWaitMinutes"`
	StdDevWait   float64 `json:"stdDevWaitMinutes"`
}

// FairnessResponse represents the complete fairness analysis
type FairnessResponse struct {
	FairnessScore    int                  `json:"fairnessScore"`    // 0-100, higher = more equitable
	FIFOCompliance   float64              `json:"fifoCompliance"`   // % of tokens served in correct order
	AvgWaitMinutes   float64              `json:"avgWaitMinutes"`
	TotalTokens      int                  `json:"totalTokens"`
	SlotMetrics      []SlotFairnessMetric `json:"slotMetrics"`
	Period           string               `json:"period"`
}

// GetFairnessIndicators computes per-slot fairness metrics and overall equity score
// Accepts ?days=N (default 7) to control the analysis window.
func (h *Handler) GetFairnessIndicators(c *gin.Context) {
	days := 7
	if d := c.Query("days"); d != "" {
		if _, err := fmt.Sscanf(d, "%d", &days); err != nil || days < 1 {
			days = 7
		}
	}

	startDate := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)

	// Fetch all served tokens with their slot details in the period
	var tokens []models.QueueToken
	h.DB.Preload("Booking.Slot").
		Joins("JOIN bookings ON bookings.id = queue_tokens.booking_id").
		Joins("JOIN meal_slots ON meal_slots.id = bookings.slot_id").
		Where("meal_slots.date >= ?", startDate).
		Where("queue_tokens.status = ?", models.TokenServed).
		Order("queue_tokens.created_at").
		Find(&tokens)

	if len(tokens) == 0 {
		c.JSON(http.StatusOK, FairnessResponse{
			FairnessScore:  100,
			FIFOCompliance: 100,
			Period:         fmt.Sprintf("Last %d days", days),
		})
		return
	}

	// Group tokens by slot and compute wait times
	type slotData struct {
		slotID   string
		date     string
		mealType string
		waits    []float64 // wait times in minutes
		times    []time.Time // creation times for FIFO check
		served   []time.Time // served times for FIFO check
	}

	slotMap := make(map[string]*slotData)

	for _, t := range tokens {
		slotKey := t.Booking.SlotID.String()
		if slotMap[slotKey] == nil {
			slotMap[slotKey] = &slotData{
				slotID:   slotKey,
				date:     t.Booking.Slot.Date.Format("2006-01-02"),
				mealType: string(t.Booking.Slot.MealType),
			}
		}
		sd := slotMap[slotKey]

		// Calculate wait time: from token creation to when it was called/served
		waitEnd := t.CreatedAt
		if t.CalledAt != nil {
			waitEnd = *t.CalledAt
		}
		waitMinutes := waitEnd.Sub(t.CreatedAt).Minutes()
		if waitMinutes < 0 {
			waitMinutes = 0
		}

		sd.waits = append(sd.waits, waitMinutes)
		sd.times = append(sd.times, t.CreatedAt)
		if t.CalledAt != nil {
			sd.served = append(sd.served, *t.CalledAt)
		} else {
			sd.served = append(sd.served, t.CreatedAt)
		}
	}

	// Compute per-slot metrics
	var slotMetrics []SlotFairnessMetric
	var allWaits []float64
	fifoCorrect := 0
	fifoTotal := 0

	for _, sd := range slotMap {
		if len(sd.waits) == 0 {
			continue
		}

		sum := 0.0
		maxW := 0.0
		minW := math.MaxFloat64
		for _, w := range sd.waits {
			sum += w
			if w > maxW {
				maxW = w
			}
			if w < minW {
				minW = w
			}
		}
		avg := sum / float64(len(sd.waits))

		// Standard deviation
		variance := 0.0
		for _, w := range sd.waits {
			variance += (w - avg) * (w - avg)
		}
		stdDev := 0.0
		if len(sd.waits) > 1 {
			stdDev = math.Sqrt(variance / float64(len(sd.waits)-1))
		}

		slotMetrics = append(slotMetrics, SlotFairnessMetric{
			SlotID:       sd.slotID,
			Date:         sd.date,
			MealType:     sd.mealType,
			TokensServed: len(sd.waits),
			AvgWaitMin:   math.Round(avg*100) / 100,
			MaxWaitMin:   math.Round(maxW*100) / 100,
			MinWaitMin:   math.Round(minW*100) / 100,
			StdDevWait:   math.Round(stdDev*100) / 100,
		})

		allWaits = append(allWaits, sd.waits...)

		// Check FIFO compliance: tokens created earlier should be served earlier
		for i := 1; i < len(sd.times); i++ {
			fifoTotal++
			if sd.served[i].After(sd.served[i-1]) || sd.served[i].Equal(sd.served[i-1]) {
				fifoCorrect++
			}
		}
	}

	// Overall statistics
	overallAvg := 0.0
	if len(allWaits) > 0 {
		sum := 0.0
		for _, w := range allWaits {
			sum += w
		}
		overallAvg = sum / float64(len(allWaits))
	}

	fifoCompliance := 100.0
	if fifoTotal > 0 {
		fifoCompliance = float64(fifoCorrect) / float64(fifoTotal) * 100
	}

	// Compute fairness score (0-100)
	// Based on: low wait time variance (40%), high FIFO compliance (40%), low max-min gap (20%)
	varianceScore := 100.0
	if len(allWaits) > 1 {
		allVariance := 0.0
		for _, w := range allWaits {
			allVariance += (w - overallAvg) * (w - overallAvg)
		}
		stdDev := math.Sqrt(allVariance / float64(len(allWaits)-1))
		// Lower stddev = higher score; stddev > 10 min = poor
		varianceScore = math.Max(0, 100-stdDev*10)
	}

	maxMinGapScore := 100.0
	if len(allWaits) > 0 {
		maxW, minW := 0.0, math.MaxFloat64
		for _, w := range allWaits {
			if w > maxW { maxW = w }
			if w < minW { minW = w }
		}
		gap := maxW - minW
		maxMinGapScore = math.Max(0, 100-gap*5)
	}

	fairnessScore := int(varianceScore*0.4 + fifoCompliance*0.4 + maxMinGapScore*0.2)
	if fairnessScore > 100 {
		fairnessScore = 100
	}

	c.JSON(http.StatusOK, FairnessResponse{
		FairnessScore:  fairnessScore,
		FIFOCompliance: math.Round(fifoCompliance*100) / 100,
		AvgWaitMinutes: math.Round(overallAvg*100) / 100,
		TotalTokens:    len(tokens),
		SlotMetrics:    slotMetrics,
		Period:         fmt.Sprintf("Last %d days", days),
	})
}
