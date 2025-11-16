package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/thansetan/berak/helper"
)

type RateLimit struct {
	store                     *sync.Map
	keyGetter                 func(r *http.Request) string
	maxVisitCount             uint64
	duration, cleanupDuration time.Duration
}

type visitor struct {
	mu          *sync.RWMutex
	windowStart time.Time
	count       uint64
}

func NewRateLimit(maxVisitCount uint64, duration, cleanupDuration time.Duration, keyGetter func(*http.Request) string) *RateLimit {
	rl := new(RateLimit)
	rl.maxVisitCount = (maxVisitCount)
	rl.duration = duration
	rl.cleanupDuration = cleanupDuration
	rl.keyGetter = keyGetter
	rl.store = new(sync.Map)
	go rl.cleanup()

	return rl
}

func (rl *RateLimit) Handle(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := rl.keyGetter(r)
		valAny, _ := rl.store.LoadOrStore(key, &visitor{
			windowStart: time.Now(),
			mu:          new(sync.RWMutex),
		})
		val := valAny.(*visitor)

		val.mu.Lock()
		if time.Since(val.windowStart) > rl.duration {
			val.count = 0
			val.windowStart = time.Now()
		}

		if time.Since(val.windowStart) <= rl.duration && val.count == rl.maxVisitCount {
			val.mu.Unlock()
			helper.WriteMessage(w, http.StatusTooManyRequests, "kecepeten 😡!")
			return
		}
		val.count++
		val.mu.Unlock()

		next.ServeHTTP(w, r)
	}
}

func (rl *RateLimit) cleanup() {
	for range time.Tick(rl.cleanupDuration) {
		rl.store.Range(func(key, value any) bool {
			v := value.(*visitor)
			v.mu.RLock()
			windowPassed := time.Since(v.windowStart) > rl.duration
			v.mu.RUnlock()
			if windowPassed {
				rl.store.Delete(key)
			}

			return true
		})
	}
}
