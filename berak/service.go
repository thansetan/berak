package berak

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/thansetan/berak/helper"
	"github.com/thansetan/berak/model"
)

type berakService struct {
	repo   *berakRepository
	offset string
}

func NewService(repo *berakRepository, offset string) *berakService {
	return &berakService{repo, offset}
}

func (s *berakService) GetMonthly(ctx context.Context, now time.Time, year uint64) (model.TableData, error) {
	var data model.TableData
	monthlyData, err := s.repo.GetMonthlyByYear(ctx, year, s.offset)
	if err != nil {
		return data, fmt.Errorf("get monthly data: %w", err)
	}

	maxMonth := now.Month()
	if year < uint64(now.Year()) {
		maxMonth = 12
	}
	completeMonthlyData := make([]model.AggData, 0, maxMonth)

	curr := 1
	for _, d := range monthlyData {
		for ; curr < d.Period; curr++ {
			completeMonthlyData = append(completeMonthlyData, model.AggData{Period: curr})
		}
		completeMonthlyData = append(completeMonthlyData, d)
		curr++
	}
	for ; curr <= int(maxMonth); curr++ {
		completeMonthlyData = append(completeMonthlyData, model.AggData{Period: curr})
	}

	data.CurrentTime = now
	data.Year = int(year)
	data.Data = completeMonthlyData

	return data, nil
}

func (s *berakService) GetDaily(ctx context.Context, now time.Time, year uint64, month uint64) (model.TableData, error) {
	var data model.TableData
	dailyData, err := s.repo.GetDailyByMonthAndYear(ctx, year, month, s.offset)
	if err != nil {
		return data, fmt.Errorf("get daily data: %w", err)
	}

	maxDays := now.Day()
	if year < uint64(now.Year()) || (year == uint64(now.Year()) && month < uint64(now.Month())) {
		maxDays = helper.GetMonth(int(month)).Days
		if month == 2 && helper.IsLeapYear(int(year)) {
			maxDays++
		}
	}
	dailyDataComplete := make([]model.AggData, 0, maxDays)

	curr := 1
	for _, d := range dailyData {
		for ; curr < d.Period; curr++ {
			dailyDataComplete = append(dailyDataComplete, model.AggData{Period: curr})
		}
		dailyDataComplete = append(dailyDataComplete, d)
		curr++
	}
	for ; curr <= maxDays; curr++ {
		dailyDataComplete = append(dailyDataComplete, model.AggData{Period: curr})
	}

	data.CurrentTime = now
	data.Data = dailyDataComplete

	return data, nil
}

func (s *berakService) GetStatistics(ctx context.Context) (model.Statistics, error) {
	var data model.Statistics
	mostPoopInADay, err := s.repo.GetMostPoopInADay(ctx, s.offset)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return data, fmt.Errorf("get most poop in a day: %w", err)
	}

	longestDayWithoutPoop, err := s.repo.GetLongestDayWithoutPoop(ctx, s.offset)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return data, fmt.Errorf("get longest day without poop: %w", err)
	}
	lastPoopAt, err := s.GetLastPoopTime(ctx)
	if err != nil {
		return data, fmt.Errorf("get last poop time: %w", err)
	}

	data.LastPoopAt = lastPoopAt
	data.MostPoopInADay = mostPoopInADay
	data.LongestDayWithoutPoop = longestDayWithoutPoop

	return data, nil
}

func (s *berakService) GetLastPoopTime(ctx context.Context) (time.Time, error) {
	t, err := s.repo.GetLastDataTimestamp(ctx, s.offset)
	if err != nil {
		return time.Time{}, fmt.Errorf("get last poop timestamp: %w", err)
	}
	return t, nil
}

func (s *berakService) DeleteLast(ctx context.Context) error {
	err := s.repo.DeleteLast(ctx)
	if err != nil {
		return fmt.Errorf("delete last poop: %w", err)
	}
	return nil
}

func (s *berakService) Add(ctx context.Context, date time.Time) error {
	if date.IsZero() {
		err := s.repo.Add(ctx)
		if err != nil {
			return fmt.Errorf("add poop: %w", err)
		}
		return nil
	}
	err := s.repo.AddWithDate(ctx, date)
	if err != nil {
		return fmt.Errorf("add poop with time: %w", err)
	}
	return nil
}
