package handlers

import (
	"net/http"
	"strconv"
)

func APIFollow(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		WriteJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	followingID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteJSON(w, 400, map[string]string{"error": "invalid user id"})
		return
	}

	if followingID == user.ID {
		WriteJSON(w, 400, map[string]string{"error": "cannot follow yourself"})
		return
	}

	err = Repos.Follows.Follow(user.ID, followingID)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	Repos.Notifications.Create(followingID, user.ID, "follow", 0, 0)

	WriteJSON(w, 200, map[string]bool{"success": true})
}

func APIUnfollow(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		WriteJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	followingID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteJSON(w, 400, map[string]string{"error": "invalid user id"})
		return
	}

	err = Repos.Follows.Unfollow(user.ID, followingID)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	WriteJSON(w, 200, map[string]bool{"success": true})
}

func APIFollowStatus(w http.ResponseWriter, r *http.Request) {
	user := GetCurrentUser(r)
	if user == nil {
		WriteJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	followingID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteJSON(w, 400, map[string]string{"error": "invalid user id"})
		return
	}

	isFollowing := Repos.Follows.IsFollowing(user.ID, followingID)
	WriteJSON(w, 200, map[string]bool{"is_following": isFollowing})
}

func APIGetFollowers(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteJSON(w, 400, map[string]string{"error": "invalid user id"})
		return
	}

	followers, err := Repos.Follows.GetFollowers(userID, 50)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	WriteJSON(w, 200, followers)
}

func APIGetFollowing(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteJSON(w, 400, map[string]string{"error": "invalid user id"})
		return
	}

	following, err := Repos.Follows.GetFollowing(userID, 50)
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}

	WriteJSON(w, 200, following)
}
