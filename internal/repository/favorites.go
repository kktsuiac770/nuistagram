package repository

import (
	"database/sql"
)

type favoriteRepository struct {
	db *sql.DB
}

func NewFavoriteRepository(db *sql.DB) FavoriteRepository {
	return &favoriteRepository{db: db}
}

func (r *favoriteRepository) IsFavorited(photoID, userID int64) bool {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM favorites WHERE photo_id = ? AND user_id = ?`,
		photoID, userID,
	).Scan(&count)
	return err == nil && count > 0
}

func (r *favoriteRepository) Toggle(photoID, userID int64) (bool, error) {
	if r.IsFavorited(photoID, userID) {
		_, err := r.db.Exec(
			`DELETE FROM favorites WHERE photo_id = ? AND user_id = ?`,
			photoID, userID,
		)
		return false, err
	}

	_, err := r.db.Exec(
		`INSERT OR IGNORE INTO favorites (photo_id, user_id) VALUES (?, ?)`,
		photoID, userID,
	)
	return true, err
}
