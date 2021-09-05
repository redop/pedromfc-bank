package server

import (
	"database/sql"

	"github.com/jackc/pgx/v4"
	pgxstdlib "github.com/jackc/pgx/v4/stdlib"
)

var DB *sql.DB

// We want all of our transactions to run in Repeatable Read.
var defaultTxOptions = sql.TxOptions{Isolation: sql.LevelRepeatableRead,
	ReadOnly: false}

// Try to open a connection and ping the database
func OpenDBPool() error {
	dbConnConfig, err := pgx.ParseConfig(
		"postgresql://postgres@localhost/pedro_bank")

	if err == nil {
		DB = pgxstdlib.OpenDB(*dbConnConfig)
		err = DB.Ping()
	}

	return err
}

// Roll back tx and log an error, if any.
func rollbackTx(tx *sql.Tx) error {
	err := tx.Rollback()

	if err != nil {
		logger.Printf("Error rolling back tx: %v", err)
	}

	return err
}
