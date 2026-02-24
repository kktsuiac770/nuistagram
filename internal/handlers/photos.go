package handlers

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"nuistagram/internal/database"
	"nuistagram/internal/models"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	exif "github.com/dsoprea/go-exif/v3"
)

const PhotosPerPage = 20
const MaxImageWidth = 1920
const ThumbnailSize = 400

type PaginationResult struct {
	Photos      []models.PhotoWithNuis
	CurrentPage int
	TotalPages  int
	TotalCount  int
	HasPrev     bool
	HasNext     bool
	Pages       []int
}

func GetAllPhotos(page int, userID int64) (*PaginationResult, error) {
	return getPhotosWithQuery(nil, "", page, userID)
}

func GetPhotosByNui(nuiName string, page int, userID int64) (*PaginationResult, error) {
	return getPhotosWithQuery([]string{nuiName}, "or", page, userID)
}

func GetPhotosByUser(userID int64, page int, currentUserID int64) (*PaginationResult, error) {
	return getPhotosWithQuery(nil, "", page, currentUserID, userID)
}

func GetFavoritePhotos(userID int64, page int) (*PaginationResult, error) {
	return getFavoritePhotosQuery(userID, page)
}

func SearchPhotos(tags []string, mode string, page int, userID int64) (*PaginationResult, error) {
	if len(tags) == 0 {
		return GetAllPhotos(page, userID)
	}
	return getPhotosWithQuery(tags, mode, page, userID)
}

func countPhotos(tags []string, mode string, filterUserID int64) (int, error) {
	var query string
	var args []interface{}

	if filterUserID > 0 {
		query = `SELECT COUNT(DISTINCT p.id) FROM photos p WHERE p.user_id = ?`
		args = append(args, filterUserID)
	} else if len(tags) == 0 {
		query = `SELECT COUNT(DISTINCT p.id) FROM photos p`
	} else if mode == "and" {
		placeholders := strings.Repeat("?,", len(tags))
		placeholders = placeholders[:len(placeholders)-1]
		query = fmt.Sprintf(`
			SELECT COUNT(DISTINCT p.id)
			FROM photos p
			JOIN photo_nuis pn ON p.id = pn.photo_id
			JOIN nuis n ON pn.nui_id = n.id
			WHERE n.name IN (%s)
			GROUP BY p.id
			HAVING COUNT(DISTINCT n.id) = ?
		`, placeholders)
		for _, tag := range tags {
			args = append(args, tag)
		}
		args = append(args, len(tags))

		var count int
		rows, err := database.DB.Query(query, args...)
		if err != nil {
			return 0, err
		}
		defer rows.Close()
		for rows.Next() {
			count++
		}
		return count, nil
	} else {
		placeholders := strings.Repeat("?,", len(tags))
		placeholders = placeholders[:len(placeholders)-1]
		query = fmt.Sprintf(`
			SELECT COUNT(DISTINCT p.id)
			FROM photos p
			JOIN photo_nuis pn ON p.id = pn.photo_id
			JOIN nuis n ON pn.nui_id = n.id
			WHERE n.name IN (%s)
		`, placeholders)
		for _, tag := range tags {
			args = append(args, tag)
		}
	}

	var count int
	err := database.DB.QueryRow(query, args...).Scan(&count)
	return count, err
}

func getPhotosWithQuery(tags []string, mode string, page int, currentUserID int64, filterUserID ...int64) (*PaginationResult, error) {
	if page < 1 {
		page = 1
	}

	var fUserID int64
	if len(filterUserID) > 0 {
		fUserID = filterUserID[0]
	}

	totalCount, err := countPhotos(tags, mode, fUserID)
	if err != nil {
		return nil, err
	}

	totalPages := (totalCount + PhotosPerPage - 1) / PhotosPerPage
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * PhotosPerPage

	var query string
	var args []interface{}

	if fUserID > 0 {
		query = `
			SELECT DISTINCT p.id, p.filename, COALESCE(p.thumbnail, ''), p.user_id, p.description, p.taken_at, p.created_at
			FROM photos p
			WHERE p.user_id = ?
			ORDER BY p.created_at DESC
			LIMIT ? OFFSET ?
		`
		args = []interface{}{fUserID, PhotosPerPage, offset}
	} else if len(tags) == 0 {
		query = `
			SELECT DISTINCT p.id, p.filename, COALESCE(p.thumbnail, ''), p.user_id, p.description, p.taken_at, p.created_at
			FROM photos p
			ORDER BY p.created_at DESC
			LIMIT ? OFFSET ?
		`
		args = []interface{}{PhotosPerPage, offset}
	} else if mode == "and" {
		placeholders := strings.Repeat("?,", len(tags))
		placeholders = placeholders[:len(placeholders)-1]
		query = fmt.Sprintf(`
			SELECT p.id, p.filename, COALESCE(p.thumbnail, ''), p.user_id, p.description, p.taken_at, p.created_at
			FROM photos p
			JOIN photo_nuis pn ON p.id = pn.photo_id
			JOIN nuis n ON pn.nui_id = n.id
			WHERE n.name IN (%s)
			GROUP BY p.id
			HAVING COUNT(DISTINCT n.id) = ?
			ORDER BY p.created_at DESC
			LIMIT ? OFFSET ?
		`, placeholders)
		for _, tag := range tags {
			args = append(args, tag)
		}
		args = append(args, len(tags), PhotosPerPage, offset)
	} else {
		placeholders := strings.Repeat("?,", len(tags))
		placeholders = placeholders[:len(placeholders)-1]
		query = fmt.Sprintf(`
			SELECT DISTINCT p.id, p.filename, COALESCE(p.thumbnail, ''), p.user_id, p.description, p.taken_at, p.created_at
			FROM photos p
			JOIN photo_nuis pn ON p.id = pn.photo_id
			JOIN nuis n ON pn.nui_id = n.id
			WHERE n.name IN (%s)
			ORDER BY p.created_at DESC
			LIMIT ? OFFSET ?
		`, placeholders)
		for _, tag := range tags {
			args = append(args, tag)
		}
		args = append(args, PhotosPerPage, offset)
	}

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []models.PhotoWithNuis
	for rows.Next() {
		var p models.PhotoWithNuis
		var takenAt sql.NullTime
		var description sql.NullString
		err := rows.Scan(
			&p.ID, &p.Filename, &p.Thumbnail, &p.UserID, &description,
			&takenAt, &p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if takenAt.Valid {
			p.TakenAt = takenAt.Time
		}
		if description.Valid {
			p.Description = description.String
		}

		p.NuiNames, _ = getNuiNamesForPhoto(p.ID)
		if currentUserID > 0 {
			p.IsFavorite = isPhotoFavorited(p.ID, currentUserID)
		}
		photos = append(photos, p)
	}

	pages := calculatePaginationPages(page, totalPages)

	return &PaginationResult{
		Photos:      photos,
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalCount:  totalCount,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
		Pages:       pages,
	}, nil
}

func getFavoritePhotosQuery(userID int64, page int) (*PaginationResult, error) {
	if page < 1 {
		page = 1
	}

	var totalCount int
	err := database.DB.QueryRow(`SELECT COUNT(*) FROM favorites WHERE user_id = ?`, userID).Scan(&totalCount)
	if err != nil {
		return nil, err
	}

	totalPages := (totalCount + PhotosPerPage - 1) / PhotosPerPage
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * PhotosPerPage

	query := `
		SELECT p.id, p.filename, COALESCE(p.thumbnail, ''), p.user_id, p.description, p.taken_at, p.created_at
		FROM photos p
		JOIN favorites f ON p.id = f.photo_id
		WHERE f.user_id = ?
		ORDER BY f.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := database.DB.Query(query, userID, PhotosPerPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []models.PhotoWithNuis
	for rows.Next() {
		var p models.PhotoWithNuis
		var takenAt sql.NullTime
		var description sql.NullString
		err := rows.Scan(
			&p.ID, &p.Filename, &p.Thumbnail, &p.UserID, &description,
			&takenAt, &p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if takenAt.Valid {
			p.TakenAt = takenAt.Time
		}
		if description.Valid {
			p.Description = description.String
		}
		p.IsFavorite = true
		p.NuiNames, _ = getNuiNamesForPhoto(p.ID)
		photos = append(photos, p)
	}

	pages := calculatePaginationPages(page, totalPages)

	return &PaginationResult{
		Photos:      photos,
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalCount:  totalCount,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
		Pages:       pages,
	}, nil
}

func calculatePaginationPages(current, total int) []int {
	if total <= 7 {
		pages := make([]int, total)
		for i := 0; i < total; i++ {
			pages[i] = i + 1
		}
		return pages
	}

	var pages []int
	start := current - 2
	end := current + 2

	if start < 1 {
		end += 1 - start
		start = 1
	}
	if end > total {
		start -= end - total
		end = total
	}

	if start < 1 {
		start = 1
	}

	for i := start; i <= end; i++ {
		pages = append(pages, i)
	}

	return pages
}

func getNuiNamesForPhoto(photoID int64) ([]string, error) {
	rows, err := database.DB.Query(`
		SELECT n.name FROM nuis n
		JOIN photo_nuis pn ON n.id = pn.nui_id
		WHERE pn.photo_id = ?
		ORDER BY n.name
	`, photoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
}

func GetAllNuis() ([]models.Nui, error) {
	rows, err := database.DB.Query(`SELECT id, name, user_id, created_at FROM nuis ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nuis []models.Nui
	for rows.Next() {
		var n models.Nui
		err := rows.Scan(&n.ID, &n.Name, &n.UserID, &n.CreatedAt)
		if err != nil {
			return nil, err
		}
		nuis = append(nuis, n)
	}
	return nuis, nil
}

func GetUserByID(userID int64) (*models.User, error) {
	var u models.User
	err := database.DB.QueryRow(`
		SELECT u.id, u.username, u.password_hash, u.created_at, 
			(SELECT COUNT(*) FROM photos WHERE user_id = u.id) as photo_count
		FROM users u WHERE u.id = ?
	`, userID).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.PhotoCount)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func GetUserByUsername(username string) (*models.User, error) {
	var u models.User
	err := database.DB.QueryRow(`
		SELECT u.id, u.username, u.password_hash, u.created_at,
			(SELECT COUNT(*) FROM photos WHERE user_id = u.id) as photo_count
		FROM users u WHERE u.username = ?
	`, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.PhotoCount)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func GetAllUsers() ([]models.User, error) {
	rows, err := database.DB.Query(`
		SELECT u.id, u.username, u.password_hash, u.created_at,
			(SELECT COUNT(*) FROM photos WHERE user_id = u.id) as photo_count
		FROM users u ORDER BY u.username
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.PhotoCount)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func NuiProfile(w http.ResponseWriter, r *http.Request) {
	nuiName := r.PathValue("name")
	user := GetCurrentUser(r)

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	var userID int64
	if user != nil {
		userID = user.ID
	}

	result, err := GetPhotosByNui(nuiName, page, userID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "nui.html", map[string]interface{}{
		"User":       user,
		"Photos":     result.Photos,
		"NuiName":    nuiName,
		"Pagination": result,
	})
}

func UserProfile(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	currentUser := GetCurrentUser(r)

	profileUser, err := GetUserByUsername(username)
	if err == sql.ErrNoRows {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	var currentUserID int64
	if currentUser != nil {
		currentUserID = currentUser.ID
	}

	result, err := GetPhotosByUser(profileUser.ID, page, currentUserID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "user.html", map[string]interface{}{
		"User":        currentUser,
		"ProfileUser": profileUser,
		"Photos":      result.Photos,
		"Pagination":  result,
	})
}

func Favorites(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	result, err := GetFavoritePhotos(user.ID, page)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	nuis, _ := GetAllNuis()

	renderTemplate(w, "favorites.html", map[string]interface{}{
		"User":       user,
		"Photos":     result.Photos,
		"Pagination": result,
		"Nuis":       nuis,
	})
}

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

func Upload(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet {
		nuis, err := GetAllNuis()
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		albums, _ := GetUserAlbums(user.ID)
		renderTemplate(w, "upload.html", map[string]interface{}{
			"User":   user,
			"Nuis":   nuis,
			"Albums": albums,
		})
		return
	}

	r.ParseMultipartForm(100 << 20)

	files := r.MultipartForm.File["photos"]
	if len(files) == 0 {
		http.Error(w, "No files uploaded", http.StatusBadRequest)
		return
	}

	nuiNamesInput := r.FormValue("nui_names")
	description := r.FormValue("description")
	takenAtStr := r.FormValue("taken_at")
	albumID := r.FormValue("album_id")

	nuiNames := parseNuiNames(nuiNamesInput)
	if len(nuiNames) == 0 {
		http.Error(w, "At least one nui name required", http.StatusBadRequest)
		return
	}

	for _, fileHeader := range r.MultipartForm.File["photos"] {
		file, err := fileHeader.Open()
		if err != nil {
			continue
		}

		fileData, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			continue
		}

		ext := filepath.Ext(fileHeader.Filename)
		baseFilename := strconv.FormatInt(time.Now().UnixNano(), 36) + "_" + strconv.FormatInt(user.ID, 10)
		filename := baseFilename + ext

		img, err := imaging.Decode(bytes.NewReader(fileData))
		if err != nil {
			dst, _ := os.Create(filepath.Join("static/uploads", filename))
			dst.Write(fileData)
			dst.Close()
		} else {
			compressedData, err := compressImage(img, MaxImageWidth)
			if err != nil {
				compressedData = fileData
			}

			dst, err := os.Create(filepath.Join("static/uploads", filename))
			if err != nil {
				continue
			}
			dst.Write(compressedData)
			dst.Close()

			if strings.ToLower(ext) != ".jpg" && strings.ToLower(ext) != ".jpeg" {
				jpgFilename := baseFilename + ".jpg"
				os.Rename(filepath.Join("static/uploads", filename), filepath.Join("static/uploads", jpgFilename))
				filename = jpgFilename
			}
		}

		var takenAt interface{}
		if takenAtStr != "" {
			t, err := time.Parse("2006-01-02", takenAtStr)
			if err == nil {
				takenAt = t
			}
		}

		if takenAt == nil {
			if exifTime := extractExifDate(fileData); exifTime != nil {
				takenAt = *exifTime
			}
		}

		var thumbnailFilename string
		img, err = imaging.Decode(bytes.NewReader(fileData))
		if err == nil {
			thumbData, err := generateThumbnail(img, ThumbnailSize)
			if err == nil {
				thumbnailFilename = "thumb_" + baseFilename + ".jpg"
				thumbFile, err := os.Create(filepath.Join("static/uploads", thumbnailFilename))
				if err == nil {
					thumbFile.Write(thumbData)
					thumbFile.Close()
				}
			}
		}

		result, err := database.DB.Exec(`
			INSERT INTO photos (filename, thumbnail, user_id, description, taken_at)
			VALUES (?, ?, ?, ?, ?)
		`, filename, thumbnailFilename, user.ID, description, takenAt)
		if err != nil {
			continue
		}

		photoID, _ := result.LastInsertId()

		for _, nuiName := range nuiNames {
			nuiID, err := getOrCreateNui(nuiName, user.ID)
			if err != nil {
				continue
			}
			database.DB.Exec(`INSERT OR IGNORE INTO photo_nuis (photo_id, nui_id) VALUES (?, ?)`, photoID, nuiID)
		}

		if albumID != "" {
			database.DB.Exec(`INSERT OR IGNORE INTO album_photos (album_id, photo_id) VALUES (?, ?)`, albumID, photoID)
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func getOrCreateNui(name string, userID int64) (int64, error) {
	var nuiID int64
	err := database.DB.QueryRow("SELECT id FROM nuis WHERE name = ? AND user_id = ?", name, userID).Scan(&nuiID)
	if err == nil {
		return nuiID, nil
	}

	result, err := database.DB.Exec("INSERT INTO nuis (name, user_id) VALUES (?, ?)", name, userID)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func DeletePhoto(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	photoID := r.PathValue("id")
	if photoID == "" {
		http.Error(w, "Photo ID required", http.StatusBadRequest)
		return
	}

	var filename, thumbnail string
	var userID int64
	err := database.DB.QueryRow("SELECT filename, thumbnail, user_id FROM photos WHERE id = ?", photoID).Scan(&filename, &thumbnail, &userID)
	if err == sql.ErrNoRows {
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if userID != user.ID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	_, err = database.DB.Exec("DELETE FROM photos WHERE id = ?", photoID)
	if err != nil {
		http.Error(w, "Failed to delete photo", http.StatusInternalServerError)
		return
	}

	os.Remove(filepath.Join("static/uploads", filename))
	if thumbnail != "" {
		os.Remove(filepath.Join("static/uploads", thumbnail))
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func ViewPhoto(w http.ResponseWriter, r *http.Request) {
	photoID := r.PathValue("id")
	user := GetCurrentUser(r)

	var p models.PhotoWithNuis
	var takenAt sql.NullTime
	var description sql.NullString
	var username string
	err := database.DB.QueryRow(`
		SELECT p.id, p.filename, COALESCE(p.thumbnail, ''), p.user_id, p.description, p.taken_at, p.created_at, u.username
		FROM photos p
		JOIN users u ON p.user_id = u.id
		WHERE p.id = ?
	`, photoID).Scan(&p.ID, &p.Filename, &p.Thumbnail, &p.UserID, &description, &takenAt, &p.CreatedAt, &username)

	if err == sql.ErrNoRows {
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if takenAt.Valid {
		p.TakenAt = takenAt.Time
	}
	if description.Valid {
		p.Description = description.String
	}

	p.NuiNames, _ = getNuiNamesForPhoto(p.ID)
	if user != nil {
		p.IsFavorite = isPhotoFavorited(p.ID, user.ID)
	}

	renderTemplate(w, "photo.html", map[string]interface{}{
		"User":     user,
		"Photo":    p,
		"Username": username,
	})
}

func EditPhoto(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	photoID := r.PathValue("id")

	if r.Method == http.MethodGet {
		var p models.PhotoWithNuis
		var takenAt sql.NullTime
		err := database.DB.QueryRow(`
			SELECT id, filename, COALESCE(thumbnail, ''), user_id, description, taken_at, created_at
			FROM photos WHERE id = ? AND user_id = ?
		`, photoID, user.ID).Scan(&p.ID, &p.Filename, &p.Thumbnail, &p.UserID, &p.Description, &takenAt, &p.CreatedAt)

		if err == sql.ErrNoRows {
			http.Error(w, "Photo not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		if takenAt.Valid {
			p.TakenAt = takenAt.Time
		}

		p.NuiNames, _ = getNuiNamesForPhoto(p.ID)

		nuis, _ := GetAllNuis()

		renderTemplate(w, "edit.html", map[string]interface{}{
			"User":  user,
			"Photo": p,
			"Nuis":  nuis,
		})
		return
	}

	description := r.FormValue("description")
	takenAtStr := r.FormValue("taken_at")
	nuiNamesInput := r.FormValue("nui_names")

	nuiNames := parseNuiNames(nuiNamesInput)
	if len(nuiNames) == 0 {
		http.Error(w, "At least one nui name required", http.StatusBadRequest)
		return
	}

	var takenAt interface{}
	if takenAtStr != "" {
		t, err := time.Parse("2006-01-02", takenAtStr)
		if err == nil {
			takenAt = t
		}
	}

	_, err := database.DB.Exec(`
		UPDATE photos SET description = ?, taken_at = ? WHERE id = ? AND user_id = ?
	`, description, takenAt, photoID, user.ID)

	if err != nil {
		http.Error(w, "Failed to update photo", http.StatusInternalServerError)
		return
	}

	database.DB.Exec(`DELETE FROM photo_nuis WHERE photo_id = ?`, photoID)

	for _, nuiName := range nuiNames {
		nuiID, err := getOrCreateNui(nuiName, user.ID)
		if err != nil {
			continue
		}
		database.DB.Exec(`INSERT OR IGNORE INTO photo_nuis (photo_id, nui_id) VALUES (?, ?)`, photoID, nuiID)
	}

	http.Redirect(w, r, fmt.Sprintf("/photo/%s", photoID), http.StatusSeeOther)
}

func isPhotoFavorited(photoID, userID int64) bool {
	var count int
	err := database.DB.QueryRow(`SELECT COUNT(*) FROM favorites WHERE photo_id = ? AND user_id = ?`, photoID, userID).Scan(&count)
	return err == nil && count > 0
}

func ToggleFavorite(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	photoID := r.PathValue("id")
	if photoID == "" {
		http.Error(w, "Photo ID required", http.StatusBadRequest)
		return
	}

	if isPhotoFavorited(parseID(photoID), user.ID) {
		database.DB.Exec(`DELETE FROM favorites WHERE photo_id = ? AND user_id = ?`, photoID, user.ID)
	} else {
		database.DB.Exec(`INSERT OR IGNORE INTO favorites (photo_id, user_id) VALUES (?, ?)`, photoID, user.ID)
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func parseID(s string) int64 {
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}

func GetUserAlbums(userID int64) ([]models.Album, error) {
	rows, err := database.DB.Query(`
		SELECT a.id, a.name, a.description, a.user_id, a.created_at,
			(SELECT COUNT(*) FROM album_photos WHERE album_id = a.id) as photo_count
		FROM albums a
		WHERE a.user_id = ?
		ORDER BY a.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []models.Album
	for rows.Next() {
		var a models.Album
		var description sql.NullString
		err := rows.Scan(&a.ID, &a.Name, &description, &a.UserID, &a.CreatedAt, &a.PhotoCount)
		if err != nil {
			return nil, err
		}
		if description.Valid {
			a.Description = description.String
		}
		albums = append(albums, a)
	}
	return albums, nil
}

func CreateAlbum(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		renderTemplate(w, "album_form.html", map[string]interface{}{
			"User": user,
		})
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")

	if name == "" {
		http.Error(w, "Album name required", http.StatusBadRequest)
		return
	}

	result, err := database.DB.Exec(`INSERT INTO albums (name, description, user_id) VALUES (?, ?, ?)`, name, description, user.ID)
	if err != nil {
		http.Error(w, "Failed to create album", http.StatusInternalServerError)
		return
	}

	albumID, _ := result.LastInsertId()
	http.Redirect(w, r, fmt.Sprintf("/album/%d", albumID), http.StatusSeeOther)
}

func ViewAlbum(w http.ResponseWriter, r *http.Request) {
	albumID := r.PathValue("id")
	user := GetCurrentUser(r)

	var a models.Album
	var description sql.NullString
	err := database.DB.QueryRow(`
		SELECT id, name, description, user_id, created_at
		FROM albums WHERE id = ?
	`, albumID).Scan(&a.ID, &a.Name, &description, &a.UserID, &a.CreatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Album not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	if description.Valid {
		a.Description = description.String
	}

	rows, err := database.DB.Query(`
		SELECT p.id, p.filename, COALESCE(p.thumbnail, ''), p.user_id, p.description, p.taken_at, p.created_at
		FROM photos p
		JOIN album_photos ap ON p.id = ap.photo_id
		WHERE ap.album_id = ?
		ORDER BY ap.position, p.created_at DESC
	`, albumID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var photos []models.PhotoWithNuis
	for rows.Next() {
		var p models.PhotoWithNuis
		var takenAt sql.NullTime
		var desc sql.NullString
		err := rows.Scan(&p.ID, &p.Filename, &p.Thumbnail, &p.UserID, &desc, &takenAt, &p.CreatedAt)
		if err != nil {
			continue
		}
		if takenAt.Valid {
			p.TakenAt = takenAt.Time
		}
		if desc.Valid {
			p.Description = desc.String
		}
		p.NuiNames, _ = getNuiNamesForPhoto(p.ID)
		if user != nil {
			p.IsFavorite = isPhotoFavorited(p.ID, user.ID)
		}
		photos = append(photos, p)
	}

	a.PhotoCount = len(photos)

	renderTemplate(w, "album.html", map[string]interface{}{
		"User":   user,
		"Album":  a,
		"Photos": photos,
	})
}

func ListAlbums(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	albums, err := GetUserAlbums(user.ID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "albums.html", map[string]interface{}{
		"User":   user,
		"Albums": albums,
	})
}

func DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	albumID := r.PathValue("id")

	var userID int64
	err := database.DB.QueryRow(`SELECT user_id FROM albums WHERE id = ?`, albumID).Scan(&userID)
	if err == sql.ErrNoRows {
		http.Error(w, "Album not found", http.StatusNotFound)
		return
	}
	if userID != user.ID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	database.DB.Exec(`DELETE FROM albums WHERE id = ?`, albumID)
	http.Redirect(w, r, "/albums", http.StatusSeeOther)
}

func ExportPhotos(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := database.DB.Query(`SELECT filename FROM photos WHERE user_id = ?`, user.ID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=nuistagram_backup_%s.zip", time.Now().Format("2006-01-02")))

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			continue
		}

		filePath := filepath.Join("static/uploads", filename)
		file, err := os.Open(filePath)
		if err != nil {
			continue
		}

		info, err := file.Stat()
		if err != nil {
			file.Close()
			continue
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			file.Close()
			continue
		}
		header.Name = filename
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			file.Close()
			continue
		}

		io.Copy(writer, file)
		file.Close()
	}
}
