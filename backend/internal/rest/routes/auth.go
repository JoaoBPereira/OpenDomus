package routes

import (
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"od/internal/managers"
)

// version of the API (backend/package.json)
const version = "1.0.34"

// rateLimiter allows at most `limit` requests per IP in a `window` of time
type rateLimiter struct {
	window   time.Duration
	limit    int
	mutex    sync.Mutex
	attempts map[string][]time.Time
}

func newRateLimiter(window time.Duration, limit int) *rateLimiter {
	return &rateLimiter{window: window, limit: limit, attempts: map[string][]time.Time{}}
}

func (limiter *rateLimiter) allow(ip string) bool {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	now := time.Now()
	recent := []time.Time{}
	for _, attempt := range limiter.attempts[ip] {
		if now.Sub(attempt) < limiter.window {
			recent = append(recent, attempt)
		}
	}
	if len(recent) >= limiter.limit {
		limiter.attempts[ip] = recent
		return false
	}
	limiter.attempts[ip] = append(recent, now)
	return true
}

func clientIP(request *http.Request) string {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		return request.RemoteAddr
	}
	return host
}

func (h *Handlers) registerAuth(mux *http.ServeMux) {
	loginRateLimit := newRateLimiter(15*time.Minute, 5)

	HandleFunc(mux, "GET /{$}", func(response http.ResponseWriter, request *http.Request) {
		SendText(response, http.StatusOK, "Hello from Open Domus! v"+version)
	})

	HandleFunc(mux, "POST /login", func(response http.ResponseWriter, request *http.Request) {
		if !loginRateLimit.allow(clientIP(request)) {
			SendMessage(response, http.StatusTooManyRequests, "Too many login attempts, please try again later.")
			return
		}

		var body LoginSchema
		if err := ParseBody(request, &body); err != nil {
			SendMessage(response, http.StatusBadRequest, "Invalid request body: "+err.Error())
			return
		}
		if message := body.Validate(); message != "" {
			SendMessage(response, http.StatusBadRequest, message)
			return
		}

		role, err := h.Users.Login(request.Context(), body.Username, body.Password)
		if errors.Is(err, managers.ErrInvalidCredentials) {
			SendMessage(response, http.StatusUnauthorized, "Invalid credentials")
			return
		}
		if err != nil {
			SendError(response, err, "", "Login failed")
			return
		}

		validFor := 24 * time.Hour
		if role == "admin" {
			validFor = 720 * time.Hour
		}
		token, err := h.auth.CreateToken(TokenPayload{Username: body.Username, Role: role}, validFor)
		if err != nil {
			SendError(response, err, "", "Internal server error")
			return
		}
		SendJSON(response, http.StatusOK, map[string]string{"token": token})
	})

	HandleFunc(mux, "GET /validJWT", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		SendJSON(response, http.StatusOK, true)
	}))
}
