package berak

import (
	"html/template"
	"net/http"
	"strconv"
	"time"
)

type controller struct {
	repo repository
	tmpl *template.Template
}

func NewController(repo repository, tmpl *template.Template) *controller {
	return &controller{repo, tmpl}
}

func (c *controller) Create(w http.ResponseWriter, r *http.Request) {
	err := c.repo.Add(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (c *controller) Delete(w http.ResponseWriter, r *http.Request) {
	err := c.repo.DeleteLast(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *controller) GetMonthly(w http.ResponseWriter, r *http.Request) {
	yearStr := r.PathValue("year")
	year, err := strconv.ParseUint(yearStr, 10, 64)
	if err != nil {
		_ = c.tmpl.ExecuteTemplate(w, "404", nil)
		return
	}

	monthlyData, err := c.repo.GetMonthlyByYear(r.Context(), year)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	if len(monthlyData) == 0 {
		_ = c.tmpl.ExecuteTemplate(w, "404", nil)
		return
	}

	lastDataAt, err := c.repo.GetLastDataTimestamp(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	_ = c.tmpl.ExecuteTemplate(w, "year", struct {
		LastDataAt time.Time
		Data       []AggData
		Year       int
	}{
		Year:       int(year),
		Data:       monthlyData,
		LastDataAt: lastDataAt,
	})
}

func (c *controller) GetDaily(w http.ResponseWriter, r *http.Request) {
	yearStr := r.PathValue("year")
	year, err := strconv.ParseUint(yearStr, 10, 64)
	if err != nil {
		_ = c.tmpl.ExecuteTemplate(w, "404", nil)
		return
	}
	monthStr := r.PathValue("month")
	month, err := strconv.ParseUint(monthStr, 10, 8)
	if err != nil {
		_ = c.tmpl.ExecuteTemplate(w, "404", nil)
		return
	}
	dailyData, err := c.repo.GetDailyByMonthAndYear(r.Context(), year, month)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	if len(dailyData) == 0 {
		_ = c.tmpl.ExecuteTemplate(w, "404", nil)
		return
	}
	lastDataAt, err := c.repo.GetLastDataTimestamp(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	_ = c.tmpl.ExecuteTemplate(w, "month", struct {
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
}
