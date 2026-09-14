package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(path string) (*sql.DB, error) {
	if path == "" {
		path = "./messageboard.db"
	}

	dsn := path + "?" + strings.Join([]string{
		"_foreign_keys=on",
		"_busy_timeout=5000",
		"_journal_mode=WAL",
		"_synchronous=NORMAL",
	}, "&")

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	if err := ApplyMigrations(db, "./Database/migrations"); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}

	return db, nil
}
