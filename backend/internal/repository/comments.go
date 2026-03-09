package repository

import (
	"database/sql"
	"nuistagram/internal/models"
)

type commentRepository struct {
	db *sql.DB
}

func NewCommentRepository(db *sql.DB) CommentRepository {
	return &commentRepository{db: db}
}

func (r *commentRepository) Create(photoID, userID int64, content string) (int64, error) {
	var id int64
	err := r.db.QueryRow(
		"INSERT INTO comments (photo_id, user_id, content) VALUES ($1, $2, $3) RETURNING id",
		photoID, userID, content,
	).Scan(&id)
	return id, err
}

func (r *commentRepository) GetByPhotoID(photoID int64, limit, offset int) ([]models.Comment, error) {
	rows, err := r.db.Query(`
		SELECT c.id, c.photo_id, c.user_id, c.content, c.created_at, u.username
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.photo_id = $1
		ORDER BY c.created_at DESC
		LIMIT $2 OFFSET $3
	`, photoID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var c models.Comment
		if err := rows.Scan(&c.ID, &c.PhotoID, &c.UserID, &c.Content, &c.CreatedAt, &c.Username); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, nil
}

func (r *commentRepository) GetCount(photoID int64) (int, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM comments WHERE photo_id = $1",
		photoID,
	).Scan(&count)
	return count, err
}

func (r *commentRepository) Delete(commentID, userID int64) error {
	_, err := r.db.Exec(
		"DELETE FROM comments WHERE id = $1 AND user_id = $2",
		commentID, userID,
	)
	return err
}

func (r *commentRepository) GetByID(commentID int64) (*models.Comment, error) {
	var c models.Comment
	err := r.db.QueryRow(`
		SELECT c.id, c.photo_id, c.user_id, c.content, c.created_at, u.username
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.id = $1
	`, commentID).Scan(&c.ID, &c.PhotoID, &c.UserID, &c.Content, &c.CreatedAt, &c.Username)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
