package follows

import (
	"net/http"

	"nuistagram/internal/httputil"
	"nuistagram/internal/monitoring/metrics"
	"nuistagram/internal/repository"
)

// Handler handles follow-related HTTP endpoints.
type Handler struct {
	auth          httputil.Authenticator
	users         repository.UserRepository
	follows       repository.FollowRepository
	notifications repository.NotificationRepository
}

func New(
	auth httputil.Authenticator,
	users repository.UserRepository,
	follows repository.FollowRepository,
	notifications repository.NotificationRepository,
) *Handler {
	return &Handler{
		auth:          auth,
		users:         users,
		follows:       follows,
		notifications: notifications,
	}
}

func (h *Handler) APIFollowByUsername(w http.ResponseWriter, r *http.Request) {
	user := h.auth.CurrentUser(r)
	if user == nil {
		httputil.JSONError(w, 401, "unauthorized")
		return
	}

	username := r.PathValue("username")
	targetUser, err := h.users.GetByUsername(username)
	if err != nil {
		httputil.JSONError(w, 404, "user not found")
		return
	}

	if targetUser.ID == user.ID {
		httputil.JSONError(w, 400, "cannot follow yourself")
		return
	}

	err = h.follows.Follow(user.ID, targetUser.ID)
	if err != nil {
		httputil.JSONError(w, 500, "internal error")
		return
	}
	metrics.FollowsTotal.Inc()

	h.notifications.Create(targetUser.ID, user.ID, "follow", 0, 0)

	httputil.WriteJSON(w, 200, map[string]bool{"success": true})
}

func (h *Handler) APIUnfollowByUsername(w http.ResponseWriter, r *http.Request) {
	user := h.auth.CurrentUser(r)
	if user == nil {
		httputil.JSONError(w, 401, "unauthorized")
		return
	}

	username := r.PathValue("username")
	targetUser, err := h.users.GetByUsername(username)
	if err != nil {
		httputil.JSONError(w, 404, "user not found")
		return
	}

	err = h.follows.Unfollow(user.ID, targetUser.ID)
	if err != nil {
		httputil.JSONError(w, 500, "internal error")
		return
	}

	httputil.WriteJSON(w, 200, map[string]bool{"success": true})
}

func (h *Handler) APIGetFollowStatus(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	currentUser := h.auth.CurrentUser(r)

	if currentUser == nil {
		httputil.JSONError(w, 401, "unauthorized")
		return
	}

	profileUser, err := h.users.GetByUsername(username)
	if err != nil {
		httputil.JSONError(w, 404, "user not found")
		return
	}

	isFollowing := h.follows.IsFollowing(currentUser.ID, profileUser.ID)
	httputil.WriteJSON(w, 200, map[string]bool{"is_following": isFollowing})
}
