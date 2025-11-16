package model

import (
	"fmt"
	"strings"
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
	LongestPoopStreak     LongestPoopStreak
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
	return (l.StartTime.IsZero() || l.EndTime.IsZero()) || (l.EndTime.Sub(l.StartTime) < time.Minute)
}

func (l LongestDayWithoutPoop) String() string {
	var sb strings.Builder
	timeDiff := l.EndTime.Sub(l.StartTime)
	dayDiff := int(timeDiff.Hours()) / 24
	hourDiff := int(timeDiff.Hours()) - 24*dayDiff
	minuteDiff := int(timeDiff.Minutes()) - 24*dayDiff*60 - 60*hourDiff
	if dayDiff > 0 {
		fmt.Fprintf(&sb, "%d day", dayDiff)
		if dayDiff > 1 {
			sb.WriteByte('s')
		}
	}
	if hourDiff > 0 {
		if sb.Len() != 0 {
			if minuteDiff == 0 {
				sb.WriteString(" and ")
			} else {
				sb.WriteString(", ")
			}
		}
		fmt.Fprintf(&sb, "%d hour", hourDiff)
		if hourDiff > 1 {
			sb.WriteByte('s')
		}
	}
	if minuteDiff > 0 {
		if sb.Len() != 0 {
			sb.WriteString(" and ")
		}
		fmt.Fprintf(&sb, "%d minute", minuteDiff)
		if minuteDiff > 1 {
			sb.WriteByte('s')
		}
	}

	return sb.String()
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
	return m.Count < 1
}

type LongestPoopStreak struct {
	StartDate, EndDate  time.Time
	DayCount, PoopCount int
}

func (l LongestPoopStreak) IsEmpty() bool {
	return l.DayCount < 2
}
