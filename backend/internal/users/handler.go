package users

import (
	"bytes"
	"image/jpeg"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/disintegration/imaging"

	"nuistagram/internal/config"
	"nuistagram/internal/httputil"
	"nuistagram/internal/ratelimit"
	"nuistagram/internal/repository"
	"nuistagram/internal/storage"
)

// Handler handles user-related HTTP endpoints.
type Handler struct {
	auth    httputil.Authenticator
	users   repository.UserRepository
	photos  repository.PhotoRepository
	storage storage.Storage
	config  *config.Config
	limits  *ratelimit.Limiter
}

func New(
	auth httputil.Authenticator,
	users repository.UserRepository,
	photos repository.PhotoRepository,
	stor storage.Storage,
	cfg *config.Config,
	limits *ratelimit.Limiter,
) *Handler {
	return &Handler{
		auth:    auth,
		users:   users,
		photos:  photos,
		storage: stor,
		config:  cfg,
		limits:  limits,
	}
}

func (h *Handler) APIGetMe(w http.ResponseWriter, r *http.Request) {
	user := h.auth.CurrentUser(r)
	if user == nil {
		httputil.JSONError(w, 401, "unauthorized")
		return
	}

	fullUser, err := h.users.GetByIDWithCounts(user.ID, user.ID)
	if err != nil {
		httputil.JSONError(w, 500, "internal error")
		return
	}

	httputil.WriteJSON(w, 200, map[string]interface{}{
		"id":              user.ID,
		"username":        user.Username,
		"bio":             fullUser.Bio,
		"avatar":          fullUser.Avatar,
		"photo_count":     fullUser.PhotoCount,
		"follower_count":  fullUser.FollowerCount,
		"following_count": fullUser.FollowingCount,
	})
}

func (h *Handler) APIGetUser(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	currentUser := h.auth.CurrentUser(r)

	user, err := h.users.GetByUsernameWithCounts(username, httputil.CurrentUserID(currentUser))
	if err != nil {
		httputil.JSONError(w, 404, "user not found")
		return
	}

	httputil.WriteJSON(w, 200, map[string]interface{}{
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

func (h *Handler) APISearchUsers(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("q")
	if query == "" {
		httputil.WriteJSON(w, 200, []interface{}{})
		return
	}

	users, err := h.users.Search(query, 20)
	if err != nil {
		httputil.JSONError(w, 500, "internal error")
		return
	}

	result := make([]map[string]interface{}, len(users))
	for i, u := range users {
		result[i] = map[string]interface{}{
			"id":          u.ID,
			"username":    u.Username,
			"avatar":      u.Avatar,
			"photo_count": u.PhotoCount,
		}
	}

	httputil.WriteJSON(w, 200, result)
}

func (h *Handler) APIUpdateProfile(w http.ResponseWriter, r *http.Request) {
	user := h.auth.CurrentUser(r)
	if user == nil {
		httputil.JSONError(w, 401, "unauthorized")
		return
	}

	bio := r.FormValue("bio")
	if len(bio) > 500 {
		httputil.JSONError(w, 400, "bio too long")
		return
	}

	err := h.users.UpdateProfile(user.ID, bio)
	if err != nil {
		httputil.JSONError(w, 500, "internal error")
		return
	}

	httputil.WriteJSON(w, 200, map[string]bool{"success": true})
}

func (h *Handler) APIUploadAvatar(w http.ResponseWriter, r *http.Request) {
	user := h.auth.CurrentUser(r)
	if user == nil {
		httputil.JSONError(w, 401, "unauthorized")
		return
	}

	limits := h.config.UsageLimits.LimitsForRole(user.Role)
	if err := h.limits.CheckAndIncrement(user.ID, "avatar", limits.AvatarsPerDay); err != nil {
		httputil.JSONError(w, http.StatusTooManyRequests, "daily avatar upload limit reached")
		return
	}

	err := r.ParseMultipartForm(5 << 20)
	if err != nil {
		httputil.JSONError(w, 400, "file too large")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		httputil.JSONError(w, 400, "no file uploaded")
		return
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		httputil.JSONError(w, 500, "failed to read file")
		return
	}

	baseFilename := strconv.FormatInt(time.Now().UnixNano(), 36) + "_" + strconv.FormatInt(user.ID, 10)

	var avatarData []byte
	var uploadName string

	img, err := imaging.Decode(bytes.NewReader(fileData), imaging.AutoOrientation(true))
	if err == nil {
		img = imaging.Fill(img, 200, 200, imaging.Center, imaging.Lanczos)
		var buf bytes.Buffer
		jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
		avatarData = buf.Bytes()
		uploadName = "avatars/" + baseFilename + ".jpg"
	} else {
		avatarData = fileData
		ext := filepath.Ext(header.Filename)
		if ext == "" {
			ext = ".jpg"
		}
		uploadName = "avatars/" + baseFilename + ext
	}

	result, err := h.storage.Upload(r.Context(), avatarData, uploadName)
	if err != nil {
		httputil.JSONError(w, 500, "failed to upload avatar")
		return
	}

	err = h.users.UpdateAvatar(user.ID, result.URL)
	if err != nil {
		httputil.JSONError(w, 500, "internal error")
		return
	}

	httputil.WriteJSON(w, 200, map[string]interface{}{
		"success": true,
		"avatar":  result.URL,
	})
}
