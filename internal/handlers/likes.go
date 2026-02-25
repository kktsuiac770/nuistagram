package handlers

import (
	"net/http"
	"strconv"
)

func APIToggleLike(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		WriteJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	photoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteJSON(w, 400, map[string]string{"error": "invalid photo id"})
		return
	}

	isLiked, err := Repos.Likes.Toggle(photoID, user.ID)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	if isLiked {
		photo, _, err := Repos.Photos.GetByID(photoID, user.ID)
		if err == nil && photo.UserID != user.ID {
			Repos.Notifications.Create(photo.UserID, user.ID, "like", photoID, 0)
		}
	}

	likeCount, _ := Repos.Likes.GetLikeCount(photoID)

	WriteJSON(w, 200, map[string]interface{}{
		"success":    true,
		"is_liked":   isLiked,
		"like_count": likeCount,
	})
}

func APIGetLikers(w http.ResponseWriter, r *http.Request) {
	photoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteJSON(w, 400, map[string]string{"error": "invalid photo id"})
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	likers, err := Repos.Likes.GetLikers(photoID, limit)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
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

	WriteJSON(w, 200, users)
}
