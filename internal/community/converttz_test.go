package community

import (
	"errors"
	"testing"
	"time"
)

func TestParseConvertTimeTimeOnlyUsesToday(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	utc, err := time.LoadLocation("UTC")
	if err != nil {
		t.Fatalf("load UTC: %v", err)
	}

	// July is daylight saving time in Berlin. A time-only value must keep
	// the offset in force on this date, not the winter offset of January 1.
	now := time.Date(2026, time.July, 15, 12, 0, 0, 0, loc)

	got, layout, err := parseConvertTime("15:04", loc, now)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if layout != "15:04" {
		t.Errorf("layout = %q, want %q", layout, "15:04")
	}
	if got.Hour() != 15 || got.Minute() != 4 {
		t.Errorf("time of day = %v, want 15:04", got)
	}
	if got.Year() != now.Year() || got.Month() != now.Month() || got.Day() != now.Day() {
		t.Errorf("date = %v, want %v", got, now)
	}
	if got.Location() != loc {
		t.Errorf("location = %v, want %v", got.Location(), loc)
	}

	wantUTC := "13:04"
	if gotUTC := got.In(utc).Format("15:04"); gotUTC != wantUTC {
		t.Errorf("UTC time = %v, want %v (winter offset would give 14:04)", gotUTC, wantUTC)
	}
}

func TestParseConvertTimeWithDateIsUnchanged(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	now := time.Date(2026, time.July, 15, 12, 0, 0, 0, loc)

	got, layout, err := parseConvertTime("2024-01-02 15:04:05", loc, now)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if layout != "2006-01-02 15:04:05" {
		t.Errorf("layout = %q, want %q", layout, "2006-01-02 15:04:05")
	}
	if got.Year() != 2024 || got.Month() != time.January || got.Day() != 2 ||
		got.Hour() != 15 || got.Minute() != 4 || got.Second() != 5 {
		t.Errorf("parsed time = %v, want 2024-01-02 15:04:05", got)
	}
}

func TestParseConvertTimeRejectsUnknownFormat(t *testing.T) {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		t.Fatalf("load UTC: %v", err)
	}

	_, _, err = parseConvertTime("not a time", loc, time.Now().In(loc))
	if !errors.Is(err, errUnrecognizedTime) {
		t.Errorf("error = %v, want %v", err, errUnrecognizedTime)
	}
}
