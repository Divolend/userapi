package database

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewConnection() (*sql.DB, error) {
	connStr := fmt.Sprintf("postgres://%s@%s:%d/%s?sslmode=disable",
		"milessn", "localhost", 5432, "rest_api_db")
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize db handle: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}
	return db, nil
}
