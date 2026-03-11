package photos

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	exif "github.com/dsoprea/go-exif/v3"

	"nuistagram/internal/config"
	"nuistagram/internal/httputil"
	"nuistagram/internal/monitoring/metrics"
	"nuistagram/internal/ratelimit"
	"nuistagram/internal/repository"
	"nuistagram/internal/storage"
)

// Handler handles photo-related HTTP endpoints.
type Handler struct {
	auth      httputil.Authenticator
	users     repository.UserRepository
	photos    repository.PhotoRepository
	nuis      repository.NuiRepository
	albums    repository.AlbumRepository
	favorites repository.FavoriteRepository
	storage   storage.Storage
	config    *config.Config
	limits    *ratelimit.Limiter
}

func New(
	auth httputil.Authenticator,
	users repository.UserRepository,
	photos repository.PhotoRepository,
	nuis repository.NuiRepository,
	albums repository.AlbumRepository,
	favorites repository.FavoriteRepository,
	stor storage.Storage,
	cfg *config.Config,
	limits *ratelimit.Limiter,
) *Handler {
	return &Handler{
		auth:      auth,
		users:     users,
		photos:    photos,
		nuis:      nuis,
		albums:    albums,
		favorites: favorites,
		storage:   stor,
		config:    cfg,
		limits:    limits,
	}
}

// --- Image processing helpers ---

func extractExifDate(data []byte) *time.Time {
	entries, _, err := exif.GetFlatExifData(data, nil)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if entry.TagName == "DateTimeOriginal" {
			if str, ok := entry.Value.(string); ok {
				t, err := time.Parse("2006:01:02 15:04:05", str)
				if err == nil {
					return &t
				}
			}
		}
	}
	return nil
}

var heicBrands = map[string]bool{
	"heic": true, "heis": true, "hevc": true, "hevx": true,
	"heim": true, "heix": true, "hevm": true, "hevs": true,
	"mif1": true, "msf1": true,
}

func detectMIME(data []byte) (string, error) {
	m := http.DetectContentType(data)
	switch m {
	case "image/jpeg", "image/png", "image/webp":
		return m, nil
	}
	if len(data) >= 12 && string(data[4:8]) == "ftyp" && heicBrands[string(data[8:12])] {
		return "", fmt.Errorf("HEIC/HEIF is not supported — please convert to JPEG, PNG, or WebP first")
	}
	return "", fmt.Errorf("unsupported format — only JPEG, PNG, and WebP are allowed")
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func generateThumbnail(src image.Image, maxSize int) ([]byte, error) {
	thumb := imaging.Thumbnail(src, maxSize, maxSize, imaging.Lanczos)
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, thumb, &jpeg.Options{Quality: 85})
	return buf.Bytes(), err
}

func compressImage(src image.Image, maxWidth int) ([]byte, error) {
	bounds := src.Bounds()
	if bounds.Dx() > maxWidth {
		src = imaging.Resize(src, maxWidth, 0, imaging.Lanczos)
	}
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: 85})
	return buf.Bytes(), err
}

func parseNuiNames(input string) []string {
	var names []string
	for _, name := range strings.Split(input, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// --- Response types ---

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

// --- Handlers ---

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	user := h.auth.CurrentUser(r)
	if user == nil {
		httputil.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	limits := h.config.UsageLimits.LimitsForRole(user.Role)
	if err := h.limits.CheckAndIncrement(user.ID, "upload", limits.UploadsPerDay); err != nil {
		httputil.JSONError(w, http.StatusTooManyRequests, "daily upload limit reached")
		return
	}

	r.ParseMultipartForm(100 << 20)

	files := r.MultipartForm.File["photos"]
	if len(files) == 0 {
		httputil.JSONError(w, http.StatusBadRequest, "No files uploaded")
		return
	}

	nuiNamesInput := r.FormValue("nui_names")
	description := r.FormValue("description")
	takenAtStr := r.FormValue("taken_at")
	albumID := r.FormValue("album_id")

	nuiNames := parseNuiNames(nuiNamesInput)
	if len(nuiNames) == 0 {
		httputil.JSONError(w, http.StatusBadRequest, "At least one nui name required")
		return
	}

	// Pre-scan: validate all files before any DB writes.
	validated := make([][]byte, 0, len(files))
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			httputil.JSONError(w, http.StatusBadRequest, "failed to read uploaded file")
			return
		}
		fileData, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			httputil.JSONError(w, http.StatusBadRequest, "failed to read uploaded file")
			return
		}
		if len(fileData) > 5*1024*1024 {
			httputil.JSONError(w, http.StatusBadRequest, fmt.Sprintf("%s exceeds the 5 MB limit", fileHeader.Filename))
			return
		}
		if _, err := detectMIME(fileData); err != nil {
			httputil.JSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		validated = append(validated, fileData)
	}

	for _, fileData := range validated {
		baseFilename := strconv.FormatInt(time.Now().UnixNano(), 36) + "_" + strconv.FormatInt(user.ID, 10)

		img, imgErr := imaging.Decode(bytes.NewReader(fileData), imaging.AutoOrientation(true))
		var photoData []byte
		if imgErr != nil {
			httputil.JSONError(w, http.StatusBadRequest, "invalid or corrupted image file")
			return
		}
		b := img.Bounds()
		if absInt(b.Dx()-b.Dy()) > 2 {
			httputil.JSONError(w, http.StatusBadRequest, "image must be square (1:1 aspect ratio)")
			return
		}

		var err error
		photoData, err = compressImage(img, 1920)
		if err != nil {
			photoData = fileData
		}

		photoResult, err := h.storage.Upload(r.Context(), photoData, baseFilename+".jpg")
		if err != nil {
			continue
		}
		filename := photoResult.URL

		var takenAt *time.Time
		if takenAtStr != "" {
			t, err := time.Parse("2006-01-02", takenAtStr)
			if err == nil {
				takenAt = &t
			}
		}
		if takenAt == nil {
			if exifTime := extractExifDate(fileData); exifTime != nil {
				takenAt = exifTime
			}
		}

		var thumbnailFilename string
		thumbData, err := generateThumbnail(img, 400)
		if err == nil {
			thumbResult, err := h.storage.Upload(r.Context(), thumbData, "thumb_"+baseFilename+".jpg")
			if err == nil {
				thumbnailFilename = thumbResult.URL
			}
		}

		photoID, err := h.photos.Create(filename, thumbnailFilename, user.ID, description, takenAt)
		if err != nil {
			continue
		}
		metrics.PhotosUploadedTotal.Inc()

		var nuiIDs []int64
		for _, nuiName := range nuiNames {
			nuiID, err := h.nuis.GetOrCreate(nuiName, user.ID)
			if err != nil {
				continue
			}
			nuiIDs = append(nuiIDs, nuiID)
		}
		h.photos.SetNuis(photoID, nuiIDs)

		if albumID != "" {
			albumIDInt, _ := strconv.ParseInt(albumID, 10, 64)
			h.albums.AddPhoto(albumIDInt, photoID)
		}
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	user := h.auth.CurrentUser(r)
	if user == nil {
		httputil.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	photoID := r.PathValue("id")
	if photoID == "" {
		httputil.JSONError(w, http.StatusBadRequest, "Photo ID required")
		return
	}

	filename, thumbnail, ownerID, err := h.photos.GetOwner(httputil.ParseID(photoID))
	if err == sql.ErrNoRows {
		httputil.JSONError(w, http.StatusNotFound, "Photo not found")
		return
	}
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "Internal error")
		return
	}

	if ownerID != user.ID {
		httputil.JSONError(w, http.StatusForbidden, "Unauthorized")
		return
	}

	if err := h.photos.Delete(httputil.ParseID(photoID)); err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "Failed to delete photo")
		return
	}

	h.storage.Delete(r.Context(), filename)
	if thumbnail != "" {
		h.storage.Delete(r.Context(), thumbnail)
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) ToggleFavorite(w http.ResponseWriter, r *http.Request) {
	user := h.auth.CurrentUser(r)
	if user == nil {
		httputil.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	photoID := r.PathValue("id")
	if photoID == "" {
		httputil.JSONError(w, http.StatusBadRequest, "Photo ID required")
		return
	}

	isFav, err := h.favorites.Toggle(httputil.ParseID(photoID), user.ID)
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "Failed to toggle favorite")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": true, "is_favorite": isFav})
}

func (h *Handler) ExportPhotos(w http.ResponseWriter, r *http.Request) {
	user := h.auth.CurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	limits := h.config.UsageLimits.LimitsForRole(user.Role)
	if err := h.limits.CheckAndIncrement(user.ID, "export", limits.ExportsPerDay); err != nil {
		httputil.JSONError(w, http.StatusTooManyRequests, "daily export limit reached")
		return
	}

	filenames, err := h.photos.GetFilenamesByUser(user.ID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=nuistagram_backup_%s.zip", time.Now().Format("2006-01-02")))

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	for _, filename := range filenames {
		data, err := h.storage.FetchContent(r.Context(), filename)
		if err != nil {
			continue
		}

		zipName := filename
		if idx := strings.LastIndex(filename, "/"); idx != -1 {
			zipName = filename[idx+1:]
		}

		header := &zip.FileHeader{
			Name:     zipName,
			Method:   zip.Deflate,
			Modified: time.Now(),
		}
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			continue
		}
		writer.Write(data)
	}
}

func (h *Handler) APIGetPhotos(w http.ResponseWriter, r *http.Request) {
	user := h.auth.CurrentUser(r)
	page, _ := strconv.Atoi(r.FormValue("page"))
	if page < 1 {
		page = 1
	}

	feed := r.FormValue("feed")
	userID := httputil.CurrentUserID(user)
	tags := parseTags(r.FormValue("tags"))

	var result *PaginationResponse
	if feed == "following" && user != nil {
		res, err := h.photos.GetFollowingFeed(user.ID, page)
		if err != nil {
			httputil.JSONError(w, 500, "internal error")
			return
		}
		result = convertToResponse(res)
	} else if len(tags) > 0 {
		res, err := h.photos.Search(tags, "or", page, userID)
		if err != nil {
			httputil.JSONError(w, 500, "internal error")
			return
		}
		result = convertToResponse(res)
	} else {
		res, err := h.photos.GetAll(page, userID)
		if err != nil {
			httputil.JSONError(w, 500, "internal error")
			return
		}
		result = convertToResponse(res)
	}

	httputil.WriteJSON(w, 200, result)
}

func (h *Handler) APIGetPhoto(w http.ResponseWriter, r *http.Request) {
	photoID := r.PathValue("id")
	user := h.auth.CurrentUser(r)

	p, username, err := h.photos.GetByID(httputil.ParseID(photoID), httputil.CurrentUserID(user))
	if err != nil {
		httputil.JSONError(w, 404, "not found")
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

	httputil.WriteJSON(w, 200, response)
}

func (h *Handler) APIToggleFavorite(w http.ResponseWriter, r *http.Request) {
	user := h.auth.CurrentUser(r)
	if user == nil {
		httputil.JSONError(w, 401, "unauthorized")
		return
	}

	photoID := r.PathValue("id")
	isFav, err := h.favorites.Toggle(httputil.ParseID(photoID), user.ID)
	if err != nil {
		httputil.JSONError(w, 500, "internal error")
		return
	}
	httputil.WriteJSON(w, 200, map[string]interface{}{"success": true, "is_favorite": isFav})
}

func (h *Handler) APIGetUserPhotos(w http.ResponseWriter, r *http.Request) {
	currentUser := h.auth.CurrentUser(r)
	username := r.PathValue("username")
	page, _ := strconv.Atoi(r.FormValue("page"))
	if page < 1 {
		page = 1
	}

	profileUser, err := h.users.GetByUsername(username)
	if err != nil {
		httputil.JSONError(w, 404, "user not found")
		return
	}

	result, err := h.photos.GetByUser(profileUser.ID, page, httputil.CurrentUserID(currentUser))
	if err != nil {
		httputil.JSONError(w, 500, "internal error")
		return
	}

	httputil.WriteJSON(w, 200, convertToResponse(result))
}

func (h *Handler) APIGetNuis(w http.ResponseWriter, r *http.Request) {
	nuis, err := h.nuis.GetAll()
	if err != nil {
		httputil.JSONError(w, 500, "internal error")
		return
	}
	httputil.WriteJSON(w, 200, nuis)
}
