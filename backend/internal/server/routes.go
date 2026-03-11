package server

import (
	"net/http"
)

func (s *Server) registerAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/photos", s.photos.APIGetPhotos)
	mux.HandleFunc("/api/photo/{id}", s.photos.APIGetPhoto)
	mux.HandleFunc("/api/photo/{id}/favorite", s.photos.APIToggleFavorite)
	mux.HandleFunc("/api/photo/{id}/like", s.likes.APIToggleLike)
	mux.HandleFunc("/api/photo/{id}/likers", s.likes.APIGetLikers)
	mux.HandleFunc("/api/photo/{id}/comments", s.comments.APIGetComments)
	mux.HandleFunc("POST /api/photo/{id}/comment", s.comments.APICreateComment)
	mux.HandleFunc("/api/comment/{id}", s.comments.APIDeleteComment)
	mux.HandleFunc("/api/me", s.users.APIGetMe)
	mux.HandleFunc("/api/nuis", s.photos.APIGetNuis)
	mux.HandleFunc("POST /api/refresh", s.auth.Refresh)
	mux.HandleFunc("/api/user/{username}", s.users.APIGetUser)
	mux.HandleFunc("/api/user/{username}/photos", s.photos.APIGetUserPhotos)
	mux.HandleFunc("/api/user/{username}/follow-status", s.follows.APIGetFollowStatus)
	mux.HandleFunc("POST /api/user/{username}/follow", s.follows.APIFollowByUsername)
	mux.HandleFunc("POST /api/user/{username}/unfollow", s.follows.APIUnfollowByUsername)
	mux.HandleFunc("/api/search/users", s.users.APISearchUsers)
	mux.HandleFunc("POST /api/profile", s.users.APIUpdateProfile)
	mux.HandleFunc("POST /api/avatar", s.users.APIUploadAvatar)
	mux.HandleFunc("/api/notifications", s.notifications.APIGetNotifications)
	mux.HandleFunc("/api/notifications/unread", s.notifications.APIUnreadCount)
	mux.HandleFunc("POST /api/notifications/{id}/read", s.notifications.APIMarkNotificationRead)
	mux.HandleFunc("POST /api/notifications/read-all", s.notifications.APIMarkAllNotificationsRead)
}

func (s *Server) registerAdminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /admin/users/{username}/role", s.admin.APIUpdateUserRole)
}

func (s *Server) registerFormRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /login", s.auth.Login)
	mux.HandleFunc("POST /register", s.auth.Register)
	mux.HandleFunc("POST /logout", s.auth.Logout)
	mux.HandleFunc("POST /upload", s.photos.Upload)
	mux.HandleFunc("POST /photo/{id}/delete", s.photos.DeletePhoto)
	mux.HandleFunc("POST /photo/{id}/favorite", s.photos.ToggleFavorite)
	mux.HandleFunc("/export", s.photos.ExportPhotos)
}
