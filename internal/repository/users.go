package repository

import (
	"database/sql"
	"nuistagram/internal/models"
	"strings"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetByID(id int64) (*models.User, error) {
	var u models.User
	var bio, avatar sql.NullString
	err := r.db.QueryRow(`
		SELECT u.id, u.username, u.password_hash, COALESCE(u.bio, ''), COALESCE(u.avatar, ''), u.created_at, 
			(SELECT COUNT(*) FROM photos WHERE user_id = u.id) as photo_count
		FROM users u WHERE u.id = ?
	`, id).Scan(&u.ID, &u.Username, &u.PasswordHash, &bio, &avatar, &u.CreatedAt, &u.PhotoCount)
	if err != nil {
		return nil, err
	}
	u.Bio = bio.String
	u.Avatar = avatar.String
	return &u, nil
}

func (r *userRepository) GetByIDWithCounts(id int64, currentUserID int64) (*models.User, error) {
	var u models.User
	var bio, avatar sql.NullString
	err := r.db.QueryRow(`
		SELECT u.id, u.username, u.password_hash, COALESCE(u.bio, ''), COALESCE(u.avatar, ''), u.created_at, 
			(SELECT COUNT(*) FROM photos WHERE user_id = u.id) as photo_count,
			(SELECT COUNT(*) FROM follows WHERE following_id = u.id) as follower_count,
			(SELECT COUNT(*) FROM follows WHERE follower_id = u.id) as following_count
		FROM users u WHERE u.id = ?
	`, id).Scan(&u.ID, &u.Username, &u.PasswordHash, &bio, &avatar, &u.CreatedAt, &u.PhotoCount, &u.FollowerCount, &u.FollowingCount)
	if err != nil {
		return nil, err
	}
	u.Bio = bio.String
	u.Avatar = avatar.String
	if currentUserID > 0 && currentUserID != id {
		u.IsFollowing = r.isFollowing(currentUserID, id)
	}
	return &u, nil
}

func (r *userRepository) GetByUsername(username string) (*models.User, error) {
	var u models.User
	var bio, avatar sql.NullString
	err := r.db.QueryRow(`
		SELECT u.id, u.username, u.password_hash, COALESCE(u.bio, ''), COALESCE(u.avatar, ''), u.created_at,
			(SELECT COUNT(*) FROM photos WHERE user_id = u.id) as photo_count
		FROM users u WHERE u.username = ?
	`, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &bio, &avatar, &u.CreatedAt, &u.PhotoCount)
	if err != nil {
		return nil, err
	}
	u.Bio = bio.String
	u.Avatar = avatar.String
	return &u, nil
}

func (r *userRepository) GetByUsernameWithCounts(username string, currentUserID int64) (*models.User, error) {
	var u models.User
	var bio, avatar sql.NullString
	err := r.db.QueryRow(`
		SELECT u.id, u.username, u.password_hash, COALESCE(u.bio, ''), COALESCE(u.avatar, ''), u.created_at,
			(SELECT COUNT(*) FROM photos WHERE user_id = u.id) as photo_count,
			(SELECT COUNT(*) FROM follows WHERE following_id = u.id) as follower_count,
			(SELECT COUNT(*) FROM follows WHERE follower_id = u.id) as following_count
		FROM users u WHERE u.username = ?
	`, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &bio, &avatar, &u.CreatedAt, &u.PhotoCount, &u.FollowerCount, &u.FollowingCount)
	if err != nil {
		return nil, err
	}
	u.Bio = bio.String
	u.Avatar = avatar.String
	if currentUserID > 0 && currentUserID != u.ID {
		u.IsFollowing = r.isFollowing(currentUserID, u.ID)
	}
	return &u, nil
}

func (r *userRepository) GetAll() ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.username, u.password_hash, COALESCE(u.bio, ''), COALESCE(u.avatar, ''), u.created_at,
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

func (r *userRepository) Search(query string, limit int) ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.username, u.password_hash, COALESCE(u.bio, ''), COALESCE(u.avatar, ''), u.created_at,
			(SELECT COUNT(*) FROM photos WHERE user_id = u.id) as photo_count
		FROM users u 
		WHERE LOWER(u.username) LIKE LOWER(?)
		ORDER BY u.username
		LIMIT ?
	`, "%"+strings.ToLower(query)+"%", limit)
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

func (r *userRepository) UpdateProfile(userID int64, bio string) error {
	_, err := r.db.Exec("UPDATE users SET bio = ? WHERE id = ?", bio, userID)
	return err
}

func (r *userRepository) UpdateAvatar(userID int64, avatar string) error {
	_, err := r.db.Exec("UPDATE users SET avatar = ? WHERE id = ?", avatar, userID)
	return err
}

func (r *userRepository) isFollowing(followerID, followingID int64) bool {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM follows WHERE follower_id = ? AND following_id = ?",
		followerID, followingID,
	).Scan(&count)
	return err == nil && count > 0
}
