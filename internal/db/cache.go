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

func SetCacheEntry(key string, valueJSON []byte, ttl time.Duration) {
	mcMu.Lock()
	defer mcMu.Unlock()
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}
	mc[key] = memCacheEntry{data: valueJSON, expiresAt: expiresAt}
}

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

func DeleteCacheEntry(key string) {
	mcMu.Lock()
	defer mcMu.Unlock()
	delete(mc, key)
}

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
