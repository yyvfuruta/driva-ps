package main

import (
	"net/http"
	"os"
	"strings"
)

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secretToken := os.Getenv("AUTH_TOKEN")
		if secretToken == "" {
			errorResponse(w, http.StatusInternalServerError, "AUTH_TOKEN not set")
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			errorResponse(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			errorResponse(w, http.StatusUnauthorized, "Invalid Authorization header format")
			return
		}

		if parts[1] != strings.TrimSpace(secretToken) {
			errorResponse(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		next(w, r)
	})
}
