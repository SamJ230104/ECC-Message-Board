package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Username            string `json:"username"`
	Password            string `json:"password"`
	SigningPublicKey    string `json:"signing_public_key"`
	EncryptionPublicKey string `json:"encryption_public_key"`
}

type RegisterResponse struct {
	Message  string `json:"message"`
	Username string `json:"username"`
	UserId   int64  `json:"user_id"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type ValidationError struct {
	Message string
}

func Register(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")

		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: "Invalid JSON format",
			})
			return
		}

		if err := validateRegistration(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: err.Error(),
			})
			return
		}

		passwordHash, err := bcrypt.GenerateFromPassword(
			[]byte(req.Password),
			bcrypt.DefaultCost,
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: "Failed to process password",
			})
			return
		}

		result, err := db.Exec(`
		INSERT INTO users (username, password_hash, signing_public_key, encryption_public_key)
		VALUES (?, ?, ?, ?)
		`, req.Username, string(passwordHash), req.SigningPublicKey, req.EncryptionPublicKey)

		if err != nil {
			if strings.Contains(err.Error(), "Unique constraint failed") {
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(ErrorResponse{
					Error: "Username already taken",
				})
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: "Failed to create user",
			})
			return
		}

		userID, err := result.LastInsertId()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: "User created, but failed to get ID",
			})
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(RegisterResponse{
			Message:  "User created successfully",
			Username: req.Username,
			UserId:   userID,
		})
	}
}

func validateRegistration(req *RegisterRequest) error {

	if req.Username == "" {
		return &ValidationError{"Username is required"}
	}

	if len(req.Username) < 3 {
		return &ValidationError{"Username must be at least 3 characters"}
	}

	if len(req.Username) > 25 {
		return &ValidationError{"Username must be less than 25 characters"}
	}

	if req.Password == "" {
		return &ValidationError{"Password is required"}
	}

	if len(req.Password) < 8 {
		return &ValidationError{"Password must be at least 8 characters"}
	}

	if req.SigningPublicKey == "" {
		return &ValidationError{"Signing public key is required"}
	}

	if req.EncryptionPublicKey == "" {
		return &ValidationError{"Encryption public key is required"}
	}

	return nil
}

func (e *ValidationError) Error() string {
	return e.Message
}
