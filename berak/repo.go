package berak

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/thansetan/berak/model"
)

type berakRepository struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *berakRepository {
	return &berakRepository{db}
}

func (r *berakRepository) Add(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
	INSERT INTO berak DEFAULT VALUES`)
	if err != nil {
		return err
	}
	return nil
}

func (r *berakRepository) AddWithDate(ctx context.Context, t time.Time) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO berak(timestamp) VALUES(?)", t)
	if err != nil {
		return err
	}
	return nil
}

func (r *berakRepository) DeleteLast(ctx context.Context) error {
	res, err := r.db.ExecContext(ctx, `
	DELETE FROM berak WHERE id = (SELECT MAX(id) FROM berak)`)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return errors.New("no data")
	}
	return nil
}

func (r *berakRepository) GetMonthlyByYear(ctx context.Context, year uint64, offset string) ([]model.AggData, error) {
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

func (r *berakRepository) GetDailyByMonthAndYear(ctx context.Context, year, month uint64, offset string) ([]model.AggData, error) {
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

func (r *berakRepository) GetLastDataTimestamp(ctx context.Context, offset string) (time.Time, error) {
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
		return time.Time{}, fmt.Errorf("error parsing lastInsertedAt: %w", err)
	}
	return lastInsertAt, nil
}

func (r *berakRepository) GetLongestDayWithoutPoop(ctx context.Context, offset string) (model.LongestDayWithoutPoop, error) {
	var startTime, endTime sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT
    		DATETIME(timestamp, '7 hours') timestamp,
    		LAG(DATETIME(timestamp, '7 hours')) OVER (ORDER BY timestamp) prev_timestamp
		FROM berak
		ORDER BY JULIANDAY(timestamp) - JULIANDAY(prev_timestamp) DESC LIMIT 1
		`, offset, offset).Scan(&endTime, &startTime)
	if err != nil {
		return model.LongestDayWithoutPoop{}, err
	}

	var l model.LongestDayWithoutPoop

	if startTime.Valid {
		l.StartTime, err = time.Parse("2006-01-02 15:04:05", startTime.String)
		if err != nil {
			return model.LongestDayWithoutPoop{}, fmt.Errorf("error parsing startTime: %s", err)
		}
	}
	if endTime.Valid {
		l.EndTime, err = time.Parse("2006-01-02 15:04:05", endTime.String)
		if err != nil {
			return model.LongestDayWithoutPoop{}, fmt.Errorf("error parsing endTime: %s", err)
		}
	}

	return l, nil
}

func (r *berakRepository) GetMostPoopInADay(ctx context.Context, offset string) (model.MostPoopInADay, error) {
	var m model.MostPoopInADay
	err := r.db.QueryRowContext(ctx, `
		WITH timestamp_with_offset AS (SELECT id,
		                                      DATETIME(timestamp, ?) timestamp
		                               FROM berak)
		SELECT 
				STRFTIME('%Y', timestamp) tahun,
			   	STRFTIME('%m', timestamp) bulan,
		       	STRFTIME('%d', timestamp) tanggal,
		       	COUNT(id)                 jumlah
		FROM timestamp_with_offset
		GROUP BY tahun, bulan, tanggal
		ORDER BY jumlah DESC, tahun DESC, bulan DESC, tanggal DESC
		LIMIT 1`, offset).Scan(&m.Year, &m.Month, &m.Day, &m.Count)
	if err != nil {
		return model.MostPoopInADay{}, err
	}
	return m, nil
}
