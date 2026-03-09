package server

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
)

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

func (s *Server) Upload(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	r.ParseMultipartForm(100 << 20)

	files := r.MultipartForm.File["photos"]
	if len(files) == 0 {
		jsonError(w, http.StatusBadRequest, "No files uploaded")
		return
	}

	nuiNamesInput := r.FormValue("nui_names")
	description := r.FormValue("description")
	takenAtStr := r.FormValue("taken_at")
	albumID := r.FormValue("album_id")

	nuiNames := parseNuiNames(nuiNamesInput)
	if len(nuiNames) == 0 {
		jsonError(w, http.StatusBadRequest, "At least one nui name required")
		return
	}

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			continue
		}

		fileData, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			continue
		}

		baseFilename := strconv.FormatInt(time.Now().UnixNano(), 36) + "_" + strconv.FormatInt(user.ID, 10)

		// Process the main image.
		img, imgErr := imaging.Decode(bytes.NewReader(fileData))
		var photoData []byte
		if imgErr == nil {
			photoData, err = compressImage(img, 1920)
			if err != nil {
				photoData = fileData
			}
		} else {
			photoData = fileData
		}

		photoResult, err := s.Storage.Upload(r.Context(), photoData, baseFilename+".jpg")
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

		// Generate and upload thumbnail.
		var thumbnailFilename string
		if imgErr == nil {
			thumbData, err := generateThumbnail(img, 400)
			if err == nil {
				thumbResult, err := s.Storage.Upload(r.Context(), thumbData, "thumb_"+baseFilename+".jpg")
				if err == nil {
					thumbnailFilename = thumbResult.URL
				}
			}
		}

		photoID, err := s.Repos.Photos.Create(filename, thumbnailFilename, user.ID, description, takenAt)
		if err != nil {
			continue
		}

		var nuiIDs []int64
		for _, nuiName := range nuiNames {
			nuiID, err := s.Repos.Nuis.GetOrCreate(nuiName, user.ID)
			if err != nil {
				continue
			}
			nuiIDs = append(nuiIDs, nuiID)
		}
		s.Repos.Photos.SetNuis(photoID, nuiIDs)

		if albumID != "" {
			albumIDInt, _ := strconv.ParseInt(albumID, 10, 64)
			s.Repos.Albums.AddPhoto(albumIDInt, photoID)
		}
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	photoID := r.PathValue("id")
	if photoID == "" {
		jsonError(w, http.StatusBadRequest, "Photo ID required")
		return
	}

	filename, thumbnail, ownerID, err := s.Repos.Photos.GetOwner(parseID(photoID))
	if err == sql.ErrNoRows {
		jsonError(w, http.StatusNotFound, "Photo not found")
		return
	}
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Internal error")
		return
	}

	if ownerID != user.ID {
		jsonError(w, http.StatusForbidden, "Unauthorized")
		return
	}

	if err := s.Repos.Photos.Delete(parseID(photoID)); err != nil {
		jsonError(w, http.StatusInternalServerError, "Failed to delete photo")
		return
	}

	s.Storage.Delete(r.Context(), filename)
	if thumbnail != "" {
		s.Storage.Delete(r.Context(), thumbnail)
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) ToggleFavorite(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	photoID := r.PathValue("id")
	if photoID == "" {
		jsonError(w, http.StatusBadRequest, "Photo ID required")
		return
	}

	isFav, err := s.Repos.Favorites.Toggle(parseID(photoID), user.ID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Failed to toggle favorite")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "is_favorite": isFav})
}

func (s *Server) ExportPhotos(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	filenames, err := s.Repos.Photos.GetFilenamesByUser(user.ID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=nuistagram_backup_%s.zip", time.Now().Format("2006-01-02")))

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	for _, filename := range filenames {
		data, err := s.Storage.FetchContent(r.Context(), filename)
		if err != nil {
			continue
		}

		// Use the base name (after the last "/") for the zip entry.
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
