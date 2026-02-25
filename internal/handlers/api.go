package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

type PhotoResponse struct {
	ID          int64    `json:"id"`
	Filename    string   `json:"filename"`
	Thumbnail   string   `json:"thumbnail"`
	UserID      int64    `json:"user_id"`
	Description string   `json:"description"`
	TakenAt     string   `json:"taken_at"`
	CreatedAt   string   `json:"created_at"`
	IsFavorite  bool     `json:"is_favorite"`
	NuiNames    []string `json:"nui_names"`
}

type PaginationResponse struct {
	Photos      []PhotoResponse `json:"photos"`
	CurrentPage int             `json:"current_page"`
	TotalPages  int             `json:"total_pages"`
	TotalCount  int             `json:"total_count"`
	HasPrev     bool            `json:"has_prev"`
	HasNext     bool            `json:"has_next"`
	Pages       []int           `json:"pages"`
}

func APIGetPhotos(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	var userID int64
	if user != nil {
		userID = user.ID
	}

	result, err := Repos.Photos.GetAll(page, userID)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	photos := make([]PhotoResponse, len(result.Photos))
	for i, p := range result.Photos {
		takenAt := ""
		if !p.TakenAt.IsZero() {
			takenAt = p.TakenAt.Format("2006-01-02")
		}
		photos[i] = PhotoResponse{
			ID:          p.ID,
			Filename:    p.Filename,
			Thumbnail:   p.Thumbnail,
			UserID:      p.UserID,
			Description: p.Description,
			TakenAt:     takenAt,
			CreatedAt:   p.CreatedAt.Format("2006-01-02 15:04:05"),
			IsFavorite:  p.IsFavorite,
			NuiNames:    p.NuiNames,
		}
	}

	WriteJSON(w, 200, PaginationResponse{
		Photos:      photos,
		CurrentPage: result.CurrentPage,
		TotalPages:  result.TotalPages,
		TotalCount:  result.TotalCount,
		HasPrev:     result.HasPrev,
		HasNext:     result.HasNext,
		Pages:       result.Pages,
	})
}

func APIGetPhoto(w http.ResponseWriter, r *http.Request) {
	photoID := r.PathValue("id")
	user := GetCurrentUser(r)

	var currentUserID int64
	if user != nil {
		currentUserID = user.ID
	}

	p, username, err := Repos.Photos.GetByID(parseID(photoID), currentUserID)
	if err != nil {
		WriteJSON(w, 404, map[string]string{"error": "not found"})
		return
	}

	response := struct {
		ID          int64    `json:"id"`
		Filename    string   `json:"filename"`
		Thumbnail   string   `json:"thumbnail"`
		UserID      int64    `json:"user_id"`
		Description string   `json:"description"`
		TakenAt     string   `json:"taken_at"`
		CreatedAt   string   `json:"created_at"`
		IsFavorite  bool     `json:"is_favorite"`
		NuiNames    []string `json:"nui_names"`
		Username    string   `json:"username"`
	}{
		ID:          p.ID,
		Filename:    p.Filename,
		Thumbnail:   p.Thumbnail,
		UserID:      p.UserID,
		Description: p.Description,
		TakenAt:     p.TakenAt.Format("2006-01-02"),
		CreatedAt:   p.CreatedAt.Format("2006-01-02 15:04:05"),
		IsFavorite:  p.IsFavorite,
		NuiNames:    p.NuiNames,
		Username:    username,
	}

	WriteJSON(w, 200, response)
}

func APIGetMe(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		WriteJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	WriteJSON(w, 200, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
	})
}

func APIGetNuis(w http.ResponseWriter, r *http.Request) {
	nuis, err := Repos.Nuis.GetAll()
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	WriteJSON(w, 200, nuis)
}

func APIToggleFavorite(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		WriteJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	photoID := r.PathValue("id")
	isFav, err := Repos.Favorites.Toggle(parseID(photoID), user.ID)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	WriteJSON(w, 200, map[string]bool{"is_favorite": isFav})
}
