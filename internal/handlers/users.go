package handlers

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

func APISearchUsers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		WriteJSON(w, 200, []interface{}{})
		return
	}

	users, err := Repos.Users.Search(query, 20)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
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

	WriteJSON(w, 200, result)
}

func APIUpdateProfile(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		WriteJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	bio := r.FormValue("bio")
	if len(bio) > 500 {
		WriteJSON(w, 400, map[string]string{"error": "bio too long"})
		return
	}

	err := Repos.Users.UpdateProfile(user.ID, bio)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	WriteJSON(w, 200, map[string]bool{"success": true})
}

func APIUploadAvatar(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		WriteJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	err := r.ParseMultipartForm(5 << 20)
	if err != nil {
		WriteJSON(w, 400, map[string]string{"error": "file too large"})
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		WriteJSON(w, 400, map[string]string{"error": "no file uploaded"})
		return
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "failed to read file"})
		return
	}

	avatarDir := filepath.Join("static", "uploads", "avatars")
	os.MkdirAll(avatarDir, 0755)

	ext := filepath.Ext(header.Filename)
	filename := strconv.FormatInt(time.Now().UnixNano(), 36) + "_" + strconv.FormatInt(user.ID, 10) + ext

	img, err := imaging.Decode(bytes.NewReader(fileData))
	if err == nil {
		img = imaging.Fill(img, 200, 200, imaging.Center, imaging.Lanczos)
		avatarPath := filepath.Join(avatarDir, filename)
		if ext == ".jpg" || ext == ".jpeg" {
			imaging.Save(img, avatarPath)
		} else {
			filename = filename[:len(filename)-len(ext)] + ".jpg"
			imaging.Save(img, filepath.Join(avatarDir, filename))
		}
	} else {
		avatarPath := filepath.Join(avatarDir, filename)
		dst, err := os.Create(avatarPath)
		if err != nil {
			WriteJSON(w, 500, map[string]string{"error": "failed to save file"})
			return
		}
		dst.Write(fileData)
		dst.Close()
	}

	avatarURL := "avatars/" + filename
	err = Repos.Users.UpdateAvatar(user.ID, avatarURL)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	WriteJSON(w, 200, map[string]interface{}{
		"success": true,
		"avatar":  avatarURL,
	})
}

func APIFollowByUsername(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		WriteJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	username := r.PathValue("username")
	targetUser, err := Repos.Users.GetByUsername(username)
	if err != nil {
		WriteJSON(w, 404, map[string]string{"error": "user not found"})
		return
	}

	if targetUser.ID == user.ID {
		WriteJSON(w, 400, map[string]string{"error": "cannot follow yourself"})
		return
	}

	err = Repos.Follows.Follow(user.ID, targetUser.ID)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	Repos.Notifications.Create(targetUser.ID, user.ID, "follow", 0, 0)

	WriteJSON(w, 200, map[string]bool{"success": true})
}

func APIUnfollowByUsername(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		WriteJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	username := r.PathValue("username")
	targetUser, err := Repos.Users.GetByUsername(username)
	if err != nil {
		WriteJSON(w, 404, map[string]string{"error": "user not found"})
		return
	}

	err = Repos.Follows.Unfollow(user.ID, targetUser.ID)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	WriteJSON(w, 200, map[string]bool{"success": true})
}

func parseIDInt(id string) (int64, error) {
	var result int64
	_, err := strconv.ParseInt(id, 10, 64)
	return result, err
}
