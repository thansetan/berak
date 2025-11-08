package helper

type MonthData struct {
	Name string
	Days int
}

var months = []MonthData{
	{Name: "January", Days: 31},
	{Name: "February", Days: 28},
	{Name: "March", Days: 31},
	{Name: "April", Days: 30},
	{Name: "May", Days: 31},
	{Name: "June", Days: 30},
	{Name: "July", Days: 31},
	{Name: "August", Days: 31},
	{Name: "September", Days: 30},
	{Name: "October", Days: 31},
	{Name: "November", Days: 30},
	{Name: "December", Days: 31},
}

func GetMonth(monthNumber int) MonthData {
	if monthNumber < 1 || monthNumber > 12 {
		return MonthData{}
	}
	return months[monthNumber-1]
}

func IsLeapYear(year int) bool {
	if year < 0 {
		return false
	}
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}
