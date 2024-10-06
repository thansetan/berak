package berak

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
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
	err := c.repo.Add(r.Context())
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
	yearStr := r.PathValue("year")
	year, err := strconv.ParseUint(yearStr, 10, 64)
	if err != nil {
		c.fourOFour(w, r)
		return
	}

	monthlyData, err := c.repo.GetMonthlyByYear(r.Context(), year, os.Getenv("TIME_OFFSET"))
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error getting monthly 💩", "error", err.Error(), "remote_addr", r.RemoteAddr)
		WriteResponseJSON(w, http.StatusInternalServerError, "it's our fault, not yours!")
		return
	}

	if len(monthlyData) == 0 {
		c.fourOFour(w, r)
		return
	}

	lastDataAt, err := c.repo.GetLastDataTimestamp(r.Context(), os.Getenv("TIME_OFFSET"))
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error getting last 💩", "error", err.Error(), "remote_addr", r.RemoteAddr)
		WriteResponseJSON(w, http.StatusInternalServerError, "it's our fault, not yours!")
		return
	}
	w.WriteHeader(http.StatusOK)
	err = c.tmpl.ExecuteTemplate(w, "year", struct {
		LastDataAt time.Time
		Data       []AggData
		Year       int
	}{
		Year:       int(year),
		Data:       monthlyData,
		LastDataAt: lastDataAt,
	})
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error executing year template", "error", err.Error(), "remote_addr", r.RemoteAddr)
	}
}

func (c *controller) GetDaily(w http.ResponseWriter, r *http.Request) {
	yearStr := r.PathValue("year")
	year, err := strconv.ParseUint(yearStr, 10, 64)
	if err != nil {
		c.fourOFour(w, r)
		return
	}
	monthStr := r.PathValue("month")
	month, err := strconv.ParseUint(monthStr, 10, 8)
	if err != nil {
		c.fourOFour(w, r)
		return
	}
	dailyData, err := c.repo.GetDailyByMonthAndYear(r.Context(), year, month, os.Getenv("TIME_OFFSET"))
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error getting daily 💩", "error", err.Error(), "remote_addr", r.RemoteAddr)
		WriteResponseJSON(w, http.StatusInternalServerError, "it's our fault, not yours!")
		return
	}
	if len(dailyData) == 0 {
		c.fourOFour(w, r)
		return
	}
	lastDataAt, err := c.repo.GetLastDataTimestamp(r.Context(), os.Getenv("TIME_OFFSET"))
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error getting last 💩", "error", err.Error(), "remote_addr", r.RemoteAddr)
		WriteResponseJSON(w, http.StatusInternalServerError, "it's our fault, not yours!")
		return
	}
	w.WriteHeader(http.StatusOK)
	err = c.tmpl.ExecuteTemplate(w, "month", struct {
		LastDataAt time.Time
		Data       []AggData
		Year       int
		Month      int
	}{
		Year:       int(year),
		Month:      int(month),
		Data:       dailyData,
		LastDataAt: lastDataAt,
	})
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error executing month template", "error", err.Error(), "remote_addr", r.RemoteAddr)
	}
}

func (c *controller) fourOFour(w http.ResponseWriter, r *http.Request) {
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
