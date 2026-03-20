package database

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init(dbPath string) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	var err error
	DB, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	DB.SetMaxOpenConns(1)
	DB.SetMaxIdleConns(1)
	DB.SetConnMaxLifetime(time.Hour)

	createTables()
}

func createTables() {
	schema := `
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		root_domain TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		subdomain_count INTEGER DEFAULT 0,
		alive_count INTEGER DEFAULT 0,
		xss_count INTEGER DEFAULT 0,
		current_step TEXT DEFAULT '',
		error_message TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		finished_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS subdomains (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT NOT NULL,
		domain TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS alive_urls (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT NOT NULL,
		url TEXT NOT NULL,
		status_code INTEGER DEFAULT 0,
		title TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS xss_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT NOT NULL,
		url TEXT NOT NULL,
		payload TEXT DEFAULT '',
		param TEXT DEFAULT '',
		position TEXT DEFAULT '',
		report_content TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
	);
	`

	_, err := DB.Exec(schema)
	if err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
