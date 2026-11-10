package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// driverName is the name modernc.org/sqlite registers itself under.
const driverName = "sqlite"

// defaultDir is the directory the database file lives in when DB_PATH is not
// set. It is relative to the working directory, so a local run keeps its data
// beside the binary rather than somewhere under the system root.
const defaultDir = "data"

// defaultInstanceID is the fallback for DB_BOT_INSTANCE_ID when it is unset.
const defaultInstanceID = "default"

// sqliteParams are the connection parameters that shape how SQLite behaves.
//
// The pragma parameters are applied to every connection the pool opens.
// busy_timeout makes a writer wait for a held lock instead of failing
// immediately, which is what keeps two writers arriving together from
// erroring with SQLITE_BUSY. It is set below the three seconds Discord allows
// an interaction to take, so a contended write still has time to report an
// error to the user. journal_mode and synchronous select the write-ahead log
// and its durability level, and foreign_keys turns on constraint enforcement.
//
// The time parameters make the driver store a time.Time as Unix milliseconds
// in an INTEGER column and read it back as a time.Time. SQLite has no
// timestamp type, and storing timestamps as text sorts them wrongly: Go trims
// trailing zeros from a fractional second, so two representations of the same
// instant can compare as different values. Integers compare exactly.
//
// The driver picks the unit when reading an integer back by its magnitude
// rather than by the declared unit, so a timestamp before September 2001
// would be read as seconds and would come back as a wrong time rather than
// as an error. Every timestamp here is written by the running bot and so is
// later than that.
//
// _txlock=immediate takes the write lock when a transaction starts rather
// than when it first writes. A deferred transaction that reads before it
// writes cannot always be upgraded to a writer later, and fails with
// SQLITE_BUSY in a way the busy timeout does not retry.
const sqliteParams = "_pragma=busy_timeout(2000)" +
	"&_pragma=journal_mode(WAL)" +
	"&_pragma=synchronous(NORMAL)" +
	"&_pragma=foreign_keys(1)" +
	"&_time_integer_format=unix_milli" +
	"&_inttotime=1" +
	"&_timezone=UTC" +
	"&_txlock=immediate"

// DB is a handle on the bot's SQLite database.
type DB struct {
	*sql.DB
}

// Config describes how to open the database.
type Config struct {
	// Path is the location of the database file.
	Path string
	// MaxOpenConns bounds the size of the connection pool.
	MaxOpenConns int
	// MaxIdleConns bounds the number of pooled connections kept for reuse.
	MaxIdleConns int
	// ConnMaxLifetime retires a connection after it has been open this long.
	ConnMaxLifetime time.Duration
}

// New opens the database at the configured path, creating the file and its
// parent directory if they do not exist.
//
// The path is resolved against the working directory, so a relative path
// keeps referring to the same file for the lifetime of the process.
func New(cfg Config) (*DB, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("database path is required")
	}

	dsn, err := dsn(cfg.Path)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &DB{db}, nil
}

// NewFromEnv opens the database described by the environment.
//
// DB_PATH selects the file. DB_BOT_INSTANCE_ID separates the instances that
// share one directory: each instance gets its own file, so two of them can
// run side by side without seeing each other's rows. A DB_PATH set in the
// environment wins over the instance id, which is how an operator points a
// single instance at a chosen file.
func NewFromEnv() (*DB, error) {
	path := os.Getenv("DB_PATH")
	if path == "" {
		path = filepath.Join(defaultDir, instanceFileName(getEnv("DB_BOT_INSTANCE_ID", defaultInstanceID)))
	}

	cfg := Config{
		Path:            path,
		MaxOpenConns:    parseIntEnv("DB_MAX_OPEN_CONNS", 4),
		MaxIdleConns:    parseIntEnv("DB_MAX_IDLE_CONNS", 4),
		ConnMaxLifetime: parseDurationEnv("DB_CONN_MAX_LIFETIME", 30*time.Minute),
	}

	return New(cfg)
}

// instanceFileName is the database file name for an instance id.
//
// The id is filtered down to characters that are safe in a file name, so a
// value taken from the environment cannot escape the directory the instances
// share or name a file outside it.
func instanceFileName(instanceID string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-', r == '_':
			return r
		default:
			return -1
		}
	}, instanceID)

	if safe == "" {
		safe = defaultInstanceID
	}

	return fmt.Sprintf("pen-bot-%s.db", safe)
}

// dsn builds the connection string for a database file.
//
// The path is made absolute so the connection refers to one file no matter
// what the working directory is, and the parent directory is created because
// SQLite will not create it itself.
func dsn(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve database path %q: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return "", fmt.Errorf("create database directory %q: %w", filepath.Dir(abs), err)
	}

	u := url.URL{
		Scheme:   "file",
		Path:     abs,
		RawQuery: sqliteParams,
	}

	return u.String(), nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func parseIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

var (
	globalDB   *DB
	globalDBMu sync.RWMutex
)

// SetGlobalDB installs the process-wide database handle.
func SetGlobalDB(d *DB) {
	globalDBMu.Lock()
	defer globalDBMu.Unlock()
	globalDB = d
}

// GlobalDB returns the process-wide database handle, or nil when no database
// has been opened yet.
func GlobalDB() *DB {
	globalDBMu.RLock()
	defer globalDBMu.RUnlock()
	return globalDB
}

// CloseDB closes the process-wide database handle, if one is open.
func CloseDB() {
	globalDBMu.Lock()
	defer globalDBMu.Unlock()
	if globalDB != nil {
		_ = globalDB.Close()
		globalDB = nil
	}
}

// ParseCleanupIntervalEnv returns the interval between cache cleanups.
func ParseCleanupIntervalEnv(key string, defaultMinutes int) time.Duration {
	return parseDurationEnv(key, time.Duration(defaultMinutes)*time.Minute)
}
