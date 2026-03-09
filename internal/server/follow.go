package server

import (
	"net/http"
)

func (s *Server) APIFollowByUsername(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		jsonError(w, 401, "unauthorized")
		return
	}

	username := r.PathValue("username")
	targetUser, err := s.Repos.Users.GetByUsername(username)
	if err != nil {
		jsonError(w, 404, "user not found")
		return
	}

	if targetUser.ID == user.ID {
		jsonError(w, 400, "cannot follow yourself")
		return
	}

	err = s.Repos.Follows.Follow(user.ID, targetUser.ID)
	if err != nil {
		jsonError(w, 500, "internal error")
		return
	}

	s.Repos.Notifications.Create(targetUser.ID, user.ID, "follow", 0, 0)

	writeJSON(w, 200, map[string]bool{"success": true})
}

func (s *Server) APIUnfollowByUsername(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		jsonError(w, 401, "unauthorized")
		return
	}

	username := r.PathValue("username")
	targetUser, err := s.Repos.Users.GetByUsername(username)
	if err != nil {
		jsonError(w, 404, "user not found")
		return
	}

	err = s.Repos.Follows.Unfollow(user.ID, targetUser.ID)
	if err != nil {
		jsonError(w, 500, "internal error")
		return
	}

	writeJSON(w, 200, map[string]bool{"success": true})
}
