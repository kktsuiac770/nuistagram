package server

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/disintegration/imaging"
)

func (s *Server) APISearchUsers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, 200, []interface{}{})
		return
	}

	users, err := s.Repos.Users.Search(query, 20)
	if err != nil {
		jsonError(w, 500, "internal error")
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

	writeJSON(w, 200, result)
}

func (s *Server) APIUpdateProfile(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		jsonError(w, 401, "unauthorized")
		return
	}

	bio := r.FormValue("bio")
	if len(bio) > 500 {
		jsonError(w, 400, "bio too long")
		return
	}

	err := s.Repos.Users.UpdateProfile(user.ID, bio)
	if err != nil {
		jsonError(w, 500, "internal error")
		return
	}

	writeJSON(w, 200, map[string]bool{"success": true})
}

func (s *Server) APIUploadAvatar(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		jsonError(w, 401, "unauthorized")
		return
	}

	err := r.ParseMultipartForm(5 << 20)
	if err != nil {
		jsonError(w, 400, "file too large")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		jsonError(w, 400, "no file uploaded")
		return
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		jsonError(w, 500, "failed to read file")
		return
	}

	avatarDir := filepath.Join("static", "uploads", "avatars")
	os.MkdirAll(avatarDir, 0755)

	ext := filepath.Ext(header.Filename)
	filename := strconv.FormatInt(time.Now().UnixNano(), 36) + "_" + strconv.FormatInt(user.ID, 10) + ext

	img, err := imaging.Decode(bytes.NewReader(fileData))
	if err == nil {
		img = imaging.Fill(img, 200, 200, imaging.Center, imaging.Lanczos)
		if ext == ".jpg" || ext == ".jpeg" {
			imaging.Save(img, filepath.Join(avatarDir, filename))
		} else {
			filename = filename[:len(filename)-len(ext)] + ".jpg"
			imaging.Save(img, filepath.Join(avatarDir, filename))
		}
	} else {
		avatarPath := filepath.Join(avatarDir, filename)
		dst, err := os.Create(avatarPath)
		if err != nil {
			jsonError(w, 500, "failed to save file")
			return
		}
		dst.Write(fileData)
		dst.Close()
	}

	avatarURL := "avatars/" + filename
	err = s.Repos.Users.UpdateAvatar(user.ID, avatarURL)
	if err != nil {
		jsonError(w, 500, "internal error")
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"success": true,
		"avatar":  avatarURL,
	})
}
