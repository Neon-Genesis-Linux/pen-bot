package db

import (
	"testing"
	"time"
)

func TestGetEnv(t *testing.T) {
	t.Setenv("TEST_GETENV_KEY", "val")
	if got := getEnv("TEST_GETENV_KEY", "fall"); got != "val" {
		t.Fatalf("got %q, want %q", got, "val")
	}
	if got := getEnv("TEST_GETENV_MISSING", "fall"); got != "fall" {
		t.Fatalf("got %q, want %q", got, "fall")
	}
}

func TestParseIntEnv(t *testing.T) {
	t.Setenv("TEST_INT_VALID", "42")
	if got := parseIntEnv("TEST_INT_VALID", 1); got != 42 {
		t.Fatalf("got %d, want %d", got, 42)
	}
	if got := parseIntEnv("TEST_INT_MISSING", 1); got != 1 {
		t.Fatalf("got %d, want %d", got, 1)
	}
	t.Setenv("TEST_INT_INVALID", "notanumber")
	if got := parseIntEnv("TEST_INT_INVALID", 5); got != 5 {
		t.Fatalf("got %d, want %d", got, 5)
	}
}

func TestParseDurationEnv(t *testing.T) {
	want := 5 * time.Minute
	t.Setenv("TEST_DUR_VALID", "5m")
	if got := parseDurationEnv("TEST_DUR_VALID", time.Hour); got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got := parseDurationEnv("TEST_DUR_MISSING", time.Hour); got != time.Hour {
		t.Fatalf("got %v, want %v", got, time.Hour)
	}
}

func TestParseCleanupIntervalEnv(t *testing.T) {
	want := 10 * time.Minute
	t.Setenv("TEST_CLEANUP", "10m")
	if got := ParseCleanupIntervalEnv("TEST_CLEANUP", 15); got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got := ParseCleanupIntervalEnv("TEST_CLEANUP_MISSING", 15); got != 15*time.Minute {
		t.Fatalf("got %v, want %v", got, 15*time.Minute)
	}
}
