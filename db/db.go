package db

import (
	"database/sql"

	_ "github.com/lib/pq"
)

func NewConn(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
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
	ID SERIAL PRIMARY KEY,
	timestamp TIMESTAMP NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON berak(timestamp);
	CREATE INDEX IF NOT EXISTS idx_year ON berak(EXTRACT(YEAR FROM timestamp));
	CREATE INDEX IF NOT EXISTS idx_month ON berak(EXTRACT(MONTH FROM timestamp));
	CREATE INDEX IF NOT EXISTS idx_day ON berak(EXTRACT(DAY FROM timestamp));
	`)
	if err != nil {
		return err
	}
	return nil
}
