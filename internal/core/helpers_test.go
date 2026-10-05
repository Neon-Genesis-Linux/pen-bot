package core

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	const zeroSeconds = "0 seconds"

	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "zero duration",
			duration: 0,
			want:     zeroSeconds,
		},
		{
			name:     "negative duration",
			duration: -5 * time.Second,
			want:     zeroSeconds,
		},
		{
			name:     "sub-second rounds down",
			duration: 400 * time.Millisecond,
			want:     zeroSeconds,
		},
		{
			name:     "single second singular",
			duration: 1 * time.Second,
			want:     "1 second",
		},
		{
			name:     "single unit plural",
			duration: 15 * time.Minute,
			want:     "15 minutes",
		},
		{
			name:     "two units joined with and",
			duration: 1*time.Hour + 30*time.Minute,
			want:     "1 hour and 30 minutes",
		},
		{
			name:     "complex multi-unit duration",
			duration: 5*24*time.Hour + 2*time.Hour + 45*time.Minute + 10*time.Second,
			want:     "5 days, 2 hours, 45 minutes and 10 seconds",
		},
		{
			name:     "skipped units",
			duration: 2*24*time.Hour + 15*time.Second,
			want:     "2 days and 15 seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("FormatDuration(%v) = %q; want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestParseDurationExt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"seconds", "30s", 30 * time.Second, false},
		{"minutes and hours", "1h30m", 90 * time.Minute, false},
		{"rounding sub-seconds down", "10s400ms", 10 * time.Second, false},
		{"rounding sub-seconds up", "10s600ms", 11 * time.Second, false},

		{"single day", "1d", 24 * time.Hour, false},
		{"multiple days", "3d", 72 * time.Hour, false},
		{"single week", "1w", 7 * 24 * time.Hour, false},
		{"fractional day", "1.5d", 36 * time.Hour, false},

		{"week and day", "1w2d", (7*24 + 2*24) * time.Hour, false},
		{"all units combined", "1d2h30m", (26 * time.Hour) + 30*time.Minute, false},

		{"empty string", "", 0, true},
		{"invalid text", "tomorrow", 0, true},
		{"invalid unit", "10y", 0, true},
		{"malformed decimals", "1.5.5d", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDurationExt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseDurationExt(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ParseDurationExt(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
