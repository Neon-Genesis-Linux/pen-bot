package db

import (
	"context"
	"sync"
	"time"
)

type memCacheEntry struct {
	data      []byte
	expiresAt time.Time
}

var (
	mc   = make(map[string]memCacheEntry)
	mcMu sync.RWMutex
)

// SetCacheEntry stores valueJSON under key for ttl.
//
// A non-positive ttl stores the value without an expiry, so it stays until
// DeleteCacheEntry removes it or the process exits.
func SetCacheEntry(key string, valueJSON []byte, ttl time.Duration) {
	mcMu.Lock()
	defer mcMu.Unlock()
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}
	mc[key] = memCacheEntry{data: valueJSON, expiresAt: expiresAt}
}

// GetCacheEntry returns the value stored under key, and whether it was found.
//
// An expired entry is deleted and reported as absent, so callers never see a
// value past its time to live.
func GetCacheEntry(key string) ([]byte, bool) {
	mcMu.RLock()
	e, ok := mc[key]
	mcMu.RUnlock()
	if !ok {
		return nil, false
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		mcMu.Lock()
		delete(mc, key)
		mcMu.Unlock()
		return nil, false
	}
	return e.data, true
}

// DeleteCacheEntry removes the value stored under key, if any.
func DeleteCacheEntry(key string) {
	mcMu.Lock()
	defer mcMu.Unlock()
	delete(mc, key)
}

// StartCleanup removes expired entries until ctx is cancelled.
//
// It is not required for correctness: GetCacheEntry drops an expired entry
// when it is read. Cleanup exists to stop expired values accumulating when a
// key is never read again.
func StartCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mcMu.Lock()
			now := time.Now()
			for k, e := range mc {
				if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
					delete(mc, k)
				}
			}
			mcMu.Unlock()
		}
	}
}
