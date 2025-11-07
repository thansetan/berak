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
	tmpl      *template.Template
	logger    *slog.Logger
	svc       *berakService
	newPoopCh chan struct{}
}

func NewController(svc *berakService, tmpl *template.Template, logger *slog.Logger) *controller {
	return &controller{tmpl, logger, svc, make(chan struct{}, 10)}
}

func (c *controller) Event(w http.ResponseWriter, r *http.Request) {
	period := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("period")))
	if period == "" || (period != "monthly" && period != "daily") {
		c.logger.WarnContext(r.Context(), "invalid period!", "period", period)
		return
	}
	c.logger.InfoContext(r.Context(), "client connected!", "ip_address", r.RemoteAddr, "params", r.URL.Query())
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to create watcher!", "error", err)
		helper.OurFault(w)
		return
	}
	defer watcher.Close()
	err = watcher.Add(os.Getenv("DATA_SOURCE_NAME"))
	if err != nil {
		c.logger.ErrorContext(r.Context(), "failed to watch sqlite file!", "error", err)
		helper.OurFault(w)
		return
	}
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				c.logger.ErrorContext(r.Context(), "event is not a watcher.Events!")
				break
			}
			if !event.Has(fsnotify.Write) {
				c.logger.InfoContext(r.Context(), "event is not a write event!", "event", event.Name)
				break
			}
			now := c.now()
			yearStr := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("year")))
			year, err := strconv.ParseUint(yearStr, 10, 64)
			if err != nil {
				c.logger.ErrorContext(r.Context(), "failed to parse year!", "error", err)
				break
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
					c.logger.ErrorContext(r.Context(), "failed to get monthly data!", "error", err)
					break
				}
				templateName = "monthly_table"
			case "daily":
				if !fetchTable {
					break
				}
				monthStr := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("month")))
				month, err := strconv.ParseUint(monthStr, 10, 64)
				if uint64(now.Month()) != month {
					fetchTable = false
					break
				}
				if err != nil {
					c.logger.ErrorContext(r.Context(), "failed to parse month!", "error", err)
					break
				}
				tableData, err = c.svc.GetDaily(r.Context(), now, year, month)
				if err != nil {
					c.logger.ErrorContext(r.Context(), "failed to get daily data!", "error", err)
					break
				}
				templateName = "daily_table"
			}
			m := make(map[string]string)

			var buf bytes.Buffer
			if fetchTable {
				err = c.tmpl.ExecuteTemplate(&buf, templateName, tableData)
				if err != nil {
					c.logger.ErrorContext(r.Context(), "failed to execute template!", "name", templateName, "error", err)
					break
				}
				m["poop-table"] = buf.String()
				buf.Reset()
			}

			stats, err := c.svc.GetStatistics(r.Context())
			if err != nil {
				c.logger.ErrorContext(r.Context(), "failed to get statistics!", "error", err)
			}

			err = c.tmpl.ExecuteTemplate(&buf, "footer", stats)
			if err != nil {
				c.logger.ErrorContext(r.Context(), "failed to execute template!", "name", "footer", "error", err)
				break
			}
			m["poop-footer"] = buf.String()

			fmt.Fprint(w, "event:poopupdate\n")
			jsonBytes, err := json.Marshal(m)
			if err != nil {
				c.logger.ErrorContext(r.Context(), "failed to marshal json!", "error", err)
				break
			}
			fmt.Fprintf(w, "data:%s\n\n", string(jsonBytes))
			rc := http.NewResponseController(w)
			err = rc.Flush()
			if err != nil {
				c.logger.ErrorContext(r.Context(), "failed to flush to writer!", "error", err)
			}
		case <-r.Context().Done():
			c.logger.InfoContext(r.Context(), "client disconnected!", "ip_address", r.RemoteAddr)
			return
		}
	}
}

func (c *controller) Create(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Timestamp time.Time `json:"timestamp"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil && !errors.Is(err, io.EOF) {
		c.logger.ErrorContext(r.Context(), "error adding new 💩", "error", err.Error(), "remote_addr", r.RemoteAddr)
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
		c.logger.ErrorContext(r.Context(), "error adding new 💩", "error", err.Error(), "remote_addr", r.RemoteAddr)
		helper.OurFault(w)
		return
	}
	c.logger.InfoContext(r.Context(), "new 💩 added!", "remote_addr", r.RemoteAddr)
	w.WriteHeader(http.StatusNoContent)
}

func (c *controller) Delete(w http.ResponseWriter, r *http.Request) {
	err := c.svc.DeleteLast(r.Context())
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error removing last 💩", "error", err.Error(), "remote_addr", r.RemoteAddr)
		helper.OurFault(w)
		return
	}
	c.logger.InfoContext(r.Context(), "last 💩 removed!", "remote_addr", r.RemoteAddr)
	w.WriteHeader(http.StatusNoContent)
}

func (c *controller) now() time.Time {
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

	// UTC + 7
	now := c.now()
	if year < 1 || year > uint64(now.Year()) {
		c.FourOFour(w, r)
		return
	}

	tableData, err := c.svc.GetMonthly(r.Context(), now, year)
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error getting monthly data!", "error", err)
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
		c.logger.ErrorContext(r.Context(), "error executing year template", "error", err.Error(), "remote_addr", r.RemoteAddr)
		helper.OurFault(w)
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
	now := c.now()
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
	tableData, err := c.svc.GetDaily(r.Context(), now, year, month)
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error getting daily data!", "error", err)
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
		c.logger.ErrorContext(r.Context(), "error executing month template", "error", err.Error(), "remote_addr", r.RemoteAddr)
	}
}

func (c *controller) GetLastPoopTime(w http.ResponseWriter, r *http.Request) {
	lastPoopTime, err := c.svc.GetLastPoopTime(r.Context())
	if err != nil {
		c.logger.ErrorContext(r.Context(), "error getting last poop time", "error", err.Error(), "remote_addr", r.RemoteAddr)
		helper.OurFault(w)
		return
	}
	helper.WriteResponseJSON(w, http.StatusOK, struct {
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
		c.logger.ErrorContext(r.Context(), "error executing 404 template", "error", err.Error(), "remote_addr", r.RemoteAddr)
	}
}
