package database

import (
	"database/sql"
	"os"
	"path/filepath"

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

	return createTables()
}

func createTables() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
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
	`)
	if err != nil {
		return err
	}

	migrateToMultiNui()

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
