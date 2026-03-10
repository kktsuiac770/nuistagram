package server

import (
	"net/http"
	"strconv"

	"nuistagram/internal/monitoring/metrics"
)

func (s *Server) APIGetComments(w http.ResponseWriter, r *http.Request) {
	photoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, 400, "invalid photo id")
		return
	}

	limit, _ := strconv.Atoi(r.FormValue("limit"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset, _ := strconv.Atoi(r.FormValue("offset"))

	comments, err := s.Repos.Comments.GetByPhotoID(photoID, limit, offset)
	if err != nil {
		jsonError(w, 500, "internal error")
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

	writeJSON(w, 200, result)
}

func (s *Server) APICreateComment(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		jsonError(w, 401, "unauthorized")
		return
	}

	limits := s.Config.UsageLimits.LimitsForRole(user.Role)
	if err := s.Limits.CheckAndIncrement(user.ID, "comment", limits.CommentsPerHour); err != nil {
		jsonError(w, http.StatusTooManyRequests, "hourly comment limit reached")
		return
	}

	photoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, 400, "invalid photo id")
		return
	}

	content := r.FormValue("content")
	if content == "" {
		jsonError(w, 400, "content required")
		return
	}

	if len(content) > 1000 {
		jsonError(w, 400, "content too long")
		return
	}

	commentID, err := s.Repos.Comments.Create(photoID, user.ID, content)
	if err != nil {
		jsonError(w, 500, "internal error")
		return
	}
	metrics.CommentsTotal.Inc()

	go func() {
		_, _, photoUserID, err := s.Repos.Photos.GetOwner(photoID)
		if err == nil && photoUserID != user.ID {
			s.Repos.Notifications.Create(photoUserID, user.ID, "comment", photoID, commentID)
		}
	}()

	writeJSON(w, 200, map[string]interface{}{
		"success":    true,
		"comment_id": commentID,
	})
}

func (s *Server) APIDeleteComment(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		jsonError(w, 401, "unauthorized")
		return
	}

	commentID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, 400, "invalid comment id")
		return
	}

	err = s.Repos.Comments.Delete(commentID, user.ID)
	if err != nil {
		jsonError(w, 500, "internal error")
		return
	}

	writeJSON(w, 200, map[string]bool{"success": true})
}
