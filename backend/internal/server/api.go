package server

import (
	"net/http"
	"nuistagram/internal/repository"
	"strconv"
	"strings"
)

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

func parseTags(param string) []string {
	if param == "" {
		return nil
	}
	var tags []string
	for _, tag := range strings.Split(param, ",") {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

func (s *Server) APIGetPhotos(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	page, _ := strconv.Atoi(r.FormValue("page"))
	if page < 1 {
		page = 1
	}

	feed := r.FormValue("feed")
	userID := currentUserID(user)
	tags := parseTags(r.FormValue("tags"))

	var result *PaginationResponse
	if feed == "following" && user != nil {
		res, err := s.Repos.Photos.GetFollowingFeed(user.ID, page)
		if err != nil {
			jsonError(w, 500, "internal error")
			return
		}
		result = convertToResponse(res)
	} else if len(tags) > 0 {
		res, err := s.Repos.Photos.Search(tags, "or", page, userID)
		if err != nil {
			jsonError(w, 500, "internal error")
			return
		}
		result = convertToResponse(res)
	} else {
		res, err := s.Repos.Photos.GetAll(page, userID)
		if err != nil {
			jsonError(w, 500, "internal error")
			return
		}
		result = convertToResponse(res)
	}

	writeJSON(w, 200, result)
}

func (s *Server) APIGetPhoto(w http.ResponseWriter, r *http.Request) {
	photoID := r.PathValue("id")
	user := s.currentUser(r)

	p, username, err := s.Repos.Photos.GetByID(parseID(photoID), currentUserID(user))
	if err != nil {
		jsonError(w, 404, "not found")
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

	writeJSON(w, 200, response)
}

func (s *Server) APIGetMe(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		jsonError(w, 401, "unauthorized")
		return
	}

	fullUser, err := s.Repos.Users.GetByIDWithCounts(user.ID, user.ID)
	if err != nil {
		jsonError(w, 500, "internal error")
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"id":              user.ID,
		"username":        user.Username,
		"bio":             fullUser.Bio,
		"avatar":          fullUser.Avatar,
		"photo_count":     fullUser.PhotoCount,
		"follower_count":  fullUser.FollowerCount,
		"following_count": fullUser.FollowingCount,
	})
}

func (s *Server) APIGetNuis(w http.ResponseWriter, r *http.Request) {
	nuis, err := s.Repos.Nuis.GetAll()
	if err != nil {
		jsonError(w, 500, "internal error")
		return
	}
	writeJSON(w, 200, nuis)
}

func (s *Server) APIToggleFavorite(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		jsonError(w, 401, "unauthorized")
		return
	}

	photoID := r.PathValue("id")
	isFav, err := s.Repos.Favorites.Toggle(parseID(photoID), user.ID)
	if err != nil {
		jsonError(w, 500, "internal error")
		return
	}
	writeJSON(w, 200, map[string]interface{}{"success": true, "is_favorite": isFav})
}

func (s *Server) APIGetUser(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	currentUser := s.currentUser(r)

	user, err := s.Repos.Users.GetByUsernameWithCounts(username, currentUserID(currentUser))
	if err != nil {
		jsonError(w, 404, "user not found")
		return
	}

	writeJSON(w, 200, map[string]interface{}{
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

func (s *Server) APIGetUserPhotos(w http.ResponseWriter, r *http.Request) {
	currentUser := s.currentUser(r)
	username := r.PathValue("username")
	page, _ := strconv.Atoi(r.FormValue("page"))
	if page < 1 {
		page = 1
	}

	profileUser, err := s.Repos.Users.GetByUsername(username)
	if err != nil {
		jsonError(w, 404, "user not found")
		return
	}

	result, err := s.Repos.Photos.GetByUser(profileUser.ID, page, currentUserID(currentUser))
	if err != nil {
		jsonError(w, 500, "internal error")
		return
	}

	writeJSON(w, 200, convertToResponse(result))
}

func (s *Server) APIGetFollowStatus(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	currentUser := s.currentUser(r)

	if currentUser == nil {
		jsonError(w, 401, "unauthorized")
		return
	}

	profileUser, err := s.Repos.Users.GetByUsername(username)
	if err != nil {
		jsonError(w, 404, "user not found")
		return
	}

	isFollowing := s.Repos.Follows.IsFollowing(currentUser.ID, profileUser.ID)
	writeJSON(w, 200, map[string]bool{"is_following": isFollowing})
}
