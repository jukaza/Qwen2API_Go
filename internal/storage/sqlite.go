package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"qwen2api/internal/config"
)

func isSQLiteMode(cfg config.Config) bool {
	return strings.EqualFold(strings.TrimSpace(cfg.DataSaveMode), "sqlite")
}

func newSQLiteDB(cfg config.Config) (*sql.DB, error) {
	dbPath := cfg.SQLiteDBPath
	if dbPath == "" {
		dbPath = "data/data.db"
	}

	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sqlite data dir: %w", err)
	}

	// modernc.org/sqlite driver name is "sqlite"
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	// PRAGMA for performance and concurrency
	if _, err := db.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA busy_timeout = 5000;
		PRAGMA foreign_keys = ON;
	`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to set sqlite PRAGMA: %w", err)
	}

	if err := autoMigrateSQLite(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to auto migrate sqlite tables: %w", err)
	}

	return db, nil
}

func autoMigrateSQLite(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS accounts (
			email TEXT PRIMARY KEY,
			password TEXT,
			token TEXT,
			source TEXT,
			expires INTEGER,
			proxy_id TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS proxies (
			id TEXT PRIMARY KEY,
			name TEXT,
			proxy_url TEXT,
			no_proxy TEXT,
			type TEXT,
			is_active BOOLEAN,
			strict_proxy BOOLEAN,
			test_status TEXT,
			last_error TEXT,
			last_tested_at INTEGER,
			created_at INTEGER,
			bound_connection_count INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS chat_sessions (
			context_hash TEXT PRIMARY KEY,
			account_email TEXT,
			chat_id TEXT,
			model TEXT,
			chat_type TEXT,
			updated_at INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			ip TEXT,
			user_agent TEXT,
			created_at INTEGER,
			expires_at INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS chat_usages (
			account_email TEXT,
			chat_id TEXT,
			updated_at INTEGER,
			PRIMARY KEY (account_email, chat_id)
		);`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			key TEXT PRIMARY KEY,
			label TEXT,
			is_admin BOOLEAN DEFAULT 0,
			created_at INTEGER
		);`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}
