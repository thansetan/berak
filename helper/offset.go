package helper

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Offset struct {
	f func(time.Time) time.Time
	s string
}

func (o Offset) Apply(t time.Time) time.Time {
	return o.f(t)
}

func (o Offset) String() string {
	return o.s
}

func NewOffset(s string) (Offset, error) {
	var offset Offset
	separated := strings.Split(strings.TrimSpace(s), " ")
	if len(separated) != 2 {
		return offset, fmt.Errorf("invalid offset format: %s", s)
	}
	n, err := strconv.Atoi(separated[0])
	if err != nil {
		return offset, fmt.Errorf("invalid time nominal: %s", separated[0])
	}
	switch strings.ToLower(separated[1]) {
	case "days", "day":
		offset.f = func(t time.Time) time.Time {
			return t.AddDate(0, 0, n)
		}
	case "hours", "hour":
		offset.f = func(t time.Time) time.Time {
			return t.Add(time.Duration(n) * time.Hour)
		}
	case "minutes", "minute":
		offset.f = func(t time.Time) time.Time {
			return t.Add(time.Duration(n) * time.Minute)
		}
	case "seconds", "second":
		offset.f = func(t time.Time) time.Time {
			return t.Add(time.Duration(n) * time.Second)
		}
	case "years":
		offset.f = func(t time.Time) time.Time {
			return t.AddDate(n, 0, 0)
		}
	}
	offset.s = s

	return offset, nil
}
