package main

import (
	"database/sql"

	"github.com/jackc/pgx/v4"
	pgxstdlib "github.com/jackc/pgx/v4/stdlib"
)

var db *sql.DB

// Try to open a connection and ping the database
func openDBPool() error {
	dbConnConfig, err := pgx.ParseConfig(
		"postgresql://postgres@localhost/pedro_bank")

	if err == nil {
		db = pgxstdlib.OpenDB(*dbConnConfig)
		err = db.Ping()
	}

	return err
}
