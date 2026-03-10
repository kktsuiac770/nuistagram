package server

import (
	"net/http"
	"strconv"
)

func (s *Server) APIToggleLike(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		jsonError(w, 401, "unauthorized")
		return
	}

	photoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, 400, "invalid photo id")
		return
	}

	isLiked, err := s.Repos.Likes.Toggle(photoID, user.ID)
	if err != nil {
		jsonError(w, 500, "internal error")
		return
	}

	if isLiked {
		_, _, photoUserID, err := s.Repos.Photos.GetOwner(photoID)
		if err == nil && photoUserID != user.ID {
			s.Repos.Notifications.Create(photoUserID, user.ID, "like", photoID, 0)
		}
	}

	likeCount, _ := s.Repos.Likes.GetLikeCount(photoID)

	writeJSON(w, 200, map[string]interface{}{
		"success":    true,
		"is_liked":   isLiked,
		"like_count": likeCount,
	})
}

func (s *Server) APIGetLikers(w http.ResponseWriter, r *http.Request) {
	photoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, 400, "invalid photo id")
		return
	}

	limit, _ := strconv.Atoi(r.FormValue("limit"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	likers, err := s.Repos.Likes.GetLikers(photoID, limit)
	if err != nil {
		jsonError(w, 500, "internal error")
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

	writeJSON(w, 200, users)
}
