package repository

import (
	"database/sql"
	"nuistagram/internal/models"
)

type albumRepository struct {
	db *sql.DB
}

func NewAlbumRepository(db *sql.DB) AlbumRepository {
	return &albumRepository{db: db}
}

func (r *albumRepository) GetByUserID(userID int64) ([]models.Album, error) {
	rows, err := r.db.Query(`
		SELECT a.id, a.name, a.description, a.user_id, a.created_at,
			(SELECT COUNT(*) FROM album_photos WHERE album_id = a.id) as photo_count
		FROM albums a
		WHERE a.user_id = ?
		ORDER BY a.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []models.Album
	for rows.Next() {
		var a models.Album
		var description sql.NullString
		if err := rows.Scan(&a.ID, &a.Name, &description, &a.UserID, &a.CreatedAt, &a.PhotoCount); err != nil {
			return nil, err
		}
		if description.Valid {
			a.Description = description.String
		}
		albums = append(albums, a)
	}
	return albums, nil
}

func (r *albumRepository) GetByID(albumID int64) (*models.AlbumWithPhotos, error) {
	var a models.Album
	var description sql.NullString
	err := r.db.QueryRow(`
		SELECT id, name, description, user_id, created_at
		FROM albums WHERE id = ?
	`, albumID).Scan(&a.ID, &a.Name, &description, &a.UserID, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	if description.Valid {
		a.Description = description.String
	}

	rows, err := r.db.Query(`
		SELECT p.id, p.filename, COALESCE(p.thumbnail, ''), p.user_id, p.description, p.taken_at, p.created_at
		FROM photos p
		JOIN album_photos ap ON p.id = ap.photo_id
		WHERE ap.album_id = ?
		ORDER BY ap.position, p.created_at DESC
	`, albumID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []models.PhotoWithNuis
	for rows.Next() {
		var p models.PhotoWithNuis
		var takenAt sql.NullTime
		var desc sql.NullString
		if err := rows.Scan(&p.ID, &p.Filename, &p.Thumbnail, &p.UserID, &desc, &takenAt, &p.CreatedAt); err != nil {
			continue
		}
		if takenAt.Valid {
			p.TakenAt = takenAt.Time
		}
		if desc.Valid {
			p.Description = desc.String
		}
		photos = append(photos, p)
	}

	a.PhotoCount = len(photos)

	return &models.AlbumWithPhotos{
		Album:  a,
		Photos: photos,
	}, nil
}

func (r *albumRepository) Create(name, description string, userID int64) (int64, error) {
	result, err := r.db.Exec(
		`INSERT INTO albums (name, description, user_id) VALUES (?, ?, ?)`,
		name, description, userID,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *albumRepository) Delete(albumID int64) error {
	_, err := r.db.Exec(`DELETE FROM albums WHERE id = ?`, albumID)
	return err
}

func (r *albumRepository) AddPhoto(albumID int64, photoID int64) error {
	_, err := r.db.Exec(
		`INSERT OR IGNORE INTO album_photos (album_id, photo_id) VALUES (?, ?)`,
		albumID, photoID,
	)
	return err
}
