package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	middleware "MessageBoard/Middleware"
)

type SendPrivateMessageRequest struct {
	ToUsername       string `json:"to_username"`
	EncryptedContent string `json:"encrypted_content"`
	Nonce            string `json:"nonce"`
	EcSignature      string `json:"ec_signature"`
}

type PrivateMessageResponse struct {
	Id               int64  `json:"id"`
	FromUserId       int64  `json:"from_user_id"`
	FromUsername     string `json:"from_username"`
	ToUserId         int64  `json:"to_user_id"`
	ToUsername       string `json:"to_username"`
	EncryptedContent string `json:"encrypted_content"`
	Nonce            string `json:"nonce"`
	EcSignature      string `json:"ec_signature"`
	CreatedAt        string `json:"created_at"`
	Read             int    `json:"read"`
}

func SendPrivateMessage(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		fromUserId := r.Context().Value(middleware.UserIdKey).(int64)

		var req SendPrivateMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON format"})
			return
		}

		if req.ToUsername == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Recipient Username required"})
			return
		}

		if req.EncryptedContent == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Encrypted Content Required"})
			return
		}

		if req.Nonce == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Nonce required"})
			return
		}

		if req.EcSignature == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Signature required"})
			return
		}

		var toUserId int64
		err := db.QueryRow(`
			SELECT id FROM users WHERE username = ?
			`, req.ToUsername).Scan(&toUserId)

		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Recipient not found"})
			return
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to find recipient"})
			return
		}

		if fromUserId == toUserId {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Cannot send a message to yourself"})
			return
		}

		result, err := db.Exec(`
		    INSERT INTO private_message (from_user_id, to_user_id, encrypted_content, nonce, ec_signature)
			VALUES (?, ?, ?, ?, ?)`, fromUserId, toUserId, req.EncryptedContent, req.Nonce, req.EcSignature)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to send message"})
			return
		}

		messageId, err := result.LastInsertId()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Message sent but failed to get id"})
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]int64{
			"id": messageId,
		})
	}
}

func GetInbox(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userId := r.Context().Value(middleware.UserIdKey).(int64)

		rows, err := db.Query(`
		    SELECT
			    pm.id,
				pm.from_user_id,
				sender.username
				pm.to_user_id,
				recipient.username,
				pm.ecrypted_content
				pm.nonce,
				pm.ec_signature,
				pm.created_at,
				pm.read
			FROM private_messages pm
			JOIN users sender ON sender.id = pm.from_user_id
			JOIN users recipient ON recipient.id = pm.to_user_id
			WHERE pm.to_user_id = ?
			ORDER BY pm.created_at DESC
		`, userId)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to get inbox"})
			return
		}
		defer rows.Close()

		messages := []PrivateMessageResponse{}
		for rows.Next() {
			var msg PrivateMessageResponse
			if err := rows.Scan(
				&msg.Id,
				&msg.FromUserId,
				&msg.FromUsername,
				&msg.ToUserId,
				&msg.ToUsername,
				&msg.EncryptedContent,
				&msg.Nonce,
				&msg.EcSignature,
				&msg.CreatedAt,
				&msg.Read,
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

func GetConversation(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userId := r.Context().Value(middleware.UserIdKey).(int64)

		username := r.PathValue("username")
		if username == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Username required"})
			return
		}

		var otherUserId int64
		err := db.QueryRow(`
		    SELECT id FROM users WHERE username = ?
		`, username).Scan(&otherUserId)

		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "User not found"})
			return
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to find user"})
			return
		}

		rows, err := db.Query(`
		    SELECT
			    pm.id,
				pm.from_user_id,
				sender.username,
				pm.to_user_id,
				recipient.username,
				pm.encrypted_content,
				pm.nonce,
				pm.ec_signature,
				pm.created_at,
				pm.read
			FROM private_messages pm
			JOIN users sender ON sender.id = pm.from_user_id
			JOIN users recipient on recipient_id = pm.to_user_id
			WHERE (pm.from_user_id = ? AND pm.to_user_id = ?)
			OR (pm.from_user_id = ? AND pm.to_user_id = ?)
			ORDER BY pm.created_at ASC
		`, userId, otherUserId, otherUserId, userId)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to get conversation"})
			return
		}
		defer rows.Close()

		messages := []PrivateMessageResponse{}
		for rows.Next() {
			var msg PrivateMessageResponse
			if err := rows.Scan(
				&msg.Id,
				&msg.FromUserId,
				&msg.FromUsername,
				&msg.ToUserId,
				&msg.ToUsername,
				&msg.EncryptedContent,
				&msg.Nonce,
				&msg.EcSignature,
				&msg.CreatedAt,
				&msg.Read,
			); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to read conversation"})
				return
			}
			messages = append(messages, msg)
		}

		if err := rows.Err(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to read conversation"})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(messages)
	}
}

func MarkMessageRead(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userId := r.Context().Value(middleware.UserIdKey).(int64)

		messageId := r.PathValue("id")
		if messageId == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Message ID required"})
			return
		}

		result, err := db.Exec(`
		    UPDATE private_messages
			SET read = 1
			WHERE id = ? AND to_user_id = ?
		`, messageId, userId)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to mark message as read"})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to confirm update"})
			return
		}

		if rowsAffected == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Message not found"})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "message marked as read",
		})
	}
}
