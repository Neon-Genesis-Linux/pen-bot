package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Neon-Genesis-Linux/pen-bot/internal/db"
)

// testModule is the module name every scope registered by these tests claims.
const testModule = "test"

// settings is the shape the framework tests register. It has one field whose
// default must survive a stored document that leaves it out, and one that must be
// overwritten by it.
type settings struct {
	Enabled bool           `json:"enabled"`
	Limit   int            `json:"limit"`
	Roles   map[string]int `json:"roles"`
}

// otherSettings exists so a scope can be read back as the wrong type.
type otherSettings struct {
	Enabled bool `json:"enabled"`
}

// sameSettings compares two settings values. The struct holds a map, so it
// cannot be compared with ==.
func sameSettings(got, want settings) bool {
	return reflect.DeepEqual(got, want)
}

func defaultSettings() settings {
	return settings{
		Enabled: true,
		Limit:   7,
		Roles:   map[string]int{"kept": 1},
	}
}

func validateSettings(s settings) error {
	if s.Limit < 0 {
		return fmt.Errorf("limit must not be negative, got %d", s.Limit)
	}
	return nil
}

// fakeStorage stands in for a real database so the framework's read, write and cache
// behaviour can be tested without one.
type fakeStorage struct {
	mu        sync.Mutex
	documents map[string][]byte
	reads     int
	writes    int
	resets    int
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{documents: make(map[string][]byte)}
}

func (f *fakeStorage) storage() storage {
	return storage{
		load: func(_ context.Context, guildID string) (json.RawMessage, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.reads++
			return json.RawMessage(f.documents[guildID]), nil
		},
		save: func(_ context.Context, guildID string, raw json.RawMessage) error {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.writes++
			f.documents[guildID] = append([]byte(nil), raw...)
			return nil
		},
		reset: func(_ context.Context, guildID string) (bool, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.resets++
			_, stored := f.documents[guildID]
			delete(f.documents, guildID)
			return stored, nil
		},
	}
}

func (f *fakeStorage) put(guildID, document string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.documents[guildID] = []byte(document)
}

func (f *fakeStorage) counts() (reads, writes, resets int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reads, f.writes, f.resets
}

// guildSeq keeps every test on its own guild. The cache the framework reads from
// is process wide and outlives a test, so reusing an ID would let one test's
// stored document answer another's.
var guildSeq atomic.Int64

func testGuildID() string {
	return strconv.FormatInt(900000000000000000+guildSeq.Add(1), 10)
}

// scopeSeq keeps every test on its own scope key, for the same reason.
var scopeSeq atomic.Int64

func testScopeKey() string {
	return "test-" + strconv.FormatInt(scopeSeq.Add(1), 10)
}

// registerTestScope registers a scope backed by the given storage and returns
// its key.
func registerTestScope(t *testing.T, store *fakeStorage) string {
	t.Helper()

	key := testScopeKey()
	err := register(ScopeDesc[settings]{
		Key:      key,
		Module:   testModule,
		Summary:  "Settings for the framework tests.",
		Defaults: defaultSettings(),
		Validate: validateSettings,
	}, store.storage())
	if err != nil {
		t.Fatalf("registering scope %q failed: %v", key, err)
	}
	return key
}

func TestGetOverlaysTheStoredDocumentOnTheDefaults(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)
	guildID := testGuildID()
	store.put(guildID, `{"limit":9,"roles":{"given":2}}`)

	got, err := Get[settings](t.Context(), key, guildID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// The default map entry survives: json decodes an object into the map it
	// finds rather than replacing it.
	want := settings{Enabled: true, Limit: 9, Roles: map[string]int{"given": 2, "kept": 1}}
	if !sameSettings(got, want) {
		t.Errorf("Get = %+v, want %+v", got, want)
	}
}

func TestGetKeepsADefaultKeyTheDocumentMergesInto(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)
	guildID := testGuildID()
	store.put(guildID, `{"limit":9}`)

	got, err := Get[settings](t.Context(), key, guildID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !got.Enabled {
		t.Error("Enabled = false, want the registered default: the document leaves the key out")
	}
	if len(got.Roles) != 1 || got.Roles["kept"] != 1 {
		t.Errorf("Roles = %v, want the registered default preserved", got.Roles)
	}
}

func TestGetWithoutAStoredDocumentReturnsTheDefaults(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)

	got, err := Get[settings](t.Context(), key, testGuildID())
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !sameSettings(got, defaultSettings()) {
		t.Errorf("Get = %+v, want the registered defaults %+v", got, defaultSettings())
	}
}

func TestGetFallsBackWhenTheDocumentCannotBeDecoded(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)
	guildID := testGuildID()
	store.put(guildID, `{"limit":"not a number"}`)

	got, err := Get[settings](t.Context(), key, guildID)
	if err != nil {
		t.Fatalf("Get = %v, want a document too broken to decode to fall back to the defaults", err)
	}
	if !sameSettings(got, defaultSettings()) {
		t.Errorf("Get = %+v, want the registered defaults %+v", got, defaultSettings())
	}
}

func TestGetFallsBackWhenTheDocumentFailsValidation(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)
	guildID := testGuildID()
	store.put(guildID, `{"limit":-1}`)

	got, err := Get[settings](t.Context(), key, guildID)
	if err != nil {
		t.Fatalf("Get = %v, want an invalid document to fall back to the defaults", err)
	}
	if !sameSettings(got, defaultSettings()) {
		t.Errorf("Get = %+v, want the registered defaults %+v", got, defaultSettings())
	}
}

func TestGetRejectsAnUnregisteredScope(t *testing.T) {
	_, err := Get[settings](t.Context(), "no-such-scope", testGuildID())
	if err == nil {
		t.Fatal("Get = nil error, want an error for an unregistered scope")
	}
}

func TestGetRejectsTheWrongSettingsType(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)

	_, err := Get[otherSettings](t.Context(), key, testGuildID())
	if err == nil {
		t.Fatal("Get = nil error, want an error when the scope holds another settings type")
	}
}

func TestRegisterRejectsADuplicateKey(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)

	err := register(ScopeDesc[settings]{
		Key:      key,
		Module:   testModule,
		Summary:  "A second registration under the same key.",
		Defaults: defaultSettings(),
		Validate: validateSettings,
	}, store.storage())
	if err == nil {
		t.Fatal("register = nil error, want an error for a key already taken")
	}
}

func TestRegisterRejectsAnEmptyKey(t *testing.T) {
	store := newFakeStorage()

	err := register(ScopeDesc[settings]{
		Key:      "",
		Module:   testModule,
		Summary:  "A registration with no key.",
		Defaults: defaultSettings(),
		Validate: validateSettings,
	}, store.storage())
	if err == nil {
		t.Fatal("register = nil error, want an error for an empty key")
	}
}

func TestRegisterRejectsIncompleteStorage(t *testing.T) {
	err := register(ScopeDesc[settings]{
		Key:      testScopeKey(),
		Module:   testModule,
		Summary:  "A registration with nowhere to store its settings.",
		Defaults: defaultSettings(),
		Validate: validateSettings,
	}, storage{
		load: func(_ context.Context, _ string) (json.RawMessage, error) { return nil, nil },
	})
	if err == nil {
		t.Fatal("register = nil error, want an error when save and reset are missing")
	}
}

func TestGetWithoutADatabaseReportsTheError(t *testing.T) {
	// The scope loads through a database handle the bot does not have, which is
	// what a bot with no database handle sees. Reads fail closed rather than
	// serving defaults that may be stale.
	unavailable := registerScopeLoading(errNoDatabase, t)

	got, err := Get[settings](t.Context(), unavailable, testGuildID())
	if err == nil {
		t.Fatalf("Get = %v, want the error when the database is unavailable", got)
	}
	if !errors.Is(err, errNoDatabase) {
		t.Fatalf("Get error = %v, want %v", err, errNoDatabase)
	}
}

func TestUpdateWithoutADatabaseReportsTheError(t *testing.T) {
	unavailable := registerScopeLoading(errNoDatabase, t)

	err := Update(t.Context(), unavailable, testGuildID(), func(s *settings) error {
		s.Limit = 1
		return nil
	})
	if !errors.Is(err, errNoDatabase) {
		t.Fatalf("Update = %v, want %v", err, errNoDatabase)
	}
}

func TestUpdateStoresTheResultAndDropsTheCache(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)
	guildID := testGuildID()
	store.put(guildID, `{"limit":3}`)

	// Prime the cache, so that a stale entry would answer the read after the write.
	if _, err := Get[settings](t.Context(), key, guildID); err != nil {
		t.Fatalf("the priming Get failed: %v", err)
	}

	err := Update(t.Context(), key, guildID, func(s *settings) error {
		s.Limit = 11
		return nil
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if _, ok := db.GetCacheEntry(cacheKey(guildID, key)); ok {
		t.Error("the cache entry survived the write")
	}

	got, err := Get[settings](t.Context(), key, guildID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Limit != 11 {
		t.Errorf("Limit = %d, want 11", got.Limit)
	}
}

func TestUpdateReadsPastTheCache(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)
	guildID := testGuildID()
	store.put(guildID, `{"limit":3}`)

	if _, err := Get[settings](t.Context(), key, guildID); err != nil {
		t.Fatalf("the priming Get failed: %v", err)
	}
	store.put(guildID, `{"limit":4,"roles":{"elsewhere":5}}`)

	err := Update(t.Context(), key, guildID, func(s *settings) error {
		s.Enabled = false
		return nil
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, err := Get[settings](t.Context(), key, guildID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Limit != 4 {
		t.Errorf("Limit = %d, want 4: a write must build on what is stored, not on the cached copy", got.Limit)
	}
	if got.Roles["elsewhere"] != 5 {
		t.Errorf("Roles = %v, want the change made after the cached read to survive", got.Roles)
	}
}

func TestUpdateLeavesTheDocumentAloneWhenMutateRefuses(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)
	guildID := testGuildID()
	refused := errors.New("refused")

	err := Update(t.Context(), key, guildID, func(s *settings) error {
		s.Limit = 11
		return refused
	})
	if !errors.Is(err, refused) {
		t.Fatalf("Update = %v, want %v", err, refused)
	}
	if _, writes, _ := store.counts(); writes != 0 {
		t.Errorf("the document was written %d times, want 0", writes)
	}
}

func TestUpdateRejectsAValueTheScopeForbids(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)
	guildID := testGuildID()

	err := Update(t.Context(), key, guildID, func(s *settings) error {
		s.Limit = -1
		return nil
	})
	if err == nil {
		t.Fatal("Update = nil error, want the scope's validation to reject the value")
	}
	if _, writes, _ := store.counts(); writes != 0 {
		t.Errorf("the document was written %d times, want 0", writes)
	}
}

func TestUpdateNeedsAMutateFunction(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)

	if err := Update[settings](t.Context(), key, testGuildID(), nil); err == nil {
		t.Fatal("Update = nil error, want an error for a missing mutate function")
	}
}

func TestUpdateStartsFromTheDefaultsWhenTheDocumentIsBroken(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)
	guildID := testGuildID()
	store.put(guildID, `{"limit":"not a number"}`)

	err := Update(t.Context(), key, guildID, func(s *settings) error {
		s.Limit = 4
		return nil
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, err := Get[settings](t.Context(), key, guildID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !got.Enabled || got.Limit != 4 {
		t.Errorf("Get = %+v, want the defaults the broken document was replaced with, plus the change", got)
	}
}

func TestDocumentShowsTheResolvedSettings(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)
	guildID := testGuildID()
	store.put(guildID, `{"limit":9}`)

	document, err := Document(t.Context(), key, guildID)
	if err != nil {
		t.Fatalf("Document failed: %v", err)
	}

	var got settings
	if err := json.Unmarshal(document, &got); err != nil {
		t.Fatalf("Document = %s, which is not a settings document: %v", document, err)
	}
	if !got.Enabled || got.Limit != 9 {
		t.Errorf("Document = %s, want the stored limit with the default enabled", document)
	}
}

func TestResetDropsTheStoredDocument(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)
	guildID := testGuildID()
	store.put(guildID, `{"limit":9}`)

	reset, err := Reset(t.Context(), key, guildID)
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}
	if !reset {
		t.Error("Reset = false, want true for a guild with a stored document")
	}

	got, err := Get[settings](t.Context(), key, guildID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !sameSettings(got, defaultSettings()) {
		t.Errorf("Get = %+v, want the registered defaults %+v", got, defaultSettings())
	}
}

func TestResetReportsNothingStored(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)

	reset, err := Reset(t.Context(), key, testGuildID())
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}
	if reset {
		t.Error("Reset = true, want false for a guild with no stored document")
	}
}

func TestResetRejectsAnUnregisteredScope(t *testing.T) {
	if _, err := Reset(t.Context(), "no-such-scope", testGuildID()); err == nil {
		t.Fatal("Reset = nil error, want an error for an unregistered scope")
	}
}

func TestAReadIsServedFromTheCacheUntilItIsWritten(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)
	guildID := testGuildID()

	if _, err := Get[settings](t.Context(), key, guildID); err != nil {
		t.Fatalf("the first Get failed: %v", err)
	}
	if _, err := Get[settings](t.Context(), key, guildID); err != nil {
		t.Fatalf("the second Get failed: %v", err)
	}

	reads, _, _ := store.counts()
	if reads != 1 {
		t.Errorf("the scope was read %d times, want 1: the second read should have come from the cache", reads)
	}
}

func TestConcurrentReadsDoNotShareTheRegisteredDefaults(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)
	guildID := testGuildID()
	store.put(guildID, `{"roles":{"given":2}}`)

	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := Get[settings](t.Context(), key, guildID)
			if err != nil {
				t.Errorf("Get failed: %v", err)
				return
			}
			if got.Roles["kept"] != 1 {
				t.Errorf("Roles = %v, want the registered default alongside the stored key", got.Roles)
				return
			}
			delete(got.Roles, "kept")
		}()
	}
	wg.Wait()

	// A guild with no stored document reads the registered defaults, so its map
	// still holding "kept" proves that no reader wrote into the shared default.
	fresh, err := Get[settings](t.Context(), key, testGuildID())
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if fresh.Roles["kept"] != 1 {
		t.Errorf("Roles = %v, want the registered default map intact after concurrent overlays", fresh.Roles)
	}
}

func TestCacheKeySeparatesGuildsAndScopes(t *testing.T) {
	if cacheKey("1", "ab") == cacheKey("1a", "b") {
		t.Errorf("cacheKey(1, ab) = %q collides with cacheKey(1a, b)", cacheKey("1", "ab"))
	}
}

func TestCacheTTLComesFromTheEnvironment(t *testing.T) {
	t.Setenv(cacheTTLEnv, "90m")

	if got := cacheTTL(); got != 90*time.Minute {
		t.Errorf("cacheTTL = %v, want 90m", got)
	}
}

func TestCacheTTLDefaultsWhenUnset(t *testing.T) {
	t.Setenv(cacheTTLEnv, "")

	if got := cacheTTL(); got != time.Duration(cacheTTLMinutes)*time.Minute {
		t.Errorf("cacheTTL = %v, want %v", got, time.Duration(cacheTTLMinutes)*time.Minute)
	}
}

func TestScopesAreOrderedAndDescribeThemselves(t *testing.T) {
	store := newFakeStorage()
	key := registerTestScope(t, store)
	if _, ok := ScopeByKey(key); !ok {
		t.Errorf("ScopeByKey(%q) = false, want the scope to be findable", key)
	}

	missing, ok := ScopeByKey("no-such-scope")
	if ok {
		t.Error("ScopeByKey(no-such-scope) = true, want false")
	}
	if missing.Key != "no-such-scope" {
		t.Errorf("ScopeByKey(no-such-scope).Key = %q, want the key that was asked for", missing.Key)
	}

	scopes := Scopes()
	if len(scopes) < 2 {
		t.Fatalf("Scopes returned %d entries, want at least the two registered by these tests", len(scopes))
	}
	for i := 1; i < len(scopes); i++ {
		previous, current := scopes[i-1], scopes[i]
		if current.Module < previous.Module ||
			(current.Module == previous.Module && current.Key < previous.Key) {
			t.Errorf("Scopes is not ordered: %q/%q comes before %q/%q",
				previous.Module, previous.Key, current.Module, current.Key)
		}
	}
}

// registerScopeLoading registers a scope whose storage always reports the given
// error on a read, which is how the framework sees a bot with no database handle.
func registerScopeLoading(loadErr error, t *testing.T) string {
	t.Helper()

	key := testScopeKey()
	store := newFakeStorage()
	writable := store.storage()
	err := register(ScopeDesc[settings]{
		Key:      key,
		Module:   testModule,
		Summary:  "Settings for the unavailable-database tests.",
		Defaults: defaultSettings(),
		Validate: validateSettings,
	}, storage{
		load: func(_ context.Context, _ string) (json.RawMessage, error) {
			return nil, loadErr
		},
		save:  writable.save,
		reset: writable.reset,
	})
	if err != nil {
		t.Fatalf("registering scope %q failed: %v", key, err)
	}
	return key
}
