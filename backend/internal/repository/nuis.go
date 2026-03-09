package repository

import (
	"database/sql"
	"nuistagram/internal/cache"
	"nuistagram/internal/models"
)

const nuiCacheKey = "nuis:all"

type nuiRepository struct {
	db    *sql.DB
	cache *cache.Cache
}

func NewCachedNuiRepository(db *sql.DB, c *cache.Cache) NuiRepository {
	return &nuiRepository{db: db, cache: c}
}

func (r *nuiRepository) GetAll() ([]models.Nui, error) {
	if cached, ok := r.cache.Get(nuiCacheKey); ok {
		return cached.([]models.Nui), nil
	}

	rows, err := r.db.Query(`SELECT id, name, user_id, created_at FROM nuis ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nuis []models.Nui
	for rows.Next() {
		var n models.Nui
		if err := rows.Scan(&n.ID, &n.Name, &n.UserID, &n.CreatedAt); err != nil {
			return nil, err
		}
		nuis = append(nuis, n)
	}

	r.cache.Set(nuiCacheKey, nuis)
	return nuis, nil
}

func (r *nuiRepository) GetByUserID(userID int64) ([]models.Nui, error) {
	rows, err := r.db.Query(
		`SELECT id, name, user_id, created_at FROM nuis WHERE user_id = ? ORDER BY name`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nuis []models.Nui
	for rows.Next() {
		var n models.Nui
		if err := rows.Scan(&n.ID, &n.Name, &n.UserID, &n.CreatedAt); err != nil {
			return nil, err
		}
		nuis = append(nuis, n)
	}
	return nuis, nil
}

func (r *nuiRepository) GetOrCreate(name string, userID int64) (int64, error) {
	var nuiID int64
	err := r.db.QueryRow(
		"SELECT id FROM nuis WHERE name = ? AND user_id = ?",
		name, userID,
	).Scan(&nuiID)
	if err == nil {
		return nuiID, nil
	}

	result, err := r.db.Exec(
		"INSERT INTO nuis (name, user_id) VALUES (?, ?)",
		name, userID,
	)
	if err != nil {
		return 0, err
	}

	r.InvalidateCache()
	return result.LastInsertId()
}

func (r *nuiRepository) InvalidateCache() {
	r.cache.Delete(nuiCacheKey)
}
