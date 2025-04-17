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
	AddWithDate(context.Context, time.Time) error
	DeleteLast(context.Context) error
	GetMonthlyByYear(context.Context, uint64, string) ([]AggData, error)
	GetDailyByMonthAndYear(context.Context, uint64, uint64, string) ([]AggData, error)
	GetLastDataTimestamp(context.Context, string) (time.Time, error)
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

func (r *repo) AddWithDate(ctx context.Context, t time.Time) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO berak(timestamp) VALUES(?)", t)
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

func (r *repo) GetMonthlyByYear(ctx context.Context, year uint64, offset string) ([]AggData, error) {
	rows, err := r.db.QueryContext(ctx, `
	WITH timestamp_with_offset AS (
		SELECT
			id,
			DATETIME(timestamp, ?) timestamp
		FROM berak
	)
	SELECT
		strftime('%m', timestamp) month,
		COUNT(id)
	FROM timestamp_with_offset
	WHERE strftime('%Y', timestamp) = ?
	GROUP BY month
	ORDER BY month;`, offset, fmt.Sprintf("%04d", year))
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

func (r *repo) GetDailyByMonthAndYear(ctx context.Context, year, month uint64, offset string) ([]AggData, error) {
	rows, err := r.db.QueryContext(ctx, `
	WITH timestamp_with_offset AS (
		SELECT
			id,
			DATETIME(timestamp, ?) timestamp
		FROM berak
	)
	SELECT
		strftime('%d', timestamp) day,
		COUNT(id)
	FROM timestamp_with_offset
	WHERE strftime('%Y', timestamp) = ? AND strftime('%m', timestamp) = ?
	GROUP BY day
	ORDER BY day;`, offset, fmt.Sprintf("%04d", year), fmt.Sprintf("%02d", month))
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

func (r *repo) GetLastDataTimestamp(ctx context.Context, offset string) (time.Time, error) {
	var s string
	err := r.db.QueryRowContext(ctx, `
	WITH timestamp_with_offset AS (
		SELECT
			id,
			DATETIME(timestamp, ?) timestamp
		FROM berak
	)
	SELECT timestamp FROM timestamp_with_offset ORDER BY timestamp DESC LIMIT 1;`, offset).Scan(&s)
	if err != nil {
		return time.Time{}, err
	}
	lastInsertAt, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return time.Time{}, nil
	}
	return lastInsertAt, nil
}
