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

	result, err := Repos.Photos.GetByNui(nuiName, page, userID)
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

	profileUser, err := Repos.Users.GetByUsername(username)
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

	result, err := Repos.Photos.GetByUser(profileUser.ID, page, currentUserID)
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

	result, err := Repos.Photos.GetFavorites(user.ID, page)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	nuis, _ := Repos.Nuis.GetAll()

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
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	r.ParseMultipartForm(100 << 20)

	files := r.MultipartForm.File["photos"]
	if len(files) == 0 {
		http.Error(w, `{"error": "No files uploaded"}`, http.StatusBadRequest)
		return
	}

	nuiNamesInput := r.FormValue("nui_names")
	description := r.FormValue("description")
	takenAtStr := r.FormValue("taken_at")
	albumID := r.FormValue("album_id")

	nuiNames := parseNuiNames(nuiNamesInput)
	if len(nuiNames) == 0 {
		http.Error(w, `{"error": "At least one nui name required"}`, http.StatusBadRequest)
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
			compressedData, err := compressImage(img, 1920)
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
		img, err = imaging.Decode(bytes.NewReader(fileData))
		if err == nil {
			thumbData, err := generateThumbnail(img, 400)
			if err == nil {
				thumbnailFilename = "thumb_" + baseFilename + ".jpg"
				thumbFile, err := os.Create(filepath.Join("static/uploads", thumbnailFilename))
				if err == nil {
					thumbFile.Write(thumbData)
					thumbFile.Close()
				}
			}
		}

		photoID, err := Repos.Photos.Create(filename, thumbnailFilename, user.ID, description, takenAt)
		if err != nil {
			continue
		}

		for _, nuiName := range nuiNames {
			nuiID, err := Repos.Nuis.GetOrCreate(nuiName, user.ID)
			if err != nil {
				continue
			}
			database.DB.Exec(`INSERT OR IGNORE INTO photo_nuis (photo_id, nui_id) VALUES (?, ?)`, photoID, nuiID)
		}

		if albumID != "" {
			albumIDInt, _ := strconv.ParseInt(albumID, 10, 64)
			Repos.Albums.AddPhoto(albumIDInt, photoID)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true}`))
}

func DeletePhoto(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	photoID := r.PathValue("id")
	if photoID == "" {
		http.Error(w, `{"error": "Photo ID required"}`, http.StatusBadRequest)
		return
	}

	var filename, thumbnail string
	var userID int64
	err := database.DB.QueryRow("SELECT filename, thumbnail, user_id FROM photos WHERE id = ?", photoID).Scan(&filename, &thumbnail, &userID)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Photo not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if userID != user.ID {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusForbidden)
		return
	}

	if err := Repos.Photos.Delete(parseID(photoID)); err != nil {
		http.Error(w, `{"error": "Failed to delete photo"}`, http.StatusInternalServerError)
		return
	}

	os.Remove(filepath.Join("static/uploads", filename))
	if thumbnail != "" {
		os.Remove(filepath.Join("static/uploads", thumbnail))
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true}`))
}

func ViewPhoto(w http.ResponseWriter, r *http.Request) {
	photoID := r.PathValue("id")
	user := GetCurrentUser(r)

	var currentUserID int64
	if user != nil {
		currentUserID = user.ID
	}

	p, username, err := Repos.Photos.GetByID(parseID(photoID), currentUserID)
	if err == sql.ErrNoRows {
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
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
		p, err := Repos.Photos.GetForEdit(parseID(photoID), user.ID)
		if err == sql.ErrNoRows {
			http.Error(w, "Photo not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		nuis, _ := Repos.Nuis.GetAll()

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

	var takenAt *time.Time
	if takenAtStr != "" {
		t, err := time.Parse("2006-01-02", takenAtStr)
		if err == nil {
			takenAt = &t
		}
	}

	if err := Repos.Photos.Update(parseID(photoID), user.ID, description, takenAt); err != nil {
		http.Error(w, "Failed to update photo", http.StatusInternalServerError)
		return
	}

	database.DB.Exec(`DELETE FROM photo_nuis WHERE photo_id = ?`, photoID)

	for _, nuiName := range nuiNames {
		nuiID, err := Repos.Nuis.GetOrCreate(nuiName, user.ID)
		if err != nil {
			continue
		}
		database.DB.Exec(`INSERT OR IGNORE INTO photo_nuis (photo_id, nui_id) VALUES (?, ?)`, photoID, nuiID)
	}

	http.Redirect(w, r, fmt.Sprintf("/photo/%s", photoID), http.StatusSeeOther)
}

func ToggleFavorite(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	photoID := r.PathValue("id")
	if photoID == "" {
		http.Error(w, `{"error": "Photo ID required"}`, http.StatusBadRequest)
		return
	}

	isFav, err := Repos.Favorites.Toggle(parseID(photoID), user.ID)
	if err != nil {
		http.Error(w, `{"error": "Failed to toggle favorite"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "is_favorite": %v}`, isFav)
}

func parseID(s string) int64 {
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}

func GetUserAlbums(userID int64) ([]models.Album, error) {
	return Repos.Albums.GetByUserID(userID)
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

	albumID, err := Repos.Albums.Create(name, description, user.ID)
	if err != nil {
		http.Error(w, "Failed to create album", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/album/%d", albumID), http.StatusSeeOther)
}

func ViewAlbum(w http.ResponseWriter, r *http.Request) {
	albumID := r.PathValue("id")
	user := GetCurrentUser(r)

	album, err := Repos.Albums.GetByID(parseID(albumID))
	if err == sql.ErrNoRows {
		http.Error(w, "Album not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "album.html", map[string]interface{}{
		"User":   user,
		"Album":  album.Album,
		"Photos": album.Photos,
	})
}

func ListAlbums(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	albums, err := Repos.Albums.GetByUserID(user.ID)
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

	Repos.Albums.Delete(parseID(albumID))
	http.Redirect(w, r, "/albums", http.StatusSeeOther)
}

func ExportPhotos(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	filenames, err := Repos.Photos.GetFilenamesByUser(user.ID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=nuistagram_backup_%s.zip", time.Now().Format("2006-01-02")))

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	for _, filename := range filenames {
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
