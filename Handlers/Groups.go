package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	middleware "MessageBoard/Middleware"
)

type CreateGroupRequest struct {
	GroupName     string            `json:"group_name"`
	EncryptedKeys map[string]string `json:"encrypted_keys"`
}

type GroupResponse struct {
	Id        int64  `json:"id"`
	GroupName string `json:"group_name"`
	CreatedBy int64  `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

type GroupMemberResponse struct {
	UserId            int64  `json:"user_id"`
	Username          string `json:"username"`
	EncryptedGroupKey string `json:"encrypted_group_key"`
	JoinedAt          string `json:"joined_at"`
}

type AddMemberRequest struct {
	Username          string `json:"username"`
	EncryptedGroupKey string `json:"encrypted_group_key"`
}

func CreateGroup(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userId := r.Context().Value(middleware.UserIdKey).(int64)

		var req CreateGroupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON format"})
			return
		}

		if strings.TrimSpace(req.GroupName) == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Group name is required"})
			return
		}
		if len(req.EncryptedKeys) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "At least one member is required"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to create group"})
			return
		}
		defer tx.Rollback()

		result, err := tx.Exec(`
			INSERT INTO groups (group_name, created_by)
			VALLUES (?, ?)
			`, req.GroupName, userId)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to create group"})
			return
		}

		groupId, err := result.LastInsertId()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to get group ID"})
			return
		}

		for username, encryptedKey := range req.EncryptedKeys {
			var memberId int64
			err := tx.QueryRow(`
				SELECT id FROM users WHERE username = ?
			`, username).Scan(&memberId)

			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(ErrorResponse{Error: "User not found: " + username})
				return
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to find user: " + username})
				return
			}

			_, err = tx.Exec(`
			    INSERT INTO group_members (group_id, user_id, encrypted_group_key)
				VALUES (?, ?, ?)
				`, groupId, memberId, encryptedKey)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to add member: " + username})
				return
			}

		}

		var creatorUsername string
		err = tx.QueryRow(`SELECT username FROM users WHERE id = ?`, userId).Scan(&creatorUsername)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to get creator info"})
			return
		}

		if _, exists := req.EncryptedKeys[creatorUsername]; !exists {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Created must be included in encrypted_keys"})
			tx.Rollback()
			return
		}

		if err := tx.Commit(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to finalise group creation"})
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]int64{
			"id": groupId,
		})
	}
}

func GetGroups(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userId := r.Context().Value(middleware.UserIdKey).(int64)

		rows, err := db.Query(`
		    SELECT g.id, g.group_name, g.created_by, g.created_at
			FROM groups g
			JOIN group_members gm ON gm.group_id = g.id
			WHERE gm.user_id = ?
			ORDER BY g.created_at DESC
		`, userId)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to get groups"})
			return
		}
		defer rows.Close()

		groups := []GroupResponse{}
		for rows.Next() {
			var g GroupResponse
			if err := rows.Scan(&g.Id, &g.GroupName, &g.CreatedBy, &g.CreatedAt); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to read groups"})
				return
			}
			groups = append(groups, g)
		}

		if err := rows.Err(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to read groups"})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(groups)

	}
}

func AddGroupMember(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userId := r.Context().Value(middleware.UserIdKey).(int64)

		groupId := r.PathValue("id")
		if groupId == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Group ID is reqjuired"})
			return
		}

		var req AddMemberRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON format"})
			return
		}

		if req.Username == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Username is required"})
			return
		}
		if req.EncryptedGroupKey == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Enrypted group key is required"})
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

		var newMemberId int64
		err = db.QueryRow(`
		    SELECT id FROM users WHERE username = ?
			`, req.Username).Scan(&newMemberId)

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

		_, err = db.Exec(`
		    INSERT INTO group_members (group_id, user_id, encrypted_group_key)
			VALUES (?, ?, ?)
		`, groupId, newMemberId, req.EncryptedGroupKey)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(ErrorResponse{Error: "User is already a member"})
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to add new member"})
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Member added successfully",
		})

	}
}

func RemoveGroupMember(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		requesterId := r.Context().Value(middleware.UserIdKey).(int64)

		groupId := r.PathValue("id")
		targetUserId := r.PathValue("userId")

		if groupId == "" || targetUserId == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Group ID and user ID are required"})
			return
		}

		var createdBy int64
		err := db.QueryRow(`
		    SELECT created_by FROM groups WHERE id = ?
		`, groupId).Scan(&createdBy)

		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Group not found"})
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to find group"})
			return
		}

		if createdBy != requesterId {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Only the group creator can remove members"})
			return
		}

		if targetUserId == fmt.Sprintf("%d", requesterId) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Group creator cannot be removed"})
			return
		}

		result, err := db.Exec(`
		    DELETE FROM group_members WHERE group_id = ? AND user_id = ?
		`, groupId, targetUserId)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to remove member"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Member not found in group"})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Member removed successfully",
		})
	}
}

func GetGroupMembers(db *sql.DB) http.HandlerFunc {
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
		    SELECT u.id, u.username, gm.encrypted_group_key, gm.joined_at
			FROM group_members gm
			JOIN users u ON u.id = gm.user_id
			WHERE gm.group_id = ?
			ORDER BY gm.joined_at ASC
		`, groupId)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to get members"})
			return
		}
		defer rows.Close()

		members := []GroupMemberResponse{}
		for rows.Next() {
			var m GroupMemberResponse
			if err := rows.Scan(&m.UserId, &m.Username, &m.EncryptedGroupKey, &m.JoinedAt); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to read members"})
				return
			}
			members = append(members, m)
		}

		if err := rows.Err(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to read members"})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(members)

	}
}
