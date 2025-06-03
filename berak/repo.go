package berak

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/thansetan/berak/model"
)

type repository interface {
	Add(context.Context) error
	AddWithDate(context.Context, time.Time) error
	DeleteLast(context.Context) error
	GetMonthlyByYear(context.Context, uint64, string) ([]model.AggData, error)
	GetDailyByMonthAndYear(context.Context, uint64, uint64, string) ([]model.AggData, error)
	GetLastDataTimestamp(context.Context, string) (time.Time, error)
	GetLongestDayWithoutPoop(context.Context) (model.LongestDayWithoutPoop, error)
	GetMostPoopInADay(context.Context, string) (model.MostPoopInADay, error)
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

func (r *repo) GetMonthlyByYear(ctx context.Context, year uint64, offset string) ([]model.AggData, error) {
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

	var data []model.AggData
	for rows.Next() {
		var monthlyData model.AggData
		err = rows.Scan(&monthlyData.Period, &monthlyData.Count)
		if err != nil {
			return nil, err
		}
		data = append(data, monthlyData)
	}

	return data, nil
}

func (r *repo) GetDailyByMonthAndYear(ctx context.Context, year, month uint64, offset string) ([]model.AggData, error) {
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

	var data []model.AggData
	for rows.Next() {
		var dailyData model.AggData
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

func (r *repo) GetLongestDayWithoutPoop(ctx context.Context) (model.LongestDayWithoutPoop, error) {
	var l model.LongestDayWithoutPoop
	err := r.db.QueryRowContext(ctx, `
		SELECT b0.timestamp curr_poop,
       		b1.timestamp prev_poop
		FROM berak b0
         	LEFT JOIN berak b1 ON b0.id - 1 = b1.id
		ORDER BY JULIANDAY(b0.timestamp) - JULIANDAY(b1.timestamp) DESC
		LIMIT 1
		`).Scan(&l.EndTime, &l.StartTime)
	if err != nil {
		return model.LongestDayWithoutPoop{}, err
	}

	return l, nil
}

func (r *repo) GetMostPoopInADay(ctx context.Context, offset string) (model.MostPoopInADay, error) {
	var m model.MostPoopInADay
	err := r.db.QueryRowContext(ctx, `
		WITH timestamp_with_offset AS (SELECT id,
		                                      DATETIME(timestamp, ?) timestamp
		                               FROM berak)
		SELECT STRFTIME('%m', timestamp) bulan,
		       STRFTIME('%d', timestamp) tanggal,
		       COUNT(id)                 jumlah
		FROM timestamp_with_offset
		GROUP BY bulan, tanggal
		ORDER BY jumlah DESC
		LIMIT 1`, offset).Scan(&m.Month, &m.Day, &m.Count)
	if err != nil {
		return model.MostPoopInADay{}, err
	}
	return m, nil
}
