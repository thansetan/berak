package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
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
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*
var staticDirFS embed.FS

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	AddSource: true,
}))

func main() {
	db, err := db.NewConn(os.Getenv("DATA_SOURCE_NAME"))
	if err != nil {
		panic(err)
	}
	repo := berak.NewRepo(db)
	tmpl := template.New("").Funcs(template.FuncMap{
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
			if n == 0 {
				return "-"
			}
			return strings.Repeat("💩", n)
		},
	})

	err = filepath.WalkDir("templates", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && filepath.Ext(path) == ".html" {
			_, err = tmpl.ParseFiles(path)
		}
		return err
	})
	if err != nil {
		panic(err)
	}
	controller := berak.NewController(repo, tmpl, logger)

	r := mux.NewRouter()

	r.NotFoundHandler = http.HandlerFunc(controller.FourOFour)
	r.MethodNotAllowedHandler = http.HandlerFunc(controller.FourOFour)

	r.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		http.Redirect(w, r, fmt.Sprintf("/%d", now.Year()), http.StatusTemporaryRedirect)
	})
	r.Path("/berak").HandlerFunc(rateLimit(protected(http.HandlerFunc(controller.Create)))).Methods(http.MethodPost)
	r.Path("/berak").HandlerFunc(rateLimit(protected(http.HandlerFunc(controller.Delete)))).Methods(http.MethodDelete)
	r.Path("/{year:[0-9]+}").HandlerFunc(controller.GetMonthly).Methods(http.MethodGet)
	r.Path("/{year:[0-9]+}/{month:[0-9]+}").HandlerFunc(controller.GetDaily).Methods(http.MethodGet)
	r.Path("/last_poop").HandlerFunc(controller.GetLastPoopTime).Methods(http.MethodGet)
	r.Path("/healthcheck").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	r.Path("/download").HandlerFunc(protected(http.HandlerFunc(controller.GetSQLiteFile))).Methods(http.MethodGet)

	staticFilesFS, err := fs.Sub(staticDirFS, "static")
	if err != nil {
		logger.Error("static dir doesn't exists!")
		os.Exit(1)
	}

	r.PathPrefix("/").Handler(http.StripPrefix("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fsHandler := http.FileServer(http.FS(staticFilesFS))
		if _, err := staticFilesFS.Open(r.URL.Path); os.IsNotExist(err) {
			controller.FourOFour(w, r)
			return
		}
		fsHandler.ServeHTTP(w, r)
	})))

	srv := new(http.Server)
	srv.Handler = logRequest(r)
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

type wrappedResponseWriter struct {
	http.ResponseWriter
	code int
}

func (wrw *wrappedResponseWriter) WriteHeader(code int) {
	wrw.code = code
	wrw.ResponseWriter.WriteHeader(code)
}

func logRequest(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wrw := &wrappedResponseWriter{
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
