package repository

import (
	"database/sql"
	"nuistagram/internal/models"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetByID(id int64) (*models.User, error) {
	var u models.User
	err := r.db.QueryRow(`
		SELECT u.id, u.username, u.password_hash, u.created_at, 
			(SELECT COUNT(*) FROM photos WHERE user_id = u.id) as photo_count
		FROM users u WHERE u.id = ?
	`, id).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.PhotoCount)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) GetByUsername(username string) (*models.User, error) {
	var u models.User
	err := r.db.QueryRow(`
		SELECT u.id, u.username, u.password_hash, u.created_at,
			(SELECT COUNT(*) FROM photos WHERE user_id = u.id) as photo_count
		FROM users u WHERE u.username = ?
	`, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.PhotoCount)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) GetAll() ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.username, u.password_hash, u.created_at,
			(SELECT COUNT(*) FROM photos WHERE user_id = u.id) as photo_count
		FROM users u ORDER BY u.username
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.PhotoCount); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *userRepository) Create(username, passwordHash string) (int64, error) {
	result, err := r.db.Exec(
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		username, passwordHash,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}
