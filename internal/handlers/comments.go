package handlers

import (
	"net/http"
	"strconv"
)

func APIGetComments(w http.ResponseWriter, r *http.Request) {
	photoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteJSON(w, 400, map[string]string{"error": "invalid photo id"})
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	comments, err := Repos.Comments.GetByPhotoID(photoID, limit, offset)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
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

	WriteJSON(w, 200, result)
}

func APICreateComment(w http.ResponseWriter, r *http.Request) {
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

	content := r.FormValue("content")
	if content == "" {
		WriteJSON(w, 400, map[string]string{"error": "content required"})
		return
	}

	if len(content) > 1000 {
		WriteJSON(w, 400, map[string]string{"error": "content too long"})
		return
	}

	commentID, err := Repos.Comments.Create(photoID, user.ID, content)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	photo, _, err := Repos.Photos.GetByID(photoID, user.ID)
	if err == nil && photo.UserID != user.ID {
		Repos.Notifications.Create(photo.UserID, user.ID, "comment", photoID, commentID)
	}

	WriteJSON(w, 200, map[string]interface{}{
		"success":    true,
		"comment_id": commentID,
	})
}

func APIDeleteComment(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		WriteJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	commentID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteJSON(w, 400, map[string]string{"error": "invalid comment id"})
		return
	}

	err = Repos.Comments.Delete(commentID, user.ID)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	WriteJSON(w, 200, map[string]bool{"success": true})
}
