package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/Neon-Genesis-Linux/pen-bot/internal/db"
)

func TestSetGet(t *testing.T) {
	const key = "test-set-get"
	t.Cleanup(func() { db.DeleteCacheEntry(key) })

	db.SetCacheEntry(key, []byte("v"), 0)
	got, ok := db.GetCacheEntry(key)
	if !ok {
		t.Fatal("expected ok")
	}
	if string(got) != "v" {
		t.Fatalf("got %q, want %q", string(got), "v")
	}

	db.DeleteCacheEntry(key)
	_, ok = db.GetCacheEntry(key)
	if ok {
		t.Fatal("expected false after delete")
	}
}

func TestExpiry(t *testing.T) {
	const key = "test-expiry"
	t.Cleanup(func() { db.DeleteCacheEntry(key) })

	db.SetCacheEntry(key, []byte("v"), time.Millisecond)
	time.Sleep(2 * time.Millisecond)
	_, ok := db.GetCacheEntry(key)
	if ok {
		t.Fatal("expected false after expiry")
	}
}

func TestNoExpiryWhenTTLZero(t *testing.T) {
	const key = "test-no-expiry"
	t.Cleanup(func() { db.DeleteCacheEntry(key) })

	db.SetCacheEntry(key, []byte("v"), 0)
	got, ok := db.GetCacheEntry(key)
	if !ok {
		t.Fatal("expected ok for TTL=0")
	}
	if string(got) != "v" {
		t.Fatalf("got %q, want %q", string(got), "v")
	}
}

func TestOverwrite(t *testing.T) {
	const key = "test-overwrite"
	t.Cleanup(func() { db.DeleteCacheEntry(key) })

	db.SetCacheEntry(key, []byte("v1"), 0)
	db.SetCacheEntry(key, []byte("v2"), 0)
	got, ok := db.GetCacheEntry(key)
	if !ok {
		t.Fatal("expected ok")
	}
	if string(got) != "v2" {
		t.Fatalf("got %q, want %q", string(got), "v2")
	}
}

func TestCleanup(t *testing.T) {
	expireKey := "test-cleanup-expire"
	keepKey := "test-cleanup-keep"
	t.Cleanup(func() {
		db.DeleteCacheEntry(expireKey)
		db.DeleteCacheEntry(keepKey)
	})

	db.SetCacheEntry(expireKey, []byte("x"), time.Millisecond)
	db.SetCacheEntry(keepKey, []byte("y"), 0)

	time.Sleep(2 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go db.StartCleanup(ctx, time.Microsecond)
	time.Sleep(10 * time.Millisecond)

	_, ok := db.GetCacheEntry(expireKey)
	if ok {
		t.Fatal("expected expired entry removed by cleanup")
	}
	_, ok = db.GetCacheEntry(keepKey)
	if !ok {
		t.Fatal("expected non-expired entry kept")
	}
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()

	done := make(chan struct{}, 2)
	go func() {
		for range 100 {
			db.SetCacheEntry("race-key", []byte("v"), time.Minute)
		}
		done <- struct{}{}
	}()
	go func() {
		for range 100 {
			db.GetCacheEntry("race-key")
		}
		done <- struct{}{}
	}()
	<-done
	<-done
}
