package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestUserRoleConstants(t *testing.T) {
	tests := []struct {
		role     UserRole
		expected string
	}{
		{RoleAdmin, "admin"},
		{RoleStudent, "student"},
		{RoleStaff, "staff"},
	}

	for _, tt := range tests {
		if string(tt.role) != tt.expected {
			t.Errorf("expected role %q, got %q", tt.expected, tt.role)
		}
	}
}

func TestHashPassword(t *testing.T) {
	user := &User{
		ID:       uuid.New(),
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "plaintext123",
	}

	err := user.hashPassword()
	if err != nil {
		t.Fatalf("hashPassword() returned error: %v", err)
	}

	// Password should now be a bcrypt hash (60 chars)
	if len(user.Password) != 60 {
		t.Errorf("expected hashed password length 60, got %d", len(user.Password))
	}

	// Should not be the original plaintext
	if user.Password == "plaintext123" {
		t.Error("password was not hashed")
	}
}

func TestHashPasswordSkipsAlreadyHashed(t *testing.T) {
	// First, create a real bcrypt hash
	user := &User{Password: "testpass"}
	_ = user.hashPassword()
	hashed := user.Password

	// Now try to hash again — it should stay the same
	err := user.hashPassword()
	if err != nil {
		t.Fatalf("hashPassword() returned error: %v", err)
	}

	if user.Password != hashed {
		t.Error("already-hashed password should not be re-hashed")
	}
}

func TestCheckPassword(t *testing.T) {
	user := &User{Password: "mypassword"}
	_ = user.hashPassword()

	if !user.CheckPassword("mypassword") {
		t.Error("CheckPassword should return true for correct password")
	}

	if user.CheckPassword("wrongpassword") {
		t.Error("CheckPassword should return false for incorrect password")
	}
}

func TestCheckPasswordEmpty(t *testing.T) {
	user := &User{Password: "secret"}
	_ = user.hashPassword()

	if user.CheckPassword("") {
		t.Error("CheckPassword should return false for empty password")
	}
}
