package berak

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	DELETE FROM berak WHERE id = (SELECT MAX(id) FROM berak)`)
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
	strftime('%m', timestamp) AS month,
	COUNT(id) AS berak_count
	FROM berak
	WHERE strftime('%Y', timestamp) = ?
	GROUP BY month
	ORDER BY month ASC;`, fmt.Sprintf("%04d", year))
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
	strftime('%d', timestamp) day,
	COUNT(id) berak_count
	FROM berak
	WHERE strftime('%Y', timestamp) = ? AND strftime('%m', timestamp) = ?
	GROUP BY day
	ORDER BY day ASC;`, fmt.Sprintf("%04d", year), fmt.Sprintf("%02d", month))
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
