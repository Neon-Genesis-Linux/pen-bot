package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstanceFileName(t *testing.T) {
	const defaultName = "pen-bot-default.db"

	tests := []struct {
		name       string
		instanceID string
		want       string
	}{
		{name: "plain", instanceID: "pen-bot-1", want: "pen-bot-pen-bot-1.db"},
		{name: "empty falls back", instanceID: "", want: defaultName},
		{name: "only unsafe characters falls back", instanceID: "///", want: defaultName},
		{name: "unsafe characters are dropped", instanceID: "my instance/../../evil", want: "pen-bot-myinstanceevil.db"},
		{name: "dots are dropped", instanceID: "..", want: defaultName},
		{name: "underscores are kept", instanceID: "a_b-c1", want: "pen-bot-a_b-c1.db"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := instanceFileName(tt.instanceID)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
			if strings.ContainsAny(got, `/\`) {
				t.Errorf("got %q, want no path separator", got)
			}
			if strings.Contains(got, "..") {
				t.Errorf("got %q, want no parent directory reference", got)
			}
		})
	}
}

// TestDSNResolvesToAnAbsolutePath checks that the connection string names one
// file whatever the working directory is.
func TestDSNResolvesToAnAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pen-bot.db")

	got, err := dsn(path)
	if err != nil {
		t.Fatalf("dsn: %v", err)
	}

	if !strings.HasPrefix(got, "file:///") {
		t.Errorf("got %q, want an absolute file URI", got)
	}
	if !strings.Contains(got, "pen-bot.db") {
		t.Errorf("got %q, want it to name the database file", got)
	}
	for _, want := range []string{"_pragma=journal_mode(WAL)", "_time_integer_format=unix_milli", "_inttotime=1", "_txlock=immediate"} {
		if !strings.Contains(got, want) {
			t.Errorf("got %q, want it to contain %q", got, want)
		}
	}
}

// TestDSNCreatesTheParentDirectory checks that the directory holding the
// database is created, because SQLite does not create it itself.
func TestDSNCreatesTheParentDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "deeper")
	path := filepath.Join(dir, "pen-bot.db")

	if _, err := dsn(path); err != nil {
		t.Fatalf("dsn: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat parent directory: %v", err)
	}
	if !info.IsDir() {
		t.Error("got a file, want a directory")
	}
}
