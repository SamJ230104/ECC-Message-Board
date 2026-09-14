package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	middleware "MessageBoard/Middleware"
)

type PostGroupMessageRequest struct {
	EncryptedContent string `json:"encrypted_content"`
	Nonce            string `json:"nonce"`
	EcSignature      string `json:"ec_signature"`
}

type GroupMessageResponse struct {
	Id               int64  `json:"id"`
	GroupId          int64  `json:"group_id"`
	UserId           int64  `json:"user_id"`
	Username         string `json:"username"`
	EncryptedContent string `json:"encrypted_content"`
	Nonce            string `json:"nonce"`
	EcSignature      string `json:"ec_signature"`
	CreatedAt        string `json:"created_at"`
}

func PostGroupMessage(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userId := r.Context().Value(middleware.UserIdKey).(int64)

		groupId := r.PathValue("id")
		if groupId == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Group ID is required"})
			return
		}

		var req PostGroupMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON format"})
			return
		}

		if req.EncryptedContent == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Encrypted Content is required"})
			return
		}

		if req.Nonce == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Nonce is required"})
			return
		}

		if req.EcSignature == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Signature is required"})
			return
		}

		var exists int
		err := db.QueryRow(`
			SELECT COUNT (*) FROM group_members WHERE group_id = ? AND user_id = ?
			`, groupId, userId).Scan(&exists)
		if err != nil || exists == 0 {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "You are not a member of this group"})
			return
		}

		result, err := db.Exec(`
			INSERT INTO group_messages (group_id, user_id, encrypted_content, nonce, ec_signature)
			VALUES (?, ?, ?, ?, ?)
		`, groupId, userId, req.EncryptedContent, req.Nonce, req.EcSignature)
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

func GetGroupMessages(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userId := r.Context().Value(middleware.UserIdKey).(int64)

		groupId := r.PathValue("id")
		if groupId == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Group ID is required"})
			return
		}

		var exists int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM group_members WHERE group_id = ? AND user_id = ?
			`, groupId, userId).Scan(&exists)
		if err != nil || exists == 0 {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "You are not a member of this group"})
			return
		}

		rows, err := db.Query(`
			SELECT gm.id, gm.group_id, gm.user_id, u.username,
				gm.encrypted_content, gm.nonce, gm.ec_signature, gm.created_at
			FROM group_messages gm
			JOIN users u ON u.id = gm.user_id
			WHERE gm.group_id = ?
			ORDER BY gm.created_at ASC
		`, groupId)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to get messages"})
			return
		}
		defer rows.Close()

		messages := []GroupMessageResponse{}
		for rows.Next() {
			var msg GroupMessageResponse
			if err := rows.Scan(
				&msg.Id,
				&msg.GroupId,
				&msg.UserId,
				&msg.Username,
				&msg.EncryptedContent,
				&msg.Nonce,
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
