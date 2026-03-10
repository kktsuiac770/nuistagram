package server

import (
	"net/http"
	"strconv"
)

func (s *Server) APIGetNotifications(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		jsonError(w, 401, "unauthorized")
		return
	}

	limit, _ := strconv.Atoi(r.FormValue("limit"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset, _ := strconv.Atoi(r.FormValue("offset"))

	notifications, err := s.Repos.Notifications.GetByUserID(user.ID, limit, offset)
	if err != nil {
		jsonError(w, 500, "internal error")
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

	writeJSON(w, 200, result)
}

func (s *Server) APIUnreadCount(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		jsonError(w, 401, "unauthorized")
		return
	}

	count, err := s.Repos.Notifications.GetUnreadCount(user.ID)
	if err != nil {
		jsonError(w, 500, "internal error")
		return
	}

	writeJSON(w, 200, map[string]int{"count": count})
}

func (s *Server) APIMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		jsonError(w, 401, "unauthorized")
		return
	}

	notificationID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, 400, "invalid notification id")
		return
	}

	err = s.Repos.Notifications.MarkAsRead(notificationID, user.ID)
	if err != nil {
		jsonError(w, 500, "internal error")
		return
	}

	writeJSON(w, 200, map[string]bool{"success": true})
}

func (s *Server) APIMarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		jsonError(w, 401, "unauthorized")
		return
	}

	err := s.Repos.Notifications.MarkAllAsRead(user.ID)
	if err != nil {
		jsonError(w, 500, "internal error")
		return
	}

	writeJSON(w, 200, map[string]bool{"success": true})
}
