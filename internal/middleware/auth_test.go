package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smart-cafeteria/backend/internal/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAdminOnly_AllowsAdmin(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("role", models.RoleAdmin)

	handler := AdminOnly()
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if c.IsAborted() {
		t.Error("request should not be aborted for admin")
	}
}

func TestAdminOnly_BlocksStudent(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("role", models.RoleStudent)

	handler := AdminOnly()
	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
	if !c.IsAborted() {
		t.Error("request should be aborted for student")
	}
}

func TestAdminOnly_BlocksNoRole(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler := AdminOnly()
	handler(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestStaffOrAdmin_AllowsAdmin(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("role", models.RoleAdmin)

	handler := StaffOrAdmin()
	handler(c)

	if c.IsAborted() {
		t.Error("request should not be aborted for admin")
	}
}

func TestStaffOrAdmin_AllowsStaff(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("role", models.RoleStaff)

	handler := StaffOrAdmin()
	handler(c)

	if c.IsAborted() {
		t.Error("request should not be aborted for staff")
	}
}

func TestStaffOrAdmin_BlocksStudent(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("role", models.RoleStudent)

	handler := StaffOrAdmin()
	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestGetUserID_Exists(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	expectedID := uuid.New()
	c.Set("userID", expectedID)

	id, ok := GetUserID(c)
	if !ok {
		t.Error("GetUserID should return true when userID exists")
	}
	if id != expectedID {
		t.Errorf("expected %v, got %v", expectedID, id)
	}
}

func TestGetUserID_NotExists(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	id, ok := GetUserID(c)
	if ok {
		t.Error("GetUserID should return false when userID does not exist")
	}
	if id != uuid.Nil {
		t.Error("GetUserID should return uuid.Nil when not found")
	}
}

func TestGetUserRole_Exists(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("role", models.RoleStaff)

	role, ok := GetUserRole(c)
	if !ok {
		t.Error("GetUserRole should return true when role exists")
	}
	if role != models.RoleStaff {
		t.Errorf("expected staff, got %v", role)
	}
}

func TestGetUserRole_NotExists(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	role, ok := GetUserRole(c)
	if ok {
		t.Error("GetUserRole should return false when role does not exist")
	}
	if role != "" {
		t.Error("GetUserRole should return empty string when not found")
	}
}
