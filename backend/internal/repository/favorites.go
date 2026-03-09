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
		`SELECT COUNT(*) FROM favorites WHERE photo_id = $1 AND user_id = $2`,
		photoID, userID,
	).Scan(&count)
	return err == nil && count > 0
}

func (r *favoriteRepository) Toggle(photoID, userID int64) (bool, error) {
	if r.IsFavorited(photoID, userID) {
		_, err := r.db.Exec(
			`DELETE FROM favorites WHERE photo_id = $1 AND user_id = $2`,
			photoID, userID,
		)
		return false, err
	}

	_, err := r.db.Exec(
		`INSERT INTO favorites (photo_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		photoID, userID,
	)
	return true, err
}
