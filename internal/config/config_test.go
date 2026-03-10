package config

import (
	"os"
	"testing"
)

func TestGetEnv_ReturnsEnvValue(t *testing.T) {
	os.Setenv("TEST_KEY_123", "from_env")
	defer os.Unsetenv("TEST_KEY_123")

	val := GetEnv("TEST_KEY_123", "default_val")
	if val != "from_env" {
		t.Errorf("expected 'from_env', got %q", val)
	}
}

func TestGetEnv_ReturnsDefault(t *testing.T) {
	os.Unsetenv("TEST_KEY_MISSING")

	val := GetEnv("TEST_KEY_MISSING", "fallback")
	if val != "fallback" {
		t.Errorf("expected 'fallback', got %q", val)
	}
}

func TestGetEnv_EmptyEnvReturnsDefault(t *testing.T) {
	os.Setenv("TEST_KEY_EMPTY", "")
	defer os.Unsetenv("TEST_KEY_EMPTY")

	val := GetEnv("TEST_KEY_EMPTY", "default")
	if val != "default" {
		t.Errorf("expected 'default' for empty env, got %q", val)
	}
}
