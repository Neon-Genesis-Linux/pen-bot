package db

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

type Config struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func New(cfg Config) (*DB, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("database DSN is required")
	}

	db, err := sql.Open("postgres", cfg.DSN)
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

func NewFromEnv() (*DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := getEnv("DB_HOST", "localhost")
		port := getEnv("DB_PORT", "5432")
		user := getEnv("DB_USER", "pen_bot")
		password := os.Getenv("DB_PASSWORD")
		dbName := os.Getenv("DB_NAME")
		if dbName == "" {
			botInstance := os.Getenv("DB_BOT_INSTANCE_ID")
			if botInstance == "" {
				dbName = "pen_bot"
			} else {
				dbName = fmt.Sprintf("pen_bot_%s", botInstance)
			}
		}

		u := &url.URL{
			Scheme: "postgres",
			User:   url.UserPassword(user, password),
			Host:   net.JoinHostPort(host, port),
			Path:   dbName,
		}
		u.RawQuery = "sslmode=disable"
		dsn = u.String()
	}

	cfg := Config{
		DSN:             dsn,
		MaxOpenConns:    parseIntEnv("DB_MAX_OPEN_CONNS", 16),
		MaxIdleConns:    parseIntEnv("DB_MAX_IDLE_CONNS", 4),
		ConnMaxLifetime: parseDurationEnv("DB_CONN_MAX_LIFETIME", 30*time.Minute),
	}

	return New(cfg)
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

func SetGlobalDB(d *DB) {
	globalDBMu.Lock()
	defer globalDBMu.Unlock()
	globalDB = d
}

func GlobalDB() *DB {
	globalDBMu.RLock()
	defer globalDBMu.RUnlock()
	return globalDB
}

func CloseDB() {
	globalDBMu.Lock()
	defer globalDBMu.Unlock()
	if globalDB != nil {
		_ = globalDB.Close()
		globalDB = nil
	}
}

func ParseCleanupIntervalEnv(key string, defaultMinutes int) time.Duration {
	return parseDurationEnv(key, time.Duration(defaultMinutes)*time.Minute)
}
