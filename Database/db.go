package Database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./messageboard.db")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	schemaBytes, err := os.ReadFile("database/schema.sql")
	if err != nil {
		return nil, fmt.Errorf("read schema: %w", err)
	}

	if _, err := db.Exec(string(schemaBytes)); err != nil {
		return nil, fmt.Errorf("create tables: %w", err)
	}

	return db, nil
}
