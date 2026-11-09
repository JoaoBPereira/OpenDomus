package rest

import (
	"log/slog"
	"net/http"
	"slices"

	"od/internal/rest/routes"
)

// Middleware wraps a handler
type Middleware func(next http.Handler) http.Handler

// Recover logs a panic in a handler and answers 500 instead of stopping the process
func Recover() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					slog.Error("unhandled panic in handler", "path", request.URL.Path, "panic", recovered)
					routes.SendMessage(response, http.StatusInternalServerError, "Internal server error")
				}
			}()
			next.ServeHTTP(response, request)
		})
	}
}

// Helmet sets the security headers the helmet package sets by default
func Helmet() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			headers := response.Header()
			headers.Set("Content-Security-Policy", "default-src 'self';base-uri 'self';font-src 'self' https: data:;form-action 'self';frame-ancestors 'self';img-src 'self' data:;object-src 'none';script-src 'self';script-src-attr 'none';style-src 'self' https: 'unsafe-inline';upgrade-insecure-requests")
			headers.Set("Cross-Origin-Opener-Policy", "same-origin")
			headers.Set("Cross-Origin-Resource-Policy", "same-origin")
			headers.Set("Origin-Agent-Cluster", "?1")
			headers.Set("Referrer-Policy", "no-referrer")
			headers.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			headers.Set("X-Content-Type-Options", "nosniff")
			headers.Set("X-DNS-Prefetch-Control", "off")
			headers.Set("X-Download-Options", "noopen")
			headers.Set("X-Frame-Options", "SAMEORIGIN")
			headers.Set("X-Permitted-Cross-Domain-Policies", "none")
			headers.Set("X-XSS-Protection", "0")
			next.ServeHTTP(response, request)
		})
	}
}

// BodyLimit rejects request bodies larger than maxBytes
func BodyLimit(maxBytes int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if request.Body != nil {
				request.Body = http.MaxBytesReader(response, request.Body, maxBytes)
			}
			next.ServeHTTP(response, request)
		})
	}
}

// CORS allows the given origins; an empty list allows every origin
func CORS(allowedOrigins []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			origin := request.Header.Get("Origin")
			if len(allowedOrigins) == 0 {
				response.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" && slices.Contains(allowedOrigins, origin) {
				response.Header().Set("Access-Control-Allow-Origin", origin)
				response.Header().Add("Vary", "Origin")
			}

			if request.Method == http.MethodOptions { // preflight
				response.Header().Set("Access-Control-Allow-Methods", "GET,HEAD,PUT,PATCH,POST,DELETE")
				if requestHeaders := request.Header.Get("Access-Control-Request-Headers"); requestHeaders != "" {
					response.Header().Set("Access-Control-Allow-Headers", requestHeaders)
					response.Header().Add("Vary", "Access-Control-Request-Headers")
				}
				response.Header().Set("Content-Length", "0")
				response.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(response, request)
		})
	}
}
