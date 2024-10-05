package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/thansetan/berak/berak"
	"github.com/thansetan/berak/db"
)

//go:embed templates/*
var templatesFS embed.FS

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	AddSource: true,
}))

func main() {
	db, err := db.NewConn(os.Getenv("DATA_SOURCE_NAME"))
	if err != nil {
		panic(err)
	}
	repo := berak.NewRepo(db)
	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"getMonthName": func(monthNumber int) string {
			if monthNumber == 0 || monthNumber > 12 {
				return ""
			}
			return []string{
				"January",
				"February",
				"March",
				"April",
				"May",
				"June",
				"July",
				"August",
				"September",
				"October",
				"November",
				"December",
			}[monthNumber-1]
		},
		"tai": func(n int) string {
			return strings.Repeat("💩", n)
		},
	}).ParseFS(templatesFS, "*/*.html"))
	controller := berak.NewController(repo, tmpl, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		http.Redirect(w, r, fmt.Sprintf("/%d", now.Year()), http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("POST /berak", rateLimit(protected(http.HandlerFunc(controller.Create))))
	mux.HandleFunc("DELETE /berak", rateLimit(protected(http.HandlerFunc(controller.Delete))))
	mux.HandleFunc("GET /{year}", controller.GetMonthly)
	mux.HandleFunc("GET /{year}/{month}", controller.GetDaily)
	mux.HandleFunc("GET /last_poop", controller.GetLastPoopTime)
	mux.HandleFunc("GET /healthcheck", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	srv := new(http.Server)
	srv.Handler = logRequest(mux)
	srv.Addr = fmt.Sprintf("0.0.0.0:%s", os.Getenv("PORT"))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http error listening", "error", err.Error())
		}
	}()
	logger.Info(fmt.Sprintf("server listeing at %s", srv.Addr))
	<-ctx.Done()
	logger.Info("shutting down server")
	err = srv.Shutdown(ctx)
	if err != nil {
		logger.Error("error shutting down server", "error", err.Error())
	}
}

func protected(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != os.Getenv("BERAK_KEY") {
			logger.WarnContext(r.Context(), "incorrect API-Key", "remote_addr", r.RemoteAddr, "api-key", r.Header.Get("X-Api-Key"))
			berak.WriteResponseJSON(w, http.StatusUnauthorized, "gaboleh 😡")
			return
		}
		next.ServeHTTP(w, r)
	}
}

var users = new(sync.Map)

func rateLimit(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-KEY")
		if apiKey == "" {
			logger.WarnContext(r.Context(), "empty API-Key", "remote_addr", r.RemoteAddr)
			berak.WriteResponseJSON(w, http.StatusUnauthorized, "gaboleh 😡")
			return
		}
		lastAccess, ok := users.LoadOrStore(apiKey, time.Now())
		if ok {
			lastAccessedAt, ok := lastAccess.(time.Time)
			if !ok {
				logger.ErrorContext(r.Context(), "failed to assert interface to time.Time", "remote_addr", r.RemoteAddr)
				berak.WriteResponseJSON(w, http.StatusInternalServerError, "it's our fault, not yours!")
				return
			}
			if time.Since(lastAccessedAt) < time.Minute {
				logger.WarnContext(r.Context(), "rate limited", "api-key", apiKey, "remote_addr", r.RemoteAddr)
				berak.WriteResponseJSON(w, http.StatusTooManyRequests, "kecepeten 😡")
				return
			}
			users.Store(apiKey, time.Now())
		}
		next.ServeHTTP(w, r)
	}
}

type wrapperResponseWriter struct {
	http.ResponseWriter
	code int
}

func (wrw *wrapperResponseWriter) WriteHeader(code int) {
	wrw.code = code
	wrw.ResponseWriter.WriteHeader(code)
}

func logRequest(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wrw := &wrapperResponseWriter{
			ResponseWriter: w,
		}
		t0 := time.Now()
		next.ServeHTTP(wrw, r)
		if wrw.code < 400 {
			logger.InfoContext(r.Context(), fmt.Sprintf("%s %s %s", r.Method, r.RequestURI, r.Proto), "remote_addr", r.RemoteAddr, "code", wrw.code, "took", time.Since(t0))
		} else if wrw.code < 500 {
			logger.WarnContext(r.Context(), fmt.Sprintf("%s %s %s", r.Method, r.RequestURI, r.Proto), "remote_addr", r.RemoteAddr, "code", wrw.code, "took", time.Since(t0))
		} else {
			logger.ErrorContext(r.Context(), fmt.Sprintf("%s %s %s", r.Method, r.RequestURI, r.Proto), "remote_addr", r.RemoteAddr, "code", wrw.code, "took", time.Since(t0))
		}
	}
}
