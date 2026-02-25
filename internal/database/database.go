package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(time.Hour)
	DB.SetConnMaxIdleTime(10 * time.Minute)

	return createTables()
}

func createTables() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			bio TEXT DEFAULT '',
			avatar TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS nuis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(name, user_id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		);

		CREATE TABLE IF NOT EXISTS photos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT NOT NULL,
			thumbnail TEXT,
			user_id INTEGER NOT NULL,
			description TEXT,
			taken_at DATE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);

		CREATE TABLE IF NOT EXISTS photo_nuis (
			photo_id INTEGER NOT NULL,
			nui_id INTEGER NOT NULL,
			PRIMARY KEY (photo_id, nui_id),
			FOREIGN KEY (photo_id) REFERENCES photos(id) ON DELETE CASCADE,
			FOREIGN KEY (nui_id) REFERENCES nuis(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS favorites (
			user_id INTEGER NOT NULL,
			photo_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, photo_id),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (photo_id) REFERENCES photos(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS likes (
			user_id INTEGER NOT NULL,
			photo_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, photo_id),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (photo_id) REFERENCES photos(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			photo_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (photo_id) REFERENCES photos(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS follows (
			follower_id INTEGER NOT NULL,
			following_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (follower_id, following_id),
			FOREIGN KEY (follower_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (following_id) REFERENCES users(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			actor_id INTEGER NOT NULL,
			type TEXT NOT NULL,
			photo_id INTEGER,
			comment_id INTEGER,
			read INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (actor_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (photo_id) REFERENCES photos(id) ON DELETE CASCADE,
			FOREIGN KEY (comment_id) REFERENCES comments(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS albums (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);

		CREATE TABLE IF NOT EXISTS album_photos (
			album_id INTEGER NOT NULL,
			photo_id INTEGER NOT NULL,
			position INTEGER DEFAULT 0,
			PRIMARY KEY (album_id, photo_id),
			FOREIGN KEY (album_id) REFERENCES albums(id) ON DELETE CASCADE,
			FOREIGN KEY (photo_id) REFERENCES photos(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_likes_photo_id ON likes(photo_id);
		CREATE INDEX IF NOT EXISTS idx_comments_photo_id ON comments(photo_id);
		CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
		CREATE INDEX IF NOT EXISTS idx_follows_follower_id ON follows(follower_id);
		CREATE INDEX IF NOT EXISTS idx_follows_following_id ON follows(following_id);
	`)
	if err != nil {
		return err
	}

	migrateToMultiNui()
	migrateUserProfile()

	return nil
}

func migrateToMultiNui() {
	var columnExists int
	err := DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('photos') WHERE name='nui_id'`).Scan(&columnExists)
	if err != nil || columnExists == 0 {
		return
	}

	rows, err := DB.Query(`SELECT id, nui_id FROM photos WHERE nui_id IS NOT NULL`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var photoID, nuiID int64
		if err := rows.Scan(&photoID, &nuiID); err != nil {
			continue
		}
		DB.Exec(`INSERT OR IGNORE INTO photo_nuis (photo_id, nui_id) VALUES (?, ?)`, photoID, nuiID)
	}

	DB.Exec(`ALTER TABLE photos DROP COLUMN nui_id`)
}

func migrateUserProfile() {
	var bioExists int
	err := DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('users') WHERE name='bio'`).Scan(&bioExists)
	if err != nil || bioExists > 0 {
		return
	}

	DB.Exec(`ALTER TABLE users ADD COLUMN bio TEXT DEFAULT ''`)
	DB.Exec(`ALTER TABLE users ADD COLUMN avatar TEXT DEFAULT ''`)
}
