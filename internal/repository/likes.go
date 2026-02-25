package repository

import (
	"database/sql"
	"nuistagram/internal/models"
)

type likeRepository struct {
	db *sql.DB
}

func NewLikeRepository(db *sql.DB) LikeRepository {
	return &likeRepository{db: db}
}

func (r *likeRepository) Toggle(photoID, userID int64) (bool, error) {
	if r.IsLiked(photoID, userID) {
		_, err := r.db.Exec(
			"DELETE FROM likes WHERE photo_id = ? AND user_id = ?",
			photoID, userID,
		)
		return false, err
	}

	_, err := r.db.Exec(
		"INSERT INTO likes (photo_id, user_id) VALUES (?, ?)",
		photoID, userID,
	)
	return true, err
}

func (r *likeRepository) IsLiked(photoID, userID int64) bool {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM likes WHERE photo_id = ? AND user_id = ?",
		photoID, userID,
	).Scan(&count)
	return err == nil && count > 0
}

func (r *likeRepository) GetLikeCount(photoID int64) (int, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM likes WHERE photo_id = ?",
		photoID,
	).Scan(&count)
	return count, err
}

func (r *likeRepository) GetLikers(photoID int64, limit int) ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.username, '', COALESCE(u.bio, ''), COALESCE(u.avatar, ''), u.created_at, 0
		FROM users u
		JOIN likes l ON u.id = l.user_id
		WHERE l.photo_id = ?
		ORDER BY l.created_at DESC
		LIMIT ?
	`, photoID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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
