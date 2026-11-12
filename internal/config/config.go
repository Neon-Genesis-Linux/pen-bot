// Package config resolves the settings a guild has configured for the bot, and
// backs the /config command that edits them. Each configurable area registers a
// ScopeDesc before the bot starts, so it declares its own settings shape,
// defaults and validation while the framework owns storage, caching and the
// fallback.
//
// Reads fail closed: the bot opens its database after the gateway is already
// open and leaves no handle until then, so a read before it opens returns an
// error rather than defaults that may be stale. A guild that has configured
// nothing still reads the registered defaults; only a database that will not
// open is a failure.
package config

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/Neon-Genesis-Linux/pen-bot/internal/db"
)

const (
	cacheKeyPrefix = "cfg:v1"

	cacheTTLEnv     = "CONFIG_CACHE_TTL"
	cacheTTLMinutes = 5
)

// errNoDatabase reports that the bot holds no database handle yet, so a read or
// a write has nowhere to go.
var errNoDatabase = errors.New("database is not available")

// ErrScopeNotResettable reports that a scope has no single way back to its
// defaults, so it cannot be reset in one step. Its message is written for the
// caller, who may show it as it stands.
var ErrScopeNotResettable = errors.New("this scope cannot be reset in one step")

// ScopeDesc declares one configurable area's settings shape, the value a guild
// that has configured nothing sees, and an optional check run against every
// value read or written.
//
// The defaults are overlaid onto what is stored, so a key the stored document
// leaves out keeps its default, and a setting whose zero value carries meaning
// should be declared as a pointer. A scope whose settings include a map should
// default that map to empty, or the defaults' entries reappear in every guild.
type ScopeDesc[T any] struct {
	// Key names the scope in storage and in /config. It is the top-level slash
	// command path its module answers to, so one scope covers a command and all
	// of its subcommands.
	Key string
	// Module names the package or feature area the scope belongs to, which
	// /config view groups by.
	Module string
	// Summary is the one-line description /config view shows.
	Summary string
	// Defaults is what a guild that has never configured the scope reads back.
	Defaults T
	// Validate rejects a value that should not be stored. A nil Validate
	// accepts everything.
	Validate func(T) error
}

// ScopeInfo describes a registered scope without its settings.
type ScopeInfo struct {
	Key     string
	Module  string
	Summary string
}

// registeredScope holds a scope's settings type erased, so that Get and Update
// can find it by key and refuse a caller asking for the wrong type.
type registeredScope struct {
	key     string
	module  string
	summary string

	// typ is the settings type the scope was registered with. Get and Update
	// compare it against the type their caller asked for.
	typ reflect.Type

	// defaultsJSON is the registered defaults encoded, decoded afresh for each
	// read rather than copied from the registered value every reader shares.
	defaultsJSON []byte

	// load reads the stored document, reporting errNoDatabase when there is no
	// handle. save replaces it.
	load func(ctx context.Context, guildID string) (json.RawMessage, error)
	save func(ctx context.Context, guildID string, raw json.RawMessage) error

	// resolve overlays raw onto a private copy of the defaults into dst, which
	// must be a pointer to the scope's settings type.
	resolve func(raw json.RawMessage, dst any) error
	// render does the same and encodes the result, for a caller that does not
	// know the scope's settings type.
	render func(raw json.RawMessage) (json.RawMessage, error)
	// validate reports whether a settings value may be stored.
	validate func(value any) error
	// reset drops the stored document, reporting whether there was one.
	reset func(ctx context.Context, guildID string) (bool, error)
}

// storage tells a scope where its document lives. The guild-wide scope stores it
// in the guild's settings document; any other scope stores its own elsewhere,
// but load, save and reset share this shape, so nothing else about a scope
// depends on which store its document lives in.
type storage struct {
	load  func(ctx context.Context, guildID string) (json.RawMessage, error)
	save  func(ctx context.Context, guildID string, raw json.RawMessage) error
	reset func(ctx context.Context, guildID string) (bool, error)
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]*registeredScope)
)

// register adds a scope to the registry. It reports an error when the key is
// empty, the defaults cannot be encoded, or the key is already taken.
func register[T any](d ScopeDesc[T], store storage) error {
	if d.Key == "" {
		return fmt.Errorf("scope key required")
	}
	if store.load == nil || store.save == nil || store.reset == nil {
		return fmt.Errorf("scope %q needs storage for load, save and reset", d.Key)
	}

	defaultsJSON, err := json.Marshal(d.Defaults)
	if err != nil {
		return fmt.Errorf("scope %q defaults are not encodable: %w", d.Key, err)
	}

	scope := &registeredScope{
		key:          d.Key,
		module:       d.Module,
		summary:      d.Summary,
		typ:          reflect.TypeFor[T](),
		defaultsJSON: defaultsJSON,
		load:         store.load,
		save:         store.save,
		reset:        store.reset,
		resolve: func(raw json.RawMessage, dst any) error {
			value, ok := dst.(*T)
			if !ok {
				return fmt.Errorf("config scope %q expects %s", d.Key, reflect.TypeFor[T]())
			}
			overlaid, err := overlay[T](defaultsJSON, raw)
			*value = overlaid
			return err
		},
		render: func(raw json.RawMessage) (json.RawMessage, error) {
			overlaid, err := overlay[T](defaultsJSON, raw)
			if err != nil {
				return nil, err
			}
			return json.Marshal(overlaid)
		},
		validate: func(value any) error {
			if d.Validate == nil {
				return nil
			}
			typed, ok := value.(T)
			if !ok {
				return fmt.Errorf("config scope %q expects %s", d.Key, reflect.TypeFor[T]())
			}
			return d.Validate(typed)
		},
	}

	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[scope.key]; exists {
		return fmt.Errorf("scope %q is already registered", d.Key)
	}
	registry[scope.key] = scope
	return nil
}

// RegisterScope is the public door into the framework: it registers a scope so
// Get, Update, Document and Reset can serve it by key. The load, save and reset
// functions are the scope's storage — load reads its document for a guild, save
// replaces it and reset drops it, reporting whether there was one.
func RegisterScope[T any](
	desc ScopeDesc[T],
	load func(ctx context.Context, guildID string) (json.RawMessage, error),
	save func(ctx context.Context, guildID string, raw json.RawMessage) error,
	reset func(ctx context.Context, guildID string) (bool, error),
) error {
	return register(desc, storage{load: load, save: save, reset: reset})
}

// Get returns a guild's settings for a scope with the registered defaults
// overlaid, and reports an error only when the scope is unknown, its type does
// not match the caller's, or the database could not be read. A stored document
// that cannot be decoded or does not pass validation is logged and replaced by
// the defaults, so a bad row cannot break a command.
func Get[T any](ctx context.Context, key, guildID string) (T, error) {
	scope, err := scopeByType[T](key)
	if err != nil {
		var zero T
		return zero, err
	}

	raw, err := scope.read(ctx, guildID)
	if err != nil {
		var zero T
		return zero, err
	}

	value, ok := decode[T](scope, raw, guildID)
	if !ok {
		return defaultValue[T](scope), nil
	}
	return value, nil
}

// Update applies mutate to a guild's settings for a scope and stores the result.
// It reads past the cache, because writing a document built from a copy that
// predates the call would undo changes the caller never saw, and drops the
// cache entry once the write lands.
//
// mutate reports an error to abandon the write, which is how a caller validates
// something it could not express through the scope's Validate.
func Update[T any](ctx context.Context, key, guildID string, mutate func(*T) error) error {
	if mutate == nil {
		return fmt.Errorf("scope %q needs a mutate function", key)
	}

	scope, err := scopeByType[T](key)
	if err != nil {
		return err
	}

	raw, err := scope.readFresh(ctx, guildID)
	if err != nil {
		return err
	}

	value, ok := decode[T](scope, raw, guildID)
	if !ok {
		value = defaultValue[T](scope)
	}

	if err := mutate(&value); err != nil {
		return err
	}
	if err := scope.validate(value); err != nil {
		return err
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err := scope.save(ctx, guildID, encoded); err != nil {
		return err
	}

	db.DeleteCacheEntry(cacheKey(guildID, scope.key))
	return nil
}

// Document returns a scope's settings for a guild as JSON with the registered
// defaults overlaid. It is for a caller that shows the settings rather than acts
// on them, and so does not know the scope's settings type.
func Document(ctx context.Context, key, guildID string) (json.RawMessage, error) {
	scope, err := scopeByKey(key)
	if err != nil {
		return nil, err
	}

	raw, err := scope.read(ctx, guildID)
	if err != nil {
		return nil, err
	}
	document, err := scope.render(raw)
	if err != nil {
		slog.Warn("config unreadable, showing defaults",
			"scope", key, "guild_id", guildID, slog.Any("error", err))
		return scope.render(nil)
	}
	return document, nil
}

// Reset drops a guild's stored settings for a scope so that it reads back as the
// registered defaults, reporting whether anything was stored.
func Reset(ctx context.Context, key, guildID string) (bool, error) {
	scope, err := scopeByKey(key)
	if err != nil {
		return false, err
	}

	reset, err := scope.reset(ctx, guildID)
	if err != nil || !reset {
		return reset, err
	}

	db.DeleteCacheEntry(cacheKey(guildID, scope.key))
	return true, nil
}

// ScopeByKey returns a registered scope's metadata without its settings. The
// returned ScopeInfo always carries the key that was asked for, so a caller can
// name it in the message it shows when the lookup fails.
func ScopeByKey(key string) (ScopeInfo, bool) {
	scope, err := scopeByKey(key)
	if err != nil {
		return ScopeInfo{Key: key, Module: "", Summary: ""}, false
	}
	return ScopeInfo{
		Key:     scope.key,
		Module:  scope.module,
		Summary: scope.summary,
	}, true
}

// Scopes returns every registered scope ordered by module and then key, for
// display.
func Scopes() []ScopeInfo {
	registryMu.RLock()
	defer registryMu.RUnlock()

	infos := make([]ScopeInfo, 0, len(registry))
	for _, scope := range registry {
		infos = append(infos, ScopeInfo{
			Key:     scope.key,
			Module:  scope.module,
			Summary: scope.summary,
		})
	}
	slices.SortFunc(infos, func(a, b ScopeInfo) int {
		return cmp.Or(cmp.Compare(a.Module, b.Module), cmp.Compare(a.Key, b.Key))
	})
	return infos
}

// read returns a scope's stored document for a guild, preferring the cache. A
// nil document with a nil error means the guild has never configured it.
func (s *registeredScope) read(ctx context.Context, guildID string) (json.RawMessage, error) {
	key := cacheKey(guildID, s.key)
	if cached, ok := db.GetCacheEntry(key); ok {
		return json.RawMessage(cached), nil
	}

	raw, err := s.load(ctx, guildID)
	if err != nil {
		return nil, err
	}

	db.SetCacheEntry(key, raw, cacheTTL())
	return raw, nil
}

// readFresh reads past the cache, so a write is built on what is stored now
// rather than on a copy that may predate it.
func (s *registeredScope) readFresh(ctx context.Context, guildID string) (json.RawMessage, error) {
	return s.load(ctx, guildID)
}

// decode overlays a stored document onto the defaults, reporting false when the
// result is unusable and the caller should fall back to the defaults alone.
func decode[T any](s *registeredScope, raw json.RawMessage, guildID string) (T, bool) {
	var value T
	if err := s.resolve(raw, &value); err != nil {
		slog.Warn("config unreadable, falling back to defaults",
			"scope", s.key, "guild_id", guildID, slog.Any("error", err))
		var zero T
		return zero, false
	}
	if err := s.validate(value); err != nil {
		slog.Warn("config failed validation, falling back to defaults",
			"scope", s.key, "guild_id", guildID, slog.Any("error", err))
		var zero T
		return zero, false
	}
	return value, true
}

// defaultValue returns a private copy of a scope's registered defaults.
func defaultValue[T any](s *registeredScope) T {
	value, err := overlay[T](s.defaultsJSON, nil)
	if err != nil {
		var zero T
		return zero
	}
	return value
}

// overlay decodes a stored document onto a fresh copy of a scope's registered
// defaults. It returns the defaults alone, alongside an error, when the document
// cannot be decoded, and the zero value stands in for the settings type.
func overlay[T any](defaultsJSON []byte, raw json.RawMessage) (T, error) {
	value := new(T)
	if err := json.Unmarshal(defaultsJSON, value); err != nil {
		return *value, err
	}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, value); err != nil {
			return *value, err
		}
	}
	return *value, nil
}

// scopeByType finds a scope by key and reports an error when it holds a
// different settings type than the caller asked for, which would otherwise
// decode into a zero value and look like an unconfigured guild.
func scopeByType[T any](key string) (*registeredScope, error) {
	scope, err := scopeByKey(key)
	if err != nil {
		return nil, err
	}
	if want := reflect.TypeFor[T](); scope.typ != want {
		return nil, fmt.Errorf("config scope %q holds %s, not %s", key, scope.typ, want)
	}
	return scope, nil
}

func scopeByKey(key string) (*registeredScope, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	scope, ok := registry[key]
	if !ok {
		return nil, fmt.Errorf("config scope %q is not registered", key)
	}
	return scope, nil
}

// cacheKey namespaces a scope's cache entry. A Discord snowflake is digits and a
// scope key is a slash command name, so neither contains the separator and no
// two guild and scope pairs can collide.
func cacheKey(guildID, scopeKey string) string {
	return cacheKeyPrefix + ":" + guildID + ":" + scopeKey
}

// cacheTTL is how long a stored document may be reused before a read goes to
// the database again. Any write drops the entry straight away.
func cacheTTL() time.Duration {
	return db.ParseCleanupIntervalEnv(cacheTTLEnv, cacheTTLMinutes)
}
