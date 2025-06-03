package berak

import (
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/thansetan/berak/model"
)

type response struct {
	Message string `json:"message"`
}

func WriteResponseJSON(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response{
		Message: message,
	})
}

type controller struct {
	repo   repository
	tmpl   *template.Template
	logger *slog.Logger
}

func NewController(repo repository, tmpl *template.Template, logger *slog.Logger) *controller {
	return &controller{repo, tmpl, logger}
}

func (c *controller) Create(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Timestamp time.Time `json:"timestamp"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil && !errors.Is(err, io.EOF) {
		c.logger.ErrorContext(r.Context(), "error adding new 💩", "error", err.Error(), "remote_addr", r.RemoteAddr)
		WriteResponseJSON(w, http.StatusInternalServerError, "it's our fault, not yours!")
		return
	}
	defer r.Body.Close()
	if errors.Is(err, io.EOF) {
		err = c.repo.Add(r.Context())
	} else {
		if data.Timestamp.IsZero() {
			WriteResponseJSON(w, http.StatusBadRequest, "timestamp can't be empty!")
		}
		err = c.repo.AddWithDate(r.Context(), data.Timestamp.UTC())

	}
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error adding new 💩", "error", err.Error(), "remote_addr", r.RemoteAddr)
		WriteResponseJSON(w, http.StatusInternalServerError, "it's our fault, not yours!")
		return
	}
	c.logger.InfoContext(r.Context(), "new 💩 added!", "remote_addr", r.RemoteAddr)
	w.WriteHeader(http.StatusNoContent)
}

func (c *controller) Delete(w http.ResponseWriter, r *http.Request) {
	err := c.repo.DeleteLast(r.Context())
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error removing last 💩", "error", err.Error(), "remote_addr", r.RemoteAddr)
		WriteResponseJSON(w, http.StatusInternalServerError, "it's our fault, not yours!")
		return
	}
	c.logger.InfoContext(r.Context(), "last 💩 removed!", "remote_addr", r.RemoteAddr)
	w.WriteHeader(http.StatusNoContent)
}

func (c *controller) GetMonthly(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	yearStr := vars["year"]
	year, err := strconv.ParseUint(yearStr, 10, 64)
	if err != nil {
		c.FourOFour(w, r)
		return
	}

	// UTC + 7
	now := time.Now().UTC().Add(7 * time.Hour)
	if year < 1 || year > uint64(now.Year()) {
		c.FourOFour(w, r)
		return
	}
	monthlyData, err := c.repo.GetMonthlyByYear(r.Context(), year, os.Getenv("TIME_OFFSET"))
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error getting monthly 💩", "error", err.Error(), "remote_addr", r.RemoteAddr)
		WriteResponseJSON(w, http.StatusInternalServerError, "it's our fault, not yours!")
		return
	}

	mostPoopInADay, err := c.repo.GetMostPoopInADay(r.Context(), os.Getenv("TIME_OFFSET"))
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error getting most 💩 in a day", "error", err.Error(), "remote_addr", r.RemoteAddr)
		WriteResponseJSON(w, http.StatusInternalServerError, "it's our fault, not yours!")
		return
	}

	longestDayWithoutPoop, err := c.repo.GetLongestDayWithoutPoop(r.Context(), os.Getenv("TIME_OFFSET"))
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error getting longest day without 💩", "error", err.Error(), "remote_addr", r.RemoteAddr)
		WriteResponseJSON(w, http.StatusInternalServerError, "it's our fault, not yours!")
		return
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
	lastDataAt, err := c.repo.GetLastDataTimestamp(r.Context(), os.Getenv("TIME_OFFSET"))
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error getting last 💩", "error", err.Error(), "remote_addr", r.RemoteAddr)
		WriteResponseJSON(w, http.StatusInternalServerError, "it's our fault, not yours!")
		return
	}
	w.WriteHeader(http.StatusOK)
	err = c.tmpl.ExecuteTemplate(w, "year", struct {
		LastDataAt            time.Time
		Data                  []model.AggData
		Year                  int
		CurrentTime           time.Time
		LongestDayWithoutPoop model.LongestDayWithoutPoop
		MostPoopInADay        model.MostPoopInADay
	}{
		Year:                  int(year),
		Data:                  completeMonthlyData,
		LastDataAt:            lastDataAt,
		CurrentTime:           now,
		LongestDayWithoutPoop: longestDayWithoutPoop,
		MostPoopInADay:        mostPoopInADay,
	})
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error executing year template", "error", err.Error(), "remote_addr", r.RemoteAddr)
	}
}

func (c *controller) GetDaily(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	yearStr := vars["year"]
	year, err := strconv.ParseUint(yearStr, 10, 64)
	if err != nil {
		c.FourOFour(w, r)
		return
	}

	// UTC + 7
	now := time.Now().UTC().Add(7 * time.Hour)
	if year < 1 || year > uint64(now.Year()) {
		c.FourOFour(w, r)
		return
	}
	monthStr := vars["month"]
	month, err := strconv.ParseUint(monthStr, 10, 8)
	if err != nil {
		c.FourOFour(w, r)
		return
	}
	if year == uint64(now.Year()) && month > uint64(now.Month()) {
		c.FourOFour(w, r)
		return
	}
	if _, ok := monthDays[int(month)]; !ok {
		c.FourOFour(w, r)
		return
	}
	dailyData, err := c.repo.GetDailyByMonthAndYear(r.Context(), year, month, os.Getenv("TIME_OFFSET"))
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error getting daily 💩", "error", err.Error(), "remote_addr", r.RemoteAddr)
		WriteResponseJSON(w, http.StatusInternalServerError, "it's our fault, not yours!")
		return
	}

	mostPoopInADay, err := c.repo.GetMostPoopInADay(r.Context(), os.Getenv("TIME_OFFSET"))
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error getting most 💩 in a day", "error", err.Error(), "remote_addr", r.RemoteAddr)
		WriteResponseJSON(w, http.StatusInternalServerError, "it's our fault, not yours!")
		return
	}

	lastDataAt, err := c.repo.GetLastDataTimestamp(r.Context(), os.Getenv("TIME_OFFSET"))
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error getting last 💩", "error", err.Error(), "remote_addr", r.RemoteAddr)
		WriteResponseJSON(w, http.StatusInternalServerError, "it's our fault, not yours!")
		return
	}
	longestDayWithoutPoop, err := c.repo.GetLongestDayWithoutPoop(r.Context(), os.Getenv("TIME_OFFSET"))
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error getting longest day without 💩", "error", err.Error(), "remote_addr", r.RemoteAddr)
		WriteResponseJSON(w, http.StatusInternalServerError, "it's our fault, not yours!")
		return
	}

	maxDays := now.Day()
	if year < uint64(now.Year()) || (year == uint64(now.Year()) && month < uint64(now.Month())) {
		maxDays = monthDays[int(month)]
		if month == 2 && isLeapYear(int(year)) {
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

	w.WriteHeader(http.StatusOK)
	err = c.tmpl.ExecuteTemplate(w, "month", struct {
		LastDataAt            time.Time
		Data                  []model.AggData
		Year                  int
		Month                 int
		CurrentTime           time.Time
		LongestDayWithoutPoop model.LongestDayWithoutPoop
		MostPoopInADay        model.MostPoopInADay
	}{
		Year:                  int(year),
		Month:                 int(month),
		Data:                  dailyDataComplete,
		LastDataAt:            lastDataAt,
		CurrentTime:           now,
		LongestDayWithoutPoop: longestDayWithoutPoop,
		MostPoopInADay:        mostPoopInADay,
	})
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error executing month template", "error", err.Error(), "remote_addr", r.RemoteAddr)
	}
}

func (c *controller) FourOFour(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	err := c.tmpl.ExecuteTemplate(w, "404", nil)
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error executing 404 template", "error", err.Error(), "remote_addr", r.RemoteAddr)
	}
}

func (c *controller) GetLastPoopTime(w http.ResponseWriter, r *http.Request) {
	lastPoopTime, err := c.repo.GetLastDataTimestamp(r.Context(), os.Getenv("TIME_OFFSET"))
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error getting last poop time", "error", err.Error(), "remote_addr", r.RemoteAddr)
		WriteResponseJSON(w, http.StatusInternalServerError, "it's our fault, not yours!")
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(struct {
		LastPoopTime time.Time `json:"last_poop_time"`
	}{
		LastPoopTime: lastPoopTime,
	})
}

func (c *controller) GetSQLiteFile(w http.ResponseWriter, r *http.Request) {
	filePath := os.Getenv("DATA_SOURCE_NAME")
	_, err := os.Stat(filePath)
	if err != nil {
		c.logger.ErrorContext(r.Context(), "file doesn't exist", "error", err)
		WriteResponseJSON(w, http.StatusInternalServerError, "it's our fault, not yours!")
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename=berak.sqlite3")
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, filePath)
}
