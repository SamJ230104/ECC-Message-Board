package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	middleware "MessageBoard/Middleware"
)

type PostPublicMessageRequest struct {
	Content     string `json:"content"`
	EcSignature string `json:"ec_signature"`
}

type PublicMessageResponse struct {
	Id          int64  `json:"id"`
	UserId      int64  `json:"user_id"`
	Username    string `json:"username"`
	Content     string `json:"content"`
	EcSignature string `json:"ec_signature"`
	CreatedAt   string `json:"creared_at"`
}

func PostPublicMessage(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userId := r.Context().Value(middleware.UserIdKey).(int64)

		var req PostPublicMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON format"})
			return
		}

		if req.Content == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Content is required"})
			return
		}

		if req.EcSignature == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Signature required"})
			return
		}

		result, err := db.Exec(`
			INSERT INTO public_messages (user_id, content, ec_signature)
			VALUES (?, ?, ?)
			`, userId, req.Content, req.EcSignature)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to post message"})
			return
		}

		messageId, err := result.LastInsertId()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Message posted, failed to get ID"})
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]int64{
			"id": messageId,
		})

	}
}

func GetPublicMessages(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		rows, err := db.Query(`
			SELECT pm.id, pm.user_id, u.username, pm.content, pm.ec_signature, pm.created_at
			FROM public_messages pm
			JOIN users u on u.id = pm.user_id
			ORDER BY pm.created_at DESC
		`)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to get messages"})
			return
		}
		defer rows.Close()

		messages := []PublicMessageResponse{}
		for rows.Next() {
			var msg PublicMessageResponse
			if err := rows.Scan(
				&msg.Id,
				&msg.UserId,
				&msg.Username,
				&msg.Content,
				&msg.EcSignature,
				&msg.CreatedAt,
			); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to read messages"})
				return
			}
			messages = append(messages, msg)
		}

		if err := rows.Err(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to read messages"})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(messages)
	}
}
