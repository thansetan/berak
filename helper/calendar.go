package helper

var MonthDays = map[int]int{
	1:  31,
	2:  28,
	3:  31,
	4:  30,
	5:  31,
	6:  30,
	7:  31,
	8:  31,
	9:  30,
	10: 31,
	11: 30,
	12: 31,
}

func IsLeapYear(year int) bool {
	if year < 0 {
		return false
	}
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}
