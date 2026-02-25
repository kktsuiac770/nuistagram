package handlers

import (
	"net/http"
	"strconv"
)

func APIGetNotifications(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		WriteJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	notifications, err := Repos.Notifications.GetByUserID(user.ID, limit, offset)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	result := make([]map[string]interface{}, len(notifications))
	for i, n := range notifications {
		result[i] = map[string]interface{}{
			"id":         n.ID,
			"type":       n.Type,
			"read":       n.Read,
			"created_at": n.CreatedAt.Format("2006-01-02 15:04:05"),
			"actor": map[string]interface{}{
				"id":       n.Actor.ID,
				"username": n.Actor.Username,
				"avatar":   n.Actor.Avatar,
			},
		}
		if n.PhotoID > 0 {
			result[i]["photo_id"] = n.PhotoID
		}
		if n.CommentID > 0 {
			result[i]["comment_id"] = n.CommentID
		}
	}

	WriteJSON(w, 200, result)
}

func APIUnreadCount(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		WriteJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	count, err := Repos.Notifications.GetUnreadCount(user.ID)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	WriteJSON(w, 200, map[string]int{"count": count})
}

func APIMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		WriteJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	notificationID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteJSON(w, 400, map[string]string{"error": "invalid notification id"})
		return
	}

	err = Repos.Notifications.MarkAsRead(notificationID, user.ID)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	WriteJSON(w, 200, map[string]bool{"success": true})
}

func APIMarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		WriteJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	err := Repos.Notifications.MarkAllAsRead(user.ID)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	WriteJSON(w, 200, map[string]bool{"success": true})
}
