package model

import (
	"fmt"
	"time"
)

type AggData struct {
	Period int
	Count  int
}

type LongestDayWithoutPoop struct {
	StartTime time.Time
	EndTime   time.Time
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
	Month int
	Day   int
	Count int
}
