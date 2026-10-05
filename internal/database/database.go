package database

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func NewConnection() (*sql.DB, error) {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(getenv("DB_USER", "milessn"), getenv("DB_PASSWORD", "")),
		Host:     fmt.Sprintf("%s:%s", getenv("DB_HOST", "localhost"), getenv("DB_PORT", "5432")),
		Path:     getenv("DB_NAME", "rest_api_db"),
		RawQuery: "sslmode=disable",
	}
	db, err := sql.Open("pgx", u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize db handle: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}
	return db, nil
}
