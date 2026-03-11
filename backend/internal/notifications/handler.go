package notifications

import (
	"net/http"
	"strconv"

	"nuistagram/internal/httputil"
	"nuistagram/internal/repository"
)

// Handler handles notification-related HTTP endpoints.
type Handler struct {
	auth          httputil.Authenticator
	notifications repository.NotificationRepository
}

func New(auth httputil.Authenticator, notifications repository.NotificationRepository) *Handler {
	return &Handler{auth: auth, notifications: notifications}
}

func (h *Handler) APIGetNotifications(w http.ResponseWriter, r *http.Request) {
	user := h.auth.CurrentUser(r)
	if user == nil {
		httputil.JSONError(w, 401, "unauthorized")
		return
	}

	limit, _ := strconv.Atoi(r.FormValue("limit"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset, _ := strconv.Atoi(r.FormValue("offset"))

	notifications, err := h.notifications.GetByUserID(user.ID, limit, offset)
	if err != nil {
		httputil.JSONError(w, 500, "internal error")
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

	httputil.WriteJSON(w, 200, result)
}

func (h *Handler) APIUnreadCount(w http.ResponseWriter, r *http.Request) {
	user := h.auth.CurrentUser(r)
	if user == nil {
		httputil.JSONError(w, 401, "unauthorized")
		return
	}

	count, err := h.notifications.GetUnreadCount(user.ID)
	if err != nil {
		httputil.JSONError(w, 500, "internal error")
		return
	}

	httputil.WriteJSON(w, 200, map[string]int{"count": count})
}

func (h *Handler) APIMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	user := h.auth.CurrentUser(r)
	if user == nil {
		httputil.JSONError(w, 401, "unauthorized")
		return
	}

	notificationID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSONError(w, 400, "invalid notification id")
		return
	}

	err = h.notifications.MarkAsRead(notificationID, user.ID)
	if err != nil {
		httputil.JSONError(w, 500, "internal error")
		return
	}

	httputil.WriteJSON(w, 200, map[string]bool{"success": true})
}

func (h *Handler) APIMarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	user := h.auth.CurrentUser(r)
	if user == nil {
		httputil.JSONError(w, 401, "unauthorized")
		return
	}

	err := h.notifications.MarkAllAsRead(user.ID)
	if err != nil {
		httputil.JSONError(w, 500, "internal error")
		return
	}

	httputil.WriteJSON(w, 200, map[string]bool{"success": true})
}
