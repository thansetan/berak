package api

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/thansetan/berak/berak"
	"github.com/thansetan/berak/db"
)

func HandlerBerak(w http.ResponseWriter, r *http.Request) {
	db, err := db.NewConn(os.Getenv("POSTGRES_URL"))
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
	}).ParseGlob("../templates/*.html"))
	controller := berak.NewController(repo, tmpl)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		http.Redirect(w, r, fmt.Sprintf("/%d", now.Year()), http.StatusTemporaryRedirect)
	})
	mux.Handle("POST /berak", rateLimit(protected(controller.Create)))
	mux.Handle("DELETE /berak", rateLimit(protected(controller.Delete)))
	mux.HandleFunc("GET /{year}", controller.GetMonthly)
	mux.HandleFunc("GET /{year}/{month}", controller.GetDaily)
	// err = http.ListenAndServe(":8080", mux)
	// if err != nil {
	// 	panic(err)
	// }
	mux.ServeHTTP(w, r)
}

func protected(next func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-KEY") != os.Getenv("BERAK_KEY") {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("gaboleh"))
			return
		}
		next(w, r)
	}
}

var users = new(sync.Map)

func rateLimit(next func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-KEY")
		if apiKey == "" {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("gaboleh"))
			return
		}
		lastAccess, ok := users.LoadOrStore(apiKey, time.Now())
		if ok {
			lastAccessedAt, ok := lastAccess.(time.Time)
			if !ok {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("error"))
				return
			}
			if time.Since(lastAccessedAt) < time.Minute {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte("kecepeten"))
				return
			}
			users.Store(apiKey, time.Now())
		}
		next(w, r)
	}
}
