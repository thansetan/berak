package model

import (
	"fmt"
	"time"
)

type Data struct {
	TableData
	Statistics
	Year  int
	Month int
}

type TableData struct {
	CurrentTime time.Time
	Data        []AggData
	Year        int
}

type Statistics struct {
	LastPoopAt            time.Time
	LongestDayWithoutPoop LongestDayWithoutPoop
	MostPoopInADay        MostPoopInADay
}

type AggData struct {
	Period int
	Count  int
}

type LongestDayWithoutPoop struct {
	StartTime time.Time
	EndTime   time.Time
}

func (l LongestDayWithoutPoop) IsEmpty() bool {
	return (l.StartTime.IsZero() || l.EndTime.IsZero()) || (l.EndTime.Sub(l.StartTime) < time.Hour)
}

func (l LongestDayWithoutPoop) String() string {
	timeDiff := l.EndTime.Sub(l.StartTime)
	dayDiff := int(timeDiff.Hours()) / 24
	hourDiff := int(timeDiff.Hours()) - 24*dayDiff
	dayStr := "day"
	hourStr := "hour"
	if dayDiff != 1 {
		dayStr += "s"
	}
	if hourDiff != 1 {
		hourStr += "s"
	}
	return fmt.Sprintf("%d %s and %d %s", dayDiff, dayStr, hourDiff, hourStr)
}

type MostPoopInADay struct {
	Year  int
	Month int
	Day   int
	Count int
}

func (m MostPoopInADay) Path() string {
	return fmt.Sprintf("/%d/%d#%d", m.Year, m.Month, m.Day)
}

func (m MostPoopInADay) IsEmpty() bool {
	return m.Count <= 1
}
