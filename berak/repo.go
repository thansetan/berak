package berak

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type repository interface {
	Add(context.Context) error
	DeleteLast(context.Context) error
	GetMonthlyByYear(context.Context, uint64) ([]AggData, error)
	GetDailyByMonthAndYear(context.Context, uint64, uint64) ([]AggData, error)
	GetLastDataTimestamp(context.Context) (time.Time, error)
}

type repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *repo {
	return &repo{db}
}

func (r *repo) Add(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
	INSERT INTO berak DEFAULT VALUES`)
	if err != nil {
		return err
	}
	return nil
}

func (r *repo) DeleteLast(ctx context.Context) error {
	res, err := r.db.ExecContext(ctx, `
	DELETE FROM berak WHERE id = (SELECT id FROM berak ORDER BY timestamp DESC LIMIT 1)`)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return errors.New("no data")
	}
	return nil
}

type AggData struct {
	Period int
	Count  int
}

func (r *repo) GetMonthlyByYear(ctx context.Context, year uint64) ([]AggData, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT
	EXTRACT(MONTH FROM timestamp) AS month,
	COUNT(id) berak_count
	FROM berak
	WHERE EXTRACT(YEAR FROM timestamp) = $1
	GROUP BY EXTRACT(MONTH FROM timestamp)
	ORDER BY EXTRACT(MONTH FROM timestamp) ASC;`, year)
	if err != nil {
		return nil, err
	}

	var data []AggData
	for rows.Next() {
		var monthlyData AggData
		err = rows.Scan(&monthlyData.Period, &monthlyData.Count)
		if err != nil {
			return nil, err
		}
		data = append(data, monthlyData)
	}

	return data, nil
}

func (r *repo) GetDailyByMonthAndYear(ctx context.Context, year, month uint64) ([]AggData, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT
	EXTRACT (DAY FROM timestamp),
	COUNT(id) berak_count
	FROM berak
	WHERE EXTRACT(YEAR FROM timestamp) = $1 AND EXTRACT(MONTH FROM timestamp) = $2
	GROUP BY EXTRACT (DAY FROM timestamp)
	ORDER BY EXTRACT (DAY FROM timestamp) ASC;`, year, month)
	if err != nil {
		return nil, err
	}

	var data []AggData
	for rows.Next() {
		var dailyData AggData
		err = rows.Scan(&dailyData.Period, &dailyData.Count)
		if err != nil {
			return nil, err
		}
		data = append(data, dailyData)
	}

	return data, nil
}

func (r *repo) GetLastDataTimestamp(ctx context.Context) (time.Time, error) {
	var lastDataAt time.Time
	err := r.db.QueryRowContext(ctx, `
	SELECT timestamp FROM berak ORDER BY timestamp DESC LIMIT 1;`).Scan(&lastDataAt)
	if err != nil {
		return time.Time{}, err
	}
	return lastDataAt, nil
}
