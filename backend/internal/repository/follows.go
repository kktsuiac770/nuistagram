package repository

import (
	"database/sql"
	"nuistagram/internal/models"
)

type followRepository struct {
	db *sql.DB
}

func NewFollowRepository(db *sql.DB) FollowRepository {
	return &followRepository{db: db}
}

func (r *followRepository) Follow(followerID, followingID int64) error {
	if followerID == followingID {
		return nil
	}
	_, err := r.db.Exec(
		"INSERT OR IGNORE INTO follows (follower_id, following_id) VALUES (?, ?)",
		followerID, followingID,
	)
	return err
}

func (r *followRepository) Unfollow(followerID, followingID int64) error {
	_, err := r.db.Exec(
		"DELETE FROM follows WHERE follower_id = ? AND following_id = ?",
		followerID, followingID,
	)
	return err
}

func (r *followRepository) IsFollowing(followerID, followingID int64) bool {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM follows WHERE follower_id = ? AND following_id = ?",
		followerID, followingID,
	).Scan(&count)
	return err == nil && count > 0
}

func (r *followRepository) GetFollowers(userID int64, limit int) ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.username, '', COALESCE(u.bio, ''), COALESCE(u.avatar, ''), u.created_at, 0
		FROM users u
		JOIN follows f ON u.id = f.follower_id
		WHERE f.following_id = ?
		ORDER BY f.created_at DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanUsers(rows)
}

func (r *followRepository) GetFollowing(userID int64, limit int) ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.username, '', COALESCE(u.bio, ''), COALESCE(u.avatar, ''), u.created_at, 0
		FROM users u
		JOIN follows f ON u.id = f.following_id
		WHERE f.follower_id = ?
		ORDER BY f.created_at DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanUsers(rows)
}

func (r *followRepository) GetFollowerCount(userID int64) (int, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM follows WHERE following_id = ?",
		userID,
	).Scan(&count)
	return count, err
}

func (r *followRepository) GetFollowingCount(userID int64) (int, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM follows WHERE follower_id = ?",
		userID,
	).Scan(&count)
	return count, err
}

func (r *followRepository) scanUsers(rows *sql.Rows) ([]models.User, error) {
	var users []models.User
	for rows.Next() {
		var u models.User
		var bio, avatar sql.NullString
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &bio, &avatar, &u.CreatedAt, &u.PhotoCount); err != nil {
			return nil, err
		}
		u.Bio = bio.String
		u.Avatar = avatar.String
		users = append(users, u)
	}
	return users, nil
}
