package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smart-cafeteria/backend/internal/middleware"
	"github.com/smart-cafeteria/backend/internal/models"
)

// === Incentive Rules (Admin) ===

// GetIncentiveRules returns all incentive rules
func (h *Handler) GetIncentiveRules(c *gin.Context) {
	var rules []models.IncentiveRule
	if err := h.DB.Order("created_at DESC").Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch incentive rules"})
		return
	}
	c.JSON(http.StatusOK, rules)
}

// CreateIncentiveRuleRequest represents the request body
type CreateIncentiveRuleRequest struct {
	Name             string `json:"name" binding:"required"`
	Description      string `json:"description"`
	SlotType         string `json:"slotType"`
	MaxOccupancyPct  int    `json:"maxOccupancyPct"`
	BonusPoints      int    `json:"bonusPoints"`
	BaseAttendPoints int    `json:"baseAttendPoints"`
	NoShowPenalty    int    `json:"noShowPenalty"`
	IsActive         bool   `json:"isActive"`
}

// CreateIncentiveRule creates a new incentive rule
func (h *Handler) CreateIncentiveRule(c *gin.Context) {
	var req CreateIncentiveRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule := models.IncentiveRule{
		Name:             req.Name,
		Description:      req.Description,
		SlotType:         models.MealType(req.SlotType),
		MaxOccupancyPct:  req.MaxOccupancyPct,
		BonusPoints:      req.BonusPoints,
		BaseAttendPoints: req.BaseAttendPoints,
		NoShowPenalty:    req.NoShowPenalty,
		IsActive:         req.IsActive,
	}

	// Set defaults if not provided
	if rule.MaxOccupancyPct == 0 {
		rule.MaxOccupancyPct = 50
	}
	if rule.BonusPoints == 0 {
		rule.BonusPoints = 10
	}
	if rule.BaseAttendPoints == 0 {
		rule.BaseAttendPoints = 5
	}
	if rule.NoShowPenalty == 0 {
		rule.NoShowPenalty = 10
	}

	if err := h.DB.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create incentive rule"})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// UpdateIncentiveRule updates an existing rule
func (h *Handler) UpdateIncentiveRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rule ID"})
		return
	}

	var rule models.IncentiveRule
	if err := h.DB.First(&rule, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	var req CreateIncentiveRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule.Name = req.Name
	rule.Description = req.Description
	rule.SlotType = models.MealType(req.SlotType)
	rule.MaxOccupancyPct = req.MaxOccupancyPct
	rule.BonusPoints = req.BonusPoints
	rule.BaseAttendPoints = req.BaseAttendPoints
	rule.NoShowPenalty = req.NoShowPenalty
	rule.IsActive = req.IsActive

	if err := h.DB.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update rule"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// DeleteIncentiveRule deletes a rule
func (h *Handler) DeleteIncentiveRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rule ID"})
		return
	}

	if err := h.DB.Delete(&models.IncentiveRule{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rule deleted successfully"})
}

// === User Points ===

// GetMyPoints returns the current user's points balance
func (h *Handler) GetMyPoints(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var userPoints models.UserPoints
	result := h.DB.Where("user_id = ?", userID).First(&userPoints)

	if result.Error != nil {
		// Create points record if doesn't exist
		userPoints = models.UserPoints{
			UserID:      userID,
			TotalPoints: 0,
		}
		h.DB.Create(&userPoints)
	}

	c.JSON(http.StatusOK, gin.H{
		"userId":      userID,
		"totalPoints": userPoints.TotalPoints,
	})
}

// GetPointsHistory returns the user's points transaction history
func (h *Handler) GetPointsHistory(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var transactions []models.PointTransaction
	if err := h.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(50).
		Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

// GetIncentiveStatus returns a summary of user's incentive status
func (h *Handler) GetIncentiveStatus(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get total points
	var userPoints models.UserPoints
	h.DB.Where("user_id = ?", userID).First(&userPoints)

	// Get attendance stats
	var totalBookings int64
	var attendedBookings int64
	var noShowBookings int64

	h.DB.Model(&models.AttendanceLog{}).Where("user_id = ?", userID).Count(&totalBookings)
	h.DB.Model(&models.AttendanceLog{}).Where("user_id = ? AND status = ?", userID, models.AttendanceAttended).Count(&attendedBookings)
	h.DB.Model(&models.AttendanceLog{}).Where("user_id = ? AND status = ?", userID, models.AttendanceNoShow).Count(&noShowBookings)

	// Get recent transactions
	var recentTransactions []models.PointTransaction
	h.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(5).
		Find(&recentTransactions)

	// Calculate attendance rate
	var attendanceRate float64 = 100
	if totalBookings > 0 {
		attendanceRate = float64(attendedBookings) / float64(totalBookings) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"totalPoints":        userPoints.TotalPoints,
		"totalBookings":      totalBookings,
		"attendedBookings":   attendedBookings,
		"noShowBookings":     noShowBookings,
		"attendanceRate":     attendanceRate,
		"recentTransactions": recentTransactions,
	})
}

// === Attendance & Points Awarding ===

// AwardAttendancePoints awards points when a booking is served
func (h *Handler) AwardAttendancePoints(userID, bookingID, slotID uuid.UUID, hasIncentive bool, incentivePoints int) error {
	// Get active rule (use first active rule for now)
	var rule models.IncentiveRule
	if err := h.DB.Where("is_active = ?", true).First(&rule).Error; err != nil {
		// Use defaults if no rule
		rule.BaseAttendPoints = 5
		rule.BonusPoints = 10
	}

	totalPoints := rule.BaseAttendPoints
	reason := "Attendance reward"

	if hasIncentive {
		bonusPoints := incentivePoints
		if bonusPoints == 0 {
			bonusPoints = rule.BonusPoints
		}
		totalPoints += bonusPoints
		reason = "Attendance + off-peak bonus"
	}

	// Record attendance log
	now := time.Now()
	attendanceLog := models.AttendanceLog{
		UserID:        userID,
		BookingID:     bookingID,
		SlotID:        slotID,
		Status:        models.AttendanceAttended,
		PointsAwarded: totalPoints,
		CheckedAt:     &now,
	}
	h.DB.Create(&attendanceLog)

	// Record point transaction
	transaction := models.PointTransaction{
		UserID:    userID,
		BookingID: &bookingID,
		Points:    totalPoints,
		Type:      models.PointsAttendance,
		Reason:    reason,
	}
	h.DB.Create(&transaction)

	// Update or create user points
	var userPoints models.UserPoints
	result := h.DB.Where("user_id = ?", userID).First(&userPoints)
	if result.Error != nil {
		userPoints = models.UserPoints{
			UserID:      userID,
			TotalPoints: totalPoints,
		}
		h.DB.Create(&userPoints)
	} else {
		userPoints.TotalPoints += totalPoints
		h.DB.Save(&userPoints)
	}

	// Auto-block user if points reach -100 or below
	if userPoints.TotalPoints <= -100 {
		h.DB.Model(&models.User{}).Where("id = ?", userID).Update("blocked", true)
	}

	return nil
}

// RecordNoShow records a no-show and applies penalty
func (h *Handler) RecordNoShow(userID, bookingID, slotID uuid.UUID) error {
	// Get active rule
	var rule models.IncentiveRule
	if err := h.DB.Where("is_active = ?", true).First(&rule).Error; err != nil {
		rule.NoShowPenalty = 10
	}

	// Record attendance log
	attendanceLog := models.AttendanceLog{
		UserID:        userID,
		BookingID:     bookingID,
		SlotID:        slotID,
		Status:        models.AttendanceNoShow,
		PointsAwarded: -rule.NoShowPenalty,
	}
	h.DB.Create(&attendanceLog)

	// Record penalty transaction
	transaction := models.PointTransaction{
		UserID:    userID,
		BookingID: &bookingID,
		Points:    -rule.NoShowPenalty,
		Type:      models.PointsNoShowPenalty,
		Reason:    "No-show penalty",
	}
	h.DB.Create(&transaction)

	// Update user points
	var userPoints models.UserPoints
	result := h.DB.Where("user_id = ?", userID).First(&userPoints)
	if result.Error != nil {
		userPoints = models.UserPoints{
			UserID:      userID,
			TotalPoints: -rule.NoShowPenalty,
		}
		h.DB.Create(&userPoints)
	} else {
		userPoints.TotalPoints -= rule.NoShowPenalty
		h.DB.Save(&userPoints)
	}

	// Auto-block user if points reach -100 or below
	if userPoints.TotalPoints <= -100 {
		h.DB.Model(&models.User{}).Where("id = ?", userID).Update("blocked", true)
	}

	return nil
}

// === Incentive Application ===

// ApplyIncentivesToSlots marks eligible slots with incentives
func (h *Handler) ApplyIncentivesToSlots(c *gin.Context) {
	// Get active rules
	var rules []models.IncentiveRule
	if err := h.DB.Where("is_active = ?", true).Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rules"})
		return
	}

	if len(rules) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No active rules", "updated": 0})
		return
	}

	// Get today's and future slots
	today := time.Now().Truncate(24 * time.Hour)
	var slots []models.MealSlot
	h.DB.Where("date >= ?", today).Find(&slots)

	updated := 0
	for _, slot := range slots {
		// Calculate occupancy percentage
		occupancyPct := 0
		if slot.Capacity > 0 {
			occupancyPct = (slot.BookedCount * 100) / slot.Capacity
		}

		// Check against rules
		hasIncentive := false
		maxPoints := 0

		for _, rule := range rules {
			// Check if rule applies to this slot type
			if rule.SlotType != "" && rule.SlotType != slot.MealType {
				continue
			}

			// Check occupancy threshold
			if occupancyPct < rule.MaxOccupancyPct {
				hasIncentive = true
				if rule.BonusPoints > maxPoints {
					maxPoints = rule.BonusPoints
				}
			}
		}

		// Update slot if incentive status changed
		if slot.HasIncentive != hasIncentive || slot.IncentivePoints != maxPoints {
			slot.HasIncentive = hasIncentive
			slot.IncentivePoints = maxPoints
			h.DB.Save(&slot)
			updated++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Incentives applied",
		"slotsChecked": len(slots),
		"slotsUpdated": updated,
	})
}

// === Abuse Detection ===

// GetAbuseReport returns potential abuse cases
func (h *Handler) GetAbuseReport(c *gin.Context) {
	// Find users with high no-show rates
	type AbuseRecord struct {
		UserID        uuid.UUID `json:"userId"`
		UserName      string    `json:"userName"`
		TotalBookings int64     `json:"totalBookings"`
		NoShows       int64     `json:"noShows"`
		NoShowRate    float64   `json:"noShowRate"`
	}

	var records []AbuseRecord

	// Raw query to get users with >30% no-show rate and at least 3 bookings
	rows, err := h.DB.Raw(`
		SELECT 
			u.id as user_id,
			u.name as user_name,
			COUNT(a.id) as total_bookings,
			SUM(CASE WHEN a.status = 'no-show' THEN 1 ELSE 0 END) as no_shows
		FROM users u
		JOIN attendance_logs a ON u.id = a.user_id
		GROUP BY u.id, u.name
		HAVING COUNT(a.id) >= 3
		ORDER BY no_shows DESC
	`).Rows()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var r AbuseRecord
		rows.Scan(&r.UserID, &r.UserName, &r.TotalBookings, &r.NoShows)
		if r.TotalBookings > 0 {
			r.NoShowRate = float64(r.NoShows) / float64(r.TotalBookings) * 100
		}
		if r.NoShowRate >= 30 {
			records = append(records, r)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"abuseRecords": records,
		"threshold":    "30% no-show rate with 3+ bookings",
	})
}

// === Behavior Trend Analytics (US-IN-8) ===

// WeeklyBehaviorData represents behavior data for one week
type WeeklyBehaviorData struct {
	WeekStart      string  `json:"weekStart"`
	WeekEnd        string  `json:"weekEnd"`
	TotalBookings  int     `json:"totalBookings"`
	Attended       int     `json:"attended"`
	NoShows        int     `json:"noShows"`
	PointsAwarded  int     `json:"pointsAwarded"`
	AttendanceRate float64 `json:"attendanceRate"`
}

// BehaviorSummary represents overall behavior analysis
type BehaviorSummary struct {
	TotalUsers         int     `json:"totalUsers"`
	AvgAttendanceRate  float64 `json:"avgAttendanceRate"`
	TrendDirection     string  `json:"trendDirection"` // "improving", "declining", "stable"
	TotalPointsAwarded int     `json:"totalPointsAwarded"`
	TotalNoShows       int     `json:"totalNoShows"`
}

// GetBehaviorTrends returns time-series behavioral analysis (admin only)
// Accepts ?days=N (default 30) to control the analysis window.
func (h *Handler) GetBehaviorTrends(c *gin.Context) {
	days := 30
	if d := c.Query("days"); d != "" {
		if _, err := fmt.Sscanf(d, "%d", &days); err != nil || days < 7 {
			days = 30
		}
	}

	startDate := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)

	// Fetch attendance logs for the period
	var logs []models.AttendanceLog
	h.DB.Where("created_at >= ?", startDate).Order("created_at").Find(&logs)

	// Group logs into weekly buckets
	weeklyMap := make(map[string]*WeeklyBehaviorData)
	for _, log := range logs {
		// Calculate the Monday of the week for this log
		weekday := log.CreatedAt.Weekday()
		offset := int(weekday - time.Monday)
		if offset < 0 {
			offset += 7
		}
		monday := log.CreatedAt.AddDate(0, 0, -offset).Truncate(24 * time.Hour)
		key := monday.Format("2006-01-02")

		if weeklyMap[key] == nil {
			weeklyMap[key] = &WeeklyBehaviorData{
				WeekStart: monday.Format("2006-01-02"),
				WeekEnd:   monday.AddDate(0, 0, 6).Format("2006-01-02"),
			}
		}

		week := weeklyMap[key]
		week.TotalBookings++
		week.PointsAwarded += log.PointsAwarded
		if log.Status == models.AttendanceAttended {
			week.Attended++
		} else if log.Status == models.AttendanceNoShow {
			week.NoShows++
		}
	}

	// Build sorted weekly data and compute rates
	var weeklyData []WeeklyBehaviorData
	var keys []string
	for k := range weeklyMap {
		keys = append(keys, k)
	}
	// Sort keys chronologically
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	for _, k := range keys {
		w := weeklyMap[k]
		if w.TotalBookings > 0 {
			w.AttendanceRate = float64(w.Attended) / float64(w.TotalBookings) * 100
		}
		weeklyData = append(weeklyData, *w)
	}

	// Compute overall summary
	totalAttended := 0
	totalNoShows := 0
	totalPoints := 0
	for _, w := range weeklyData {
		totalAttended += w.Attended
		totalNoShows += w.NoShows
		totalPoints += w.PointsAwarded
	}

	totalBookings := totalAttended + totalNoShows
	avgAttendance := 0.0
	if totalBookings > 0 {
		avgAttendance = float64(totalAttended) / float64(totalBookings) * 100
	}

	// Determine trend direction by comparing first half vs second half
	trendDirection := "stable"
	if len(weeklyData) >= 2 {
		mid := len(weeklyData) / 2
		firstHalfRate := 0.0
		secondHalfRate := 0.0
		for _, w := range weeklyData[:mid] {
			firstHalfRate += w.AttendanceRate
		}
		for _, w := range weeklyData[mid:] {
			secondHalfRate += w.AttendanceRate
		}
		firstHalfRate /= float64(mid)
		secondHalfRate /= float64(len(weeklyData) - mid)

		if secondHalfRate > firstHalfRate+5 {
			trendDirection = "improving"
		} else if secondHalfRate < firstHalfRate-5 {
			trendDirection = "declining"
		}
	}

	// Count unique users
	var uniqueUsers int64
	h.DB.Model(&models.AttendanceLog{}).Where("created_at >= ?", startDate).
		Distinct("user_id").Count(&uniqueUsers)

	c.JSON(http.StatusOK, gin.H{
		"weeklyTrends": weeklyData,
		"summary": BehaviorSummary{
			TotalUsers:         int(uniqueUsers),
			AvgAttendanceRate:  avgAttendance,
			TrendDirection:     trendDirection,
			TotalPointsAwarded: totalPoints,
			TotalNoShows:       totalNoShows,
		},
		"period": fmt.Sprintf("Last %d days", days),
	})
}
