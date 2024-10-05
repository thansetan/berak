package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func NewConn(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	err = initDB(db)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func initDB(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS berak (
	ID INTEGER PRIMARY KEY,
	timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON berak(timestamp);
	CREATE INDEX IF NOT EXISTS idx_ymd ON berak(
		strftime('%Y', timestamp),
    	strftime('%m', timestamp),
    	strftime('%d', timestamp)
	);
	`)
	if err != nil {
		return err
	}
	return nil
}
