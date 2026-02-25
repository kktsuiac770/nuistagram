package handlers

import (
	"encoding/json"
	"net/http"
	"nuistagram/internal/repository"
	"strconv"
	"strings"
)

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

type PhotoResponse struct {
	ID           int64    `json:"id"`
	Filename     string   `json:"filename"`
	Thumbnail    string   `json:"thumbnail"`
	UserID       int64    `json:"user_id"`
	Description  string   `json:"description"`
	TakenAt      string   `json:"taken_at"`
	CreatedAt    string   `json:"created_at"`
	IsFavorite   bool     `json:"is_favorite"`
	NuiNames     []string `json:"nui_names"`
	Username     string   `json:"username"`
	LikeCount    int      `json:"like_count"`
	IsLiked      bool     `json:"is_liked"`
	CommentCount int      `json:"comment_count"`
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

	feed := r.URL.Query().Get("feed")

	var userID int64
	if user != nil {
		userID = user.ID
	}

	var tags []string
	if tagsParam := r.URL.Query().Get("tags"); tagsParam != "" {
		for _, tag := range strings.Split(tagsParam, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	var result *PaginationResponse
	if feed == "following" && user != nil {
		res, err := Repos.Photos.GetFollowingFeed(user.ID, page)
		if err != nil {
			WriteJSON(w, 500, map[string]string{"error": "internal error"})
			return
		}
		result = convertToResponse(res)
	} else if len(tags) > 0 {
		res, err := Repos.Photos.Search(tags, "or", page, userID)
		if err != nil {
			WriteJSON(w, 500, map[string]string{"error": "internal error"})
			return
		}
		result = convertToResponse(res)
	} else {
		res, err := Repos.Photos.GetAll(page, userID)
		if err != nil {
			WriteJSON(w, 500, map[string]string{"error": "internal error"})
			return
		}
		result = convertToResponse(res)
	}

	WriteJSON(w, 200, result)
}

func convertToResponse(result *repository.PaginationResult) *PaginationResponse {
	photos := make([]PhotoResponse, len(result.Photos))
	for i, p := range result.Photos {
		takenAt := ""
		if !p.TakenAt.IsZero() {
			takenAt = p.TakenAt.Format("2006-01-02")
		}
		photos[i] = PhotoResponse{
			ID:           p.ID,
			Filename:     p.Filename,
			Thumbnail:    p.Thumbnail,
			UserID:       p.UserID,
			Description:  p.Description,
			TakenAt:      takenAt,
			CreatedAt:    p.CreatedAt.Format("2006-01-02 15:04:05"),
			IsFavorite:   p.IsFavorite,
			NuiNames:     p.NuiNames,
			Username:     p.Username,
			LikeCount:    p.LikeCount,
			IsLiked:      p.IsLiked,
			CommentCount: p.CommentCount,
		}
	}

	return &PaginationResponse{
		Photos:      photos,
		CurrentPage: result.CurrentPage,
		TotalPages:  result.TotalPages,
		TotalCount:  result.TotalCount,
		HasPrev:     result.HasPrev,
		HasNext:     result.HasNext,
		Pages:       result.Pages,
	}
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

	response := PhotoResponse{
		ID:           p.ID,
		Filename:     p.Filename,
		Thumbnail:    p.Thumbnail,
		UserID:       p.UserID,
		Description:  p.Description,
		TakenAt:      p.TakenAt.Format("2006-01-02"),
		CreatedAt:    p.CreatedAt.Format("2006-01-02 15:04:05"),
		IsFavorite:   p.IsFavorite,
		NuiNames:     p.NuiNames,
		Username:     username,
		LikeCount:    p.LikeCount,
		IsLiked:      p.IsLiked,
		CommentCount: p.CommentCount,
	}

	WriteJSON(w, 200, response)
}

func APIGetMe(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		WriteJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	fullUser, err := Repos.Users.GetByIDWithCounts(user.ID, user.ID)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	WriteJSON(w, 200, map[string]interface{}{
		"id":              user.ID,
		"username":        user.Username,
		"bio":             fullUser.Bio,
		"avatar":          fullUser.Avatar,
		"photo_count":     fullUser.PhotoCount,
		"follower_count":  fullUser.FollowerCount,
		"following_count": fullUser.FollowingCount,
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
	WriteJSON(w, 200, map[string]interface{}{"success": true, "is_favorite": isFav})
}

func APIGetUser(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	currentUser := GetCurrentUser(r)

	var currentUserID int64
	if currentUser != nil {
		currentUserID = currentUser.ID
	}

	user, err := Repos.Users.GetByUsernameWithCounts(username, currentUserID)
	if err != nil {
		WriteJSON(w, 404, map[string]string{"error": "user not found"})
		return
	}

	WriteJSON(w, 200, map[string]interface{}{
		"id":              user.ID,
		"username":        user.Username,
		"bio":             user.Bio,
		"avatar":          user.Avatar,
		"photo_count":     user.PhotoCount,
		"follower_count":  user.FollowerCount,
		"following_count": user.FollowingCount,
		"is_following":    user.IsFollowing,
	})
}

func APIGetUserPhotos(w http.ResponseWriter, r *http.Request) {
	currentUser := GetCurrentUser(r)
	username := r.PathValue("username")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	profileUser, err := Repos.Users.GetByUsername(username)
	if err != nil {
		WriteJSON(w, 404, map[string]string{"error": "user not found"})
		return
	}

	var currentUserID int64
	if currentUser != nil {
		currentUserID = currentUser.ID
	}

	result, err := Repos.Photos.GetByUser(profileUser.ID, page, currentUserID)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	WriteJSON(w, 200, convertToResponse(result))
}
