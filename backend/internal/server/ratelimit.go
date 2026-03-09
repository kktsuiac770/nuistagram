package server

import (
	"net/http"
	"sync"
	"time"
)

type rateLimiter struct {
	mu          sync.RWMutex
	attempts    map[string][]time.Time
	maxAttempts int
	window      time.Duration
}

func newRateLimiter(maxAttempts int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		attempts:    make(map[string][]time.Time),
		maxAttempts: maxAttempts,
		window:      window,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, timestamps := range rl.attempts {
			var valid []time.Time
			for _, t := range timestamps {
				if now.Sub(t) <= rl.window {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.attempts, ip)
			} else {
				rl.attempts[ip] = valid
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	timestamps := rl.attempts[ip]
	var valid []time.Time
	for _, t := range timestamps {
		if now.Sub(t) <= rl.window {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.maxAttempts {
		rl.attempts[ip] = valid
		return false
	}

	rl.attempts[ip] = append(valid, now)
	return true
}

func (rl *rateLimiter) retryAfter(ip string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	timestamps := rl.attempts[ip]
	if len(timestamps) == 0 {
		return 0
	}

	oldest := timestamps[0]
	for _, t := range timestamps {
		if t.Before(oldest) {
			oldest = t
		}
	}

	retryAt := oldest.Add(rl.window)
	remaining := time.Until(retryAt)
	if remaining < 0 {
		return 0
	}
	return int(remaining.Seconds())
}

var loginLimiter = newRateLimiter(5, time.Minute)

func rateLimitLogin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := getRealIP(r)

		if !loginLimiter.allow(ip) {
			retryAfter := loginLimiter.retryAfter(ip)
			w.Header().Set("Retry-After", string(rune(retryAfter)))
			jsonError(w, http.StatusTooManyRequests, "Too many login attempts. Please try again later.")
			return
		}

		next(w, r)
	}
}

func getRealIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}
