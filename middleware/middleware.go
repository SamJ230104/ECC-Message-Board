package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const UserIdKey contextKey = "user_id"

type errorResponse struct {
	Error string `json:"error"`
}

func Auth(db *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(errorResponse{Error: "Missing authorisation header"})
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(errorResponse{Error: "Invalid authorisation format"})
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")

		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(errorResponse{Error: "Missing token"})
			return
		}

		var userId int64
		var expiresAt string
		err := db.QueryRow(`
			SELECT user_id, expires_at FROM user_sessions WHERE id = ?
		`, token).Scan(&userId, &expiresAt)

		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(errorResponse{Error: "Invalid or expired token"})
			return
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errorResponse{Error: "Failed to validate token"})
			return
		}

		expiry, err := time.Parse(time.RFC3339, expiresAt)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errorResponse{Error: "Failed to parse token expiry"})
			return
		}

		if time.Now().UTC().After(expiry) {
			db.Exec(`DELETE FROM user_sessions WHERE id = ?`, token)

			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(errorResponse{Error: "Invalid or expired token"})
			return
		}

		ctx := context.WithValue(r.Context(), UserIdKey, userId)
		next.ServeHTTP(w, r.WithContext(ctx))

	}
}
