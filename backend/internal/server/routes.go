package server

import (
	"net/http"
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
