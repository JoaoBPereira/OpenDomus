package routes

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const userContextKey contextKey = "user"

// Middleware checks the JWT of the request (Authorization: Bearer <token> header, or the `token`
// cookie) and attaches its payload to the request context (see User)
func (auth *Auth) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		token := ""
		if header := request.Header.Get("Authorization"); header != "" {
			if _, bearer, found := strings.Cut(header, " "); found {
				token = bearer
			}
		}
		if token == "" {
			if cookie, err := request.Cookie("token"); err == nil {
				token = cookie.Value
			}
		}
		if token == "" {
			SendMessage(response, http.StatusUnauthorized, "No token, authorization denied")
			return
		}

		payload, err := auth.VerifyToken(token)
		if err != nil {
			SendMessage(response, http.StatusUnauthorized, "Invalid token")
			return
		}
		next(response, request.WithContext(context.WithValue(request.Context(), userContextKey, payload)))
	}
}

// User returns the token payload of an authenticated request, or nil
func User(request *http.Request) *TokenPayload {
	user, _ := request.Context().Value(userContextKey).(*TokenPayload)
	return user
}
