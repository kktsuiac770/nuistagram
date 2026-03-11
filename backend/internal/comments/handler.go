package comments

import (
	"net/http"
	"strconv"

	"nuistagram/internal/config"
	"nuistagram/internal/httputil"
	"nuistagram/internal/monitoring/metrics"
	"nuistagram/internal/ratelimit"
	"nuistagram/internal/repository"
)

// Handler handles comment-related HTTP endpoints.
type Handler struct {
	auth          httputil.Authenticator
	comments      repository.CommentRepository
	photos        repository.PhotoRepository
	notifications repository.NotificationRepository
	config        *config.Config
	limits        *ratelimit.Limiter
}

func New(
	auth httputil.Authenticator,
	comments repository.CommentRepository,
	photos repository.PhotoRepository,
	notifications repository.NotificationRepository,
	cfg *config.Config,
	limits *ratelimit.Limiter,
) *Handler {
	return &Handler{
		auth:          auth,
		comments:      comments,
		photos:        photos,
		notifications: notifications,
		config:        cfg,
		limits:        limits,
	}
}

func (h *Handler) APIGetComments(w http.ResponseWriter, r *http.Request) {
	photoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSONError(w, 400, "invalid photo id")
		return
	}

	limit, _ := strconv.Atoi(r.FormValue("limit"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset, _ := strconv.Atoi(r.FormValue("offset"))

	comments, err := h.comments.GetByPhotoID(photoID, limit, offset)
	if err != nil {
		httputil.JSONError(w, 500, "internal error")
		return
	}

	result := make([]map[string]interface{}, len(comments))
	for i, c := range comments {
		result[i] = map[string]interface{}{
			"id":         c.ID,
			"photo_id":   c.PhotoID,
			"user_id":    c.UserID,
			"content":    c.Content,
			"created_at": c.CreatedAt.Format("2006-01-02 15:04:05"),
			"username":   c.Username,
		}
	}

	httputil.WriteJSON(w, 200, result)
}

func (h *Handler) APICreateComment(w http.ResponseWriter, r *http.Request) {
	user := h.auth.CurrentUser(r)
	if user == nil {
		httputil.JSONError(w, 401, "unauthorized")
		return
	}

	limits := h.config.UsageLimits.LimitsForRole(user.Role)
	if err := h.limits.CheckAndIncrement(user.ID, "comment", limits.CommentsPerHour); err != nil {
		httputil.JSONError(w, http.StatusTooManyRequests, "hourly comment limit reached")
		return
	}

	photoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSONError(w, 400, "invalid photo id")
		return
	}

	content := r.FormValue("content")
	if content == "" {
		httputil.JSONError(w, 400, "content required")
		return
	}

	if len(content) > 1000 {
		httputil.JSONError(w, 400, "content too long")
		return
	}

	commentID, err := h.comments.Create(photoID, user.ID, content)
	if err != nil {
		httputil.JSONError(w, 500, "internal error")
		return
	}
	metrics.CommentsTotal.Inc()

	go func() {
		_, _, photoUserID, err := h.photos.GetOwner(photoID)
		if err == nil && photoUserID != user.ID {
			h.notifications.Create(photoUserID, user.ID, "comment", photoID, commentID)
		}
	}()

	httputil.WriteJSON(w, 200, map[string]interface{}{
		"success":    true,
		"comment_id": commentID,
	})
}

func (h *Handler) APIDeleteComment(w http.ResponseWriter, r *http.Request) {
	user := h.auth.CurrentUser(r)
	if user == nil {
		httputil.JSONError(w, 401, "unauthorized")
		return
	}

	commentID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httputil.JSONError(w, 400, "invalid comment id")
		return
	}

	err = h.comments.Delete(commentID, user.ID)
	if err != nil {
		httputil.JSONError(w, 500, "internal error")
		return
	}

	httputil.WriteJSON(w, 200, map[string]bool{"success": true})
}
