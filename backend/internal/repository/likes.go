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

func (r *likeRepository) Toggle(photoID, userID int64) (bool, int, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return false, 0, err
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		"INSERT INTO likes (photo_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		photoID, userID,
	)
	if err != nil {
		return false, 0, err
	}

	inserted, _ := result.RowsAffected()
	isLiked := inserted > 0

	if !isLiked {
		_, err = tx.Exec(
			"DELETE FROM likes WHERE photo_id = $1 AND user_id = $2",
			photoID, userID,
		)
		if err != nil {
			return false, 0, err
		}
	}

	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM likes WHERE photo_id = $1", photoID).Scan(&count)
	if err != nil {
		return false, 0, err
	}

	if err = tx.Commit(); err != nil {
		return false, 0, err
	}
	return isLiked, count, nil
}

func (r *likeRepository) IsLiked(photoID, userID int64) bool {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM likes WHERE photo_id = $1 AND user_id = $2",
		photoID, userID,
	).Scan(&count)
	return err == nil && count > 0
}

func (r *likeRepository) GetLikeCount(photoID int64) (int, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM likes WHERE photo_id = $1",
		photoID,
	).Scan(&count)
	return count, err
}

func (r *likeRepository) GetLikers(photoID int64, limit int) ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.username, '', COALESCE(u.bio, ''), COALESCE(u.avatar, ''), u.created_at, 0
		FROM users u
		JOIN likes l ON u.id = l.user_id
		WHERE l.photo_id = $1
		ORDER BY l.created_at DESC
		LIMIT $2
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
