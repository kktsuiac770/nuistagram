package server

import (
	"bytes"
	"image/jpeg"
	"io"
	"net/http"
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

	baseFilename := strconv.FormatInt(time.Now().UnixNano(), 36) + "_" + strconv.FormatInt(user.ID, 10)

	var avatarData []byte
	var uploadName string

	img, err := imaging.Decode(bytes.NewReader(fileData))
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

	result, err := s.Storage.Upload(r.Context(), avatarData, uploadName)
	if err != nil {
		jsonError(w, 500, "failed to upload avatar")
		return
	}

	err = s.Repos.Users.UpdateAvatar(user.ID, result.URL)
	if err != nil {
		jsonError(w, 500, "internal error")
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"success": true,
		"avatar":  result.URL,
	})
}
