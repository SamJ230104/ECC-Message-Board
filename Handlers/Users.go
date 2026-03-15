package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type UserResponse struct {
	UserId              int64  `json:"user_id"`
	Username            string `json:"username"`
	SigningPublicKey    string `json:"signing_public_key"`
	EncryptionPublicKey string `json:"encryption_public_key"`
}

func GetUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		username := r.PathValue("username")
		if username == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Username is required"})
			return
		}

		var user UserResponse
		err := db.QueryRow(`
			SELECT id, username, signing_public_key, encryption_public_key
			FROM users
			WHERE username = ?
			`, username).Scan(
			&user.UserId,
			&user.Username,
			&user.SigningPublicKey,
			&user.EncryptionPublicKey,
		)

		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "User not found"})
			return
		}

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to get user"})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)

	}
}
