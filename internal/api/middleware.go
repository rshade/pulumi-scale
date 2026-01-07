package api

import (
	"net/http"
	"strings"
)

// AuthMiddleware enforces Bearer Token authentication.
func AuthMiddleware(expectedToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Unauthorized: Missing Authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Unauthorized: Invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			token := parts[1]
			if token != expectedToken {
				http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
