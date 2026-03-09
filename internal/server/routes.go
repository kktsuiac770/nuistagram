package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// registerAPIRoutes registers all JSON API routes shared between React and template modes.
func (s *Server) registerAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/photos", s.APIGetPhotos)
	mux.HandleFunc("/api/photo/{id}", s.APIGetPhoto)
	mux.HandleFunc("/api/photo/{id}/favorite", s.csrfMiddleware(s.APIToggleFavorite))
	mux.HandleFunc("/api/photo/{id}/like", s.csrfMiddleware(s.APIToggleLike))
	mux.HandleFunc("/api/photo/{id}/likers", s.APIGetLikers)
	mux.HandleFunc("/api/photo/{id}/comments", s.APIGetComments)
	mux.HandleFunc("POST /api/photo/{id}/comment", s.csrfMiddleware(s.APICreateComment))
	mux.HandleFunc("/api/comment/{id}", s.csrfMiddleware(s.APIDeleteComment))
	mux.HandleFunc("/api/me", s.APIGetMe)
	mux.HandleFunc("/api/nuis", s.APIGetNuis)
	mux.HandleFunc("/api/csrf-token", s.GetCSRFToken)
	mux.HandleFunc("/api/user/{username}", s.APIGetUser)
	mux.HandleFunc("/api/user/{username}/photos", s.APIGetUserPhotos)
	mux.HandleFunc("/api/user/{username}/follow-status", s.APIGetFollowStatus)
	mux.HandleFunc("POST /api/user/{username}/follow", s.csrfMiddleware(s.APIFollowByUsername))
	mux.HandleFunc("POST /api/user/{username}/unfollow", s.csrfMiddleware(s.APIUnfollowByUsername))
	mux.HandleFunc("/api/search/users", s.APISearchUsers)
	mux.HandleFunc("POST /api/profile", s.csrfMiddleware(s.APIUpdateProfile))
	mux.HandleFunc("POST /api/avatar", s.csrfMiddleware(s.APIUploadAvatar))
	mux.HandleFunc("/api/notifications", s.APIGetNotifications)
	mux.HandleFunc("/api/notifications/unread", s.APIUnreadCount)
	mux.HandleFunc("POST /api/notifications/{id}/read", s.csrfMiddleware(s.APIMarkNotificationRead))
	mux.HandleFunc("POST /api/notifications/read-all", s.csrfMiddleware(s.APIMarkAllNotificationsRead))
}

// registerFormRoutes registers form-submission routes shared between both modes.
func (s *Server) registerFormRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /login", rateLimitLogin(s.Login))
	mux.HandleFunc("POST /register", rateLimitLogin(s.Register))
	mux.HandleFunc("/logout", s.Logout)
	mux.HandleFunc("POST /upload", s.csrfMiddleware(s.Upload))
	mux.HandleFunc("POST /photo/{id}/delete", s.csrfMiddleware(s.DeletePhoto))
	mux.HandleFunc("POST /photo/{id}/favorite", s.csrfMiddleware(s.ToggleFavorite))
	mux.HandleFunc("/export", s.ExportPhotos)
}

// registerTemplateRoutes registers HTML template routes only used when React is not available.
func (s *Server) registerTemplateRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", s.Home)
	mux.HandleFunc("/login", rateLimitLogin(s.Login))
	mux.HandleFunc("/register", rateLimitLogin(s.Register))
	mux.HandleFunc("/upload", s.csrfMiddleware(s.Upload))
	mux.HandleFunc("/photo/{id}/delete", s.csrfMiddleware(s.DeletePhoto))
	mux.HandleFunc("/photo/{id}/favorite", s.csrfMiddleware(s.ToggleFavorite))
	mux.HandleFunc("/nui/{name}", s.NuiProfile)
	mux.HandleFunc("/user/{username}", s.UserProfile)
	mux.HandleFunc("/favorites", s.Favorites)
	mux.HandleFunc("/photo/{id}", s.ViewPhoto)
	mux.HandleFunc("/photo/{id}/edit", s.csrfMiddleware(s.EditPhoto))
	mux.HandleFunc("/albums", s.ListAlbums)
	mux.HandleFunc("/albums/new", s.csrfMiddleware(s.CreateAlbum))
	mux.HandleFunc("/album/{id}", s.ViewAlbum)
	mux.HandleFunc("/album/{id}/delete", s.csrfMiddleware(s.DeleteAlbum))
}

// Template-rendered page handlers

func (s *Server) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	user := s.currentUser(r)

	tagsParam := r.URL.Query().Get("tags")
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "or"
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	tags := parseTags(tagsParam)

	result, err := s.Repos.Photos.Search(tags, mode, page, currentUserID(user))
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	nuis, err := s.Repos.Nuis.GetAll()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	users, _ := s.Repos.Users.GetAll()

	renderTemplate(w, "index.html", map[string]interface{}{
		"User":         user,
		"Photos":       result.Photos,
		"Nuis":         nuis,
		"Users":        users,
		"SelectedTags": tags,
		"Mode":         mode,
		"Pagination":   result,
	})
}

func (s *Server) NuiProfile(w http.ResponseWriter, r *http.Request) {
	nuiName := r.PathValue("name")
	user := s.currentUser(r)

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	result, err := s.Repos.Photos.GetByNui(nuiName, page, currentUserID(user))
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

func (s *Server) UserProfile(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	currentUser := s.currentUser(r)

	profileUser, err := s.Repos.Users.GetByUsername(username)
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

	result, err := s.Repos.Photos.GetByUser(profileUser.ID, page, currentUserID(currentUser))
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

func (s *Server) Favorites(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
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

	result, err := s.Repos.Photos.GetFavorites(user.ID, page)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	nuis, _ := s.Repos.Nuis.GetAll()

	renderTemplate(w, "favorites.html", map[string]interface{}{
		"User":       user,
		"Photos":     result.Photos,
		"Pagination": result,
		"Nuis":       nuis,
	})
}

func (s *Server) ViewPhoto(w http.ResponseWriter, r *http.Request) {
	photoID := r.PathValue("id")
	user := s.currentUser(r)

	p, username, err := s.Repos.Photos.GetByID(parseID(photoID), currentUserID(user))
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

func (s *Server) EditPhoto(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	photoID := r.PathValue("id")

	if r.Method == http.MethodGet {
		p, err := s.Repos.Photos.GetForEdit(parseID(photoID), user.ID)
		if err == sql.ErrNoRows {
			http.Error(w, "Photo not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		nuis, _ := s.Repos.Nuis.GetAll()

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

	if err := s.Repos.Photos.Update(parseID(photoID), user.ID, description, takenAt); err != nil {
		http.Error(w, "Failed to update photo", http.StatusInternalServerError)
		return
	}

	var nuiIDs []int64
	for _, nuiName := range nuiNames {
		nuiID, err := s.Repos.Nuis.GetOrCreate(nuiName, user.ID)
		if err != nil {
			continue
		}
		nuiIDs = append(nuiIDs, nuiID)
	}
	s.Repos.Photos.SetNuis(parseID(photoID), nuiIDs)

	http.Redirect(w, r, fmt.Sprintf("/photo/%s", photoID), http.StatusSeeOther)
}

func (s *Server) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
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

	albumID, err := s.Repos.Albums.Create(name, description, user.ID)
	if err != nil {
		http.Error(w, "Failed to create album", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/album/%d", albumID), http.StatusSeeOther)
}

func (s *Server) ViewAlbum(w http.ResponseWriter, r *http.Request) {
	albumID := r.PathValue("id")
	user := s.currentUser(r)

	album, err := s.Repos.Albums.GetByID(parseID(albumID))
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

func (s *Server) ListAlbums(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	albums, err := s.Repos.Albums.GetByUserID(user.ID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "albums.html", map[string]interface{}{
		"User":   user,
		"Albums": albums,
	})
}

func (s *Server) DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	albumID := r.PathValue("id")

	ownerID, err := s.Repos.Albums.GetOwnerID(parseID(albumID))
	if err == sql.ErrNoRows {
		http.Error(w, "Album not found", http.StatusNotFound)
		return
	}
	if ownerID != user.ID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	s.Repos.Albums.Delete(parseID(albumID))
	http.Redirect(w, r, "/albums", http.StatusSeeOther)
}
