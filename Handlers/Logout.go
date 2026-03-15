package handlers

import (
	middleware "MessageBoard/Middleware"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
)

func Logout(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

		userId := r.Context().Value(middleware.UserIdKey).(int64)

		_, err := db.Exec(`
			DELETE FROM user_sessions WHERE id = ? AND user_id = ?
		`, token, userId)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to logout"})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Logged out successfully",
		})
	}

}
