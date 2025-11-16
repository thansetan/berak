package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/joho/godotenv/autoload"
	"github.com/thansetan/berak/berak"
	"github.com/thansetan/berak/db"
	"github.com/thansetan/berak/helper"
	"github.com/thansetan/berak/middleware"
)

var (
	//go:embed templates/*
	templateDirFS embed.FS

	//go:embed static/*
	staticDirFS embed.FS

	apiKeys = new(sync.Map)
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}))
	db, err := db.NewConn(os.Getenv("DATA_SOURCE_NAME"))
	if err != nil {
		logger.Error("failed to establish database connection!", "error", "err")
		os.Exit(1)
	}

	tmpl := template.New("").Funcs(template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"getMonthName": func(monthNumber int) string {
			return helper.GetMonth(monthNumber).Name
		},
		"tai": func(n int) string {
			if n == 0 {
				return "-"
			}
			return strings.Repeat("💩", n)
		},
	})

	err = fs.WalkDir(templateDirFS, ".", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && filepath.Ext(path) == ".html" {
			_, err = tmpl.ParseFS(templateDirFS, path)
		}
		return err
	})
	if err != nil {
		logger.Error("failed to load template!", "error", err)
		os.Exit(1)
	}

	repo := berak.NewRepo(db)
	svc := berak.NewService(repo, os.Getenv("TIME_OFFSET"))
	controller := berak.NewController(svc, tmpl, logger)

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(controller.FourOFour)
	r.MethodNotAllowedHandler = http.HandlerFunc(controller.FourOFour)

	apiKeyRateLimiter := middleware.NewRateLimit(1, time.Minute, 1*time.Hour, func(r *http.Request) string {
		return r.Header.Get("X-Api-Key")
	})

	ipRateLimiter := middleware.NewRateLimit(5, time.Minute, 1*time.Hour, func(r *http.Request) string {
		if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[0])
		}
		if xr := strings.TrimSpace(r.Header.Get("X-Real-Ip")); xr != "" {
			return xr
		}
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return r.RemoteAddr
		}
		return host
	})

	loggerMW := middleware.NewLogger(logger)

	{
		r.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()
			http.Redirect(w, r, fmt.Sprintf("/%d", now.Year()), http.StatusTemporaryRedirect)
		})
		r.Path("/sse").HandlerFunc(controller.Event).Methods(http.MethodGet)
		r.Path("/berak").HandlerFunc(ipRateLimiter.Handle(protected(apiKeyRateLimiter.Handle(http.HandlerFunc(controller.Create))))).Methods(http.MethodPost)
		r.Path("/berak").HandlerFunc(ipRateLimiter.Handle(protected(apiKeyRateLimiter.Handle(http.HandlerFunc(controller.Delete))))).Methods(http.MethodDelete)
		r.Path("/{year:[0-9]+}").HandlerFunc(controller.GetMonthly).Methods(http.MethodGet)
		r.Path("/{year:[0-9]+}/{month:[0-9]+}").HandlerFunc(controller.GetDaily).Methods(http.MethodGet)
		r.Path("/last_poop").HandlerFunc(controller.GetLastPoopTime).Methods(http.MethodGet)
		r.Path("/healthcheck").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		})
		r.Path("/download").HandlerFunc(protected(http.HandlerFunc(controller.GetSQLiteFile))).Methods(http.MethodGet)
	}

	staticFilesFS, err := fs.Sub(staticDirFS, "static")
	if err != nil {
		logger.Error("failed to get static files directory!", "error", err)
		os.Exit(1)
	}

	r.PathPrefix("/").Handler(http.StripPrefix("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fsHandler := http.FileServer(http.FS(staticFilesFS))
		if _, err := staticFilesFS.Open(r.URL.Path); err != nil {
			if !os.IsNotExist(err) {
				logger.ErrorContext(r.Context(), "failed to open file!", "filepath", r.URL.Path, "error", err)
			}
			controller.FourOFour(w, r)
			return
		}
		fsHandler.ServeHTTP(w, r)
	})))

	srv := new(http.Server)
	srv.Handler = loggerMW.Handle(r)
	srv.Addr = net.JoinHostPort("0.0.0.0", os.Getenv("PORT"))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to listen and serve HTTP connection!", "error", err)
		}
	}()
	logger.Info(fmt.Sprintf("server listeing at %s", srv.Addr))
	<-ctx.Done()
	logger.Info("shutting down server")
	err = srv.Shutdown(ctx)
	if err != nil {
		logger.Error("failed to shut down server!", "error", err)
	}
}

func protected(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != os.Getenv("BERAK_KEY") {
			helper.WriteMessage(w, http.StatusUnauthorized, "gaboleh 😡")
			return
		}
		next.ServeHTTP(w, r)
	}
}
