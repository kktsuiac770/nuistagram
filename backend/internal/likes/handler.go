package likes

import (
	"net/http"
	"strconv"

	"nuistagram/internal/config"
	"nuistagram/internal/httputil"
	"nuistagram/internal/monitoring/metrics"
	"nuistagram/internal/ratelimit"
	"nuistagram/internal/repository"
)

// Handler handles like-related HTTP endpoints.
type Handler struct {
	auth          httputil.Authenticator
	likes         repository.LikeRepository
	photos        repository.PhotoRepository
	notifications repository.NotificationRepository
	config        *config.Config
	limits        *ratelimit.Limiter
}

func New(
	auth httputil.Authenticator,
	likes repository.LikeRepository,
	photos repository.PhotoRepository,
	notifications repository.NotificationRepository,
	cfg *config.Config,
	limits *ratelimit.Limiter,
) *Handler {
	return &Handler{
		auth:          auth,
		likes:         likes,
		photos:        photos,
		notifications: notifications,
		config:        cfg,
		limits:        limits,
	}
}

func (h *Handler) APIToggleLike(w http.ResponseWriter, r *http.Request) {
	user := h.auth.CurrentUser(r)
	if user == nil {
		httputil.JSONError(w, 401, "unauthorized")
		return
	}

	limits := h.config.UsageLimits.LimitsForRole(user.Role)
	if err := h.limits.CheckAndIncrement(user.ID, "like", limits.LikesPerHour); err != nil {
		httputil.JSONError(w, http.StatusTooManyRequests, "hourly like limit reached")
		return
	}

	photoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSONError(w, 400, "invalid photo id")
		return
	}

	isLiked, likeCount, err := h.likes.Toggle(photoID, user.ID)
	if err != nil {
		httputil.JSONError(w, 500, "internal error")
		return
	}
	metrics.LikesTotal.Inc()

	if isLiked {
		go func() {
			_, _, photoUserID, err := h.photos.GetOwner(photoID)
			if err == nil && photoUserID != user.ID {
				h.notifications.Create(photoUserID, user.ID, "like", photoID, 0)
			}
		}()
	}

	httputil.WriteJSON(w, 200, map[string]interface{}{
		"success":    true,
		"is_liked":   isLiked,
		"like_count": likeCount,
	})
}

func (h *Handler) APIGetLikers(w http.ResponseWriter, r *http.Request) {
	photoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSONError(w, 400, "invalid photo id")
		return
	}

	limit, _ := strconv.Atoi(r.FormValue("limit"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	likers, err := h.likes.GetLikers(photoID, limit)
	if err != nil {
		httputil.JSONError(w, 500, "internal error")
		return
	}

	users := make([]map[string]interface{}, len(likers))
	for i, u := range likers {
		users[i] = map[string]interface{}{
			"id":       u.ID,
			"username": u.Username,
			"avatar":   u.Avatar,
		}
	}

	httputil.WriteJSON(w, 200, users)
}
