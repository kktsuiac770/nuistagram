package server

import (
	"net/http"
)

// registerAPIRoutes registers all JSON API routes shared between React and template modes.
func (s *Server) registerAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/photos", s.APIGetPhotos)
	mux.HandleFunc("/api/photo/{id}", s.APIGetPhoto)
	mux.HandleFunc("/api/photo/{id}/favorite", s.APIToggleFavorite)
	mux.HandleFunc("/api/photo/{id}/like", s.APIToggleLike)
	mux.HandleFunc("/api/photo/{id}/likers", s.APIGetLikers)
	mux.HandleFunc("/api/photo/{id}/comments", s.APIGetComments)
	mux.HandleFunc("POST /api/photo/{id}/comment", s.APICreateComment)
	mux.HandleFunc("/api/comment/{id}", s.APIDeleteComment)
	mux.HandleFunc("/api/me", s.APIGetMe)
	mux.HandleFunc("/api/nuis", s.APIGetNuis)
	mux.HandleFunc("POST /api/refresh", s.Refresh)
	mux.HandleFunc("/api/user/{username}", s.APIGetUser)
	mux.HandleFunc("/api/user/{username}/photos", s.APIGetUserPhotos)
	mux.HandleFunc("/api/user/{username}/follow-status", s.APIGetFollowStatus)
	mux.HandleFunc("POST /api/user/{username}/follow", s.APIFollowByUsername)
	mux.HandleFunc("POST /api/user/{username}/unfollow", s.APIUnfollowByUsername)
	mux.HandleFunc("/api/search/users", s.APISearchUsers)
	mux.HandleFunc("POST /api/profile", s.APIUpdateProfile)
	mux.HandleFunc("POST /api/avatar", s.APIUploadAvatar)
	mux.HandleFunc("/api/notifications", s.APIGetNotifications)
	mux.HandleFunc("/api/notifications/unread", s.APIUnreadCount)
	mux.HandleFunc("POST /api/notifications/{id}/read", s.APIMarkNotificationRead)
	mux.HandleFunc("POST /api/notifications/read-all", s.APIMarkAllNotificationsRead)
}

// registerFormRoutes registers form-submission routes shared between both modes.
func (s *Server) registerFormRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /login", s.Login)
	mux.HandleFunc("POST /register", s.Register)
	mux.HandleFunc("POST /logout", s.Logout)
	mux.HandleFunc("POST /upload", s.Upload)
	mux.HandleFunc("POST /photo/{id}/delete", s.DeletePhoto)
	mux.HandleFunc("POST /photo/{id}/favorite", s.ToggleFavorite)
	mux.HandleFunc("/export", s.ExportPhotos)
}
