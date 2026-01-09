package berak

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/mux"
	"github.com/thansetan/berak/helper"
	"github.com/thansetan/berak/model"
)

type controller struct {
	tmpl   *template.Template
	logger *slog.Logger
	svc    *berakService
}

func NewController(svc *berakService, tmpl *template.Template, logger *slog.Logger) *controller {
	return &controller{tmpl, logger, svc}
}

func (c *controller) Event(w http.ResponseWriter, r *http.Request) {
	period := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("period")))
	if period == "" || (period != "monthly" && period != "daily") {
		c.logger.WarnContext(r.Context(), "invalid period!", "period", period)
		return
	}
	c.logger.InfoContext(r.Context(), "client connected!", "remote_addr", r.RemoteAddr, "params", r.URL.Query())
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", os.Getenv("ALLOWED_SSE_ORIGINS"))
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("X-Accel-Buffering", "no")

	fmt.Fprint(w, "retry:3000\n\n")
	rc := http.NewResponseController(w)
	rc.Flush()

	err := c.sendPoopData(w, r, period)
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to send poop data!", "error", err, "remote_addr", r.RemoteAddr)
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to create watcher!", "error", err, "remote_addr", r.RemoteAddr)
		helper.OurFault(w)
		return
	}
	defer watcher.Close()
	err = watcher.Add(os.Getenv("DATA_SOURCE_NAME"))
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to watch sqlite file!", "error", err, "remote_addr", r.RemoteAddr)
		helper.OurFault(w)
		return
	}

	keepaliveTicker := time.NewTicker(25 * time.Second)
	defer keepaliveTicker.Stop()

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				c.logger.ErrorContext(r.Context(), "failed to read data from channel!")
				break
			}
			if !event.Has(fsnotify.Write) {
				c.logger.InfoContext(r.Context(), "event is not a write event!", "event", event.Name)
				break
			}
			err = c.sendPoopData(w, r, period)
			if err != nil {
				c.logger.ErrorContext(r.Context(), "failed to send poop data!", "error", err, "remote_addr", r.RemoteAddr)
			}
		case <-keepaliveTicker.C:
			_, err := fmt.Fprint(w, ":ping\n\n")
			if err != nil {
				c.logger.InfoContext(r.Context(), "keepalive ping failed, client likely disconnected", "remote_addr", r.RemoteAddr)
				return
			}
			err = rc.Flush()
			if err != nil {
				c.logger.InfoContext(r.Context(), "flush failed, client likely disconnected", "remote_addr", r.RemoteAddr)
				return
			}
		case <-r.Context().Done():
			c.logger.InfoContext(r.Context(), "client disconnected!", "remote_addr", r.RemoteAddr)
			return
		}
	}
}

func (c *controller) sendPoopData(w http.ResponseWriter, r *http.Request, period string) error {
	now := c.now()
	yearStr := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("year")))
	year, err := strconv.ParseUint(yearStr, 10, 64)
	if err != nil {
		return fmt.Errorf("error parsing year: %w", err)
	}
	var (
		tableData    model.TableData
		templateName string
		fetchTable   = uint64(now.Year()) == year
	)
	switch period {
	case "monthly":
		if !fetchTable {
			break
		}
		tableData, err = c.svc.GetMonthly(r.Context(), now, year)
		if err != nil {
			return fmt.Errorf("error getting monthly data: %w", err)
		}
		templateName = "monthly_table"
	case "daily":
		if !fetchTable {
			break
		}
		monthStr := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("month")))
		month, err := strconv.ParseUint(monthStr, 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing month: %w", err)
		}
		if uint64(now.Month()) != month {
			fetchTable = false
			break
		}
		tableData, err = c.svc.GetDaily(r.Context(), now, year, month)
		if err != nil {
			return fmt.Errorf("error getting daily data: %w", err)
		}
		templateName = "daily_table"
	}
	m := make(map[string]string)

	var buf bytes.Buffer
	if fetchTable {
		err = c.tmpl.ExecuteTemplate(&buf, templateName, tableData)
		if err != nil {
			return fmt.Errorf("error executing template[name=%s]: %w", templateName, err)
		}
		m["poop-table"] = buf.String()
		buf.Reset()
	}

	stats, err := c.svc.GetStatistics(r.Context())
	if err != nil {
		return fmt.Errorf("error getting statistics: %w", err)
	}

	err = c.tmpl.ExecuteTemplate(&buf, "footer", stats)
	if err != nil {
		return fmt.Errorf("error executing template[name=footer]: %w", err)
	}
	m["poop-footer"] = buf.String()

	fmt.Fprint(w, "event:poopupdate\n")
	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("error marshalling JSON: %w", err)
	}
	fmt.Fprintf(w, "data:%s\n\n", string(jsonBytes))
	rc := http.NewResponseController(w)
	err = rc.Flush()
	if err != nil {
		return fmt.Errorf("error flushing writer: %w", err)
	}
	return nil
}

func (c *controller) Create(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Timestamp time.Time `json:"timestamp"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil && !errors.Is(err, io.EOF) {
		c.logger.ErrorContext(r.Context(), "failed to add new 💩", "error", err.Error(), "remote_addr", r.RemoteAddr, "error_type", fmt.Sprintf("%T", err))
		_, jsonUnmarshalTypeErrorOK := err.(*json.UnmarshalTypeError)
		_, jsonSyntaxErrorOK := err.(*json.SyntaxError)
		if jsonUnmarshalTypeErrorOK || jsonSyntaxErrorOK || errors.Is(err, io.ErrUnexpectedEOF) {
			helper.WriteMessage(w, http.StatusBadRequest, "invalid JSON format!")
			return
		}
		if timeErr, ok := err.(*time.ParseError); ok {
			helper.WriteMessage(w, http.StatusBadRequest, fmt.Sprintf("failed to parse timestamp%s", timeErr.Message))
			return
		}
		helper.OurFault(w)
		return
	}
	defer r.Body.Close()
	if err == nil {
		if data.Timestamp.IsZero() {
			helper.WriteMessage(w, http.StatusBadRequest, "timestamp can't be empty!")
			return
		}
		if data.Timestamp.After(time.Now()) {
			c.logger.InfoContext(r.Context(), "time after", "now", time.Now(), "parsed", data.Timestamp)
			helper.WriteMessage(w, http.StatusBadRequest, "💩 time can't be after current time!")
			return
		}
	}
	err = c.svc.Add(r.Context(), data.Timestamp.UTC())
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to add new 💩", "error", err.Error(), "remote_addr", r.RemoteAddr)
		helper.OurFault(w)
		return
	}
	c.logger.InfoContext(r.Context(), "new 💩 added!", "remote_addr", r.RemoteAddr)
	w.WriteHeader(http.StatusNoContent)
}

func (c *controller) Delete(w http.ResponseWriter, r *http.Request) {
	err := c.svc.DeleteLast(r.Context())
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to remove last 💩", "error", err.Error(), "remote_addr", r.RemoteAddr)
		helper.OurFault(w)
		return
	}
	c.logger.InfoContext(r.Context(), "last 💩 removed!", "remote_addr", r.RemoteAddr)
	w.WriteHeader(http.StatusNoContent)
}

func (c *controller) now() time.Time {
	// UTC + 7
	return time.Now().UTC().Add(7 * time.Hour)
}

func (c *controller) GetMonthly(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	yearStr := vars["year"]
	year, err := strconv.ParseUint(yearStr, 10, 64)
	if err != nil {
		c.FourOFour(w, r)
		return
	}

	now := c.now()
	if year < 1 || year > uint64(now.Year()) {
		c.FourOFour(w, r)
		return
	}

	tableData, err := c.svc.GetMonthly(r.Context(), now, year)
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to get monthly data!", "error", err)
		helper.OurFault(w)
		return
	}
	stats, err := c.svc.GetStatistics(r.Context())
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to get statistics data!", "error", err)
		helper.OurFault(w)
		return
	}

	w.WriteHeader(http.StatusOK)
	err = c.tmpl.ExecuteTemplate(w, "year", model.Data{
		Year:       int(year),
		TableData:  tableData,
		Statistics: stats,
	})
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to execute year template", "error", err.Error(), "remote_addr", r.RemoteAddr)
		helper.OurFault(w)
	}
}

func (c *controller) GetDaily(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	yearStr := vars["year"]
	now := c.now()
	year, err := strconv.ParseUint(yearStr, 10, 64)
	if err != nil || (year < 1 || year > uint64(now.Year())) {
		c.FourOFour(w, r)
		return
	}

	monthStr := vars["month"]
	month, err := strconv.ParseUint(monthStr, 10, 8)
	if err != nil || (month < 1 || month > 12) {
		c.FourOFour(w, r)
		return
	}

	if year == uint64(now.Year()) && month > uint64(now.Month()) {
		c.FourOFour(w, r)
		return
	}

	tableData, err := c.svc.GetDaily(r.Context(), now, year, month)
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to get daily data!", "error", err)
		helper.OurFault(w)
		return
	}
	stats, err := c.svc.GetStatistics(r.Context())
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to get statistics data!", "error", err)
		helper.OurFault(w)
		return
	}

	w.WriteHeader(http.StatusOK)
	err = c.tmpl.ExecuteTemplate(w, "month", model.Data{
		Year:       int(year),
		Month:      int(month),
		TableData:  tableData,
		Statistics: stats,
	})
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to execute month template", "error", err.Error(), "remote_addr", r.RemoteAddr)
	}
}

func (c *controller) GetLastPoopTime(w http.ResponseWriter, r *http.Request) {
	lastPoopTime, err := c.svc.GetLastPoopTime(r.Context())
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to get last poop time", "error", err.Error(), "remote_addr", r.RemoteAddr)
		helper.OurFault(w)
		return
	}
	helper.WriteJSON(w, http.StatusOK, struct {
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
		helper.OurFault(w)
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename=berak.sqlite3")
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, filePath)
}

func (c controller) FourOFour(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	err := c.tmpl.ExecuteTemplate(w, "404", nil)
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to execute 404 template", "error", err.Error(), "remote_addr", r.RemoteAddr)
	}
}
