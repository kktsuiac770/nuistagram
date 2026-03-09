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
	"os"
	"path/filepath"
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

	os.Remove(filepath.Join("static/uploads", filename))
	if thumbnail != "" {
		os.Remove(filepath.Join("static/uploads", thumbnail))
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
