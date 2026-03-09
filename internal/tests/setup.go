package tests

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"nuistagram/internal/cache"
	"nuistagram/internal/database"
	"nuistagram/internal/repository"
	"nuistagram/internal/server"
	"nuistagram/internal/session"

	_ "github.com/mattn/go-sqlite3"
)

type TestServer struct {
	DB     *sql.DB
	Repos  *repository.Repositories
	Srv    *server.Server
	Server *httptest.Server
	Client *http.Client
}

func SetupTestServer(mux *http.ServeMux) (*TestServer, error) {
	tmpDir, err := os.MkdirTemp("", "nuistagram-test-*")
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	pool := database.PoolConfig{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,
	}
	db, err := database.Init(dbPath, pool)
	if err != nil {
		return nil, err
	}

	c := cache.New(5 * 60 * 1000000000)
	repos := repository.NewRepositories(db, c)

	srv := &server.Server{
		Repos:    repos,
		Sessions: session.NewManager(),
		DB:       db,
	}

	httpServer := httptest.NewServer(mux)
	client := httpServer.Client()

	return &TestServer{
		DB:     db,
		Repos:  repos,
		Srv:    srv,
		Server: httpServer,
		Client: client,
	}, nil
}

func (ts *TestServer) Cleanup() {
	ts.Server.Close()
	ts.DB.Close()
}

func (ts *TestServer) URL() string {
	return ts.Server.URL
}

func CreateTestUser(db *sql.DB, username, passwordHash string) (int64, error) {
	result, err := db.Exec(
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		username, passwordHash,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func CreateTestPhoto(db *sql.DB, userID int64, filename, description string) (int64, error) {
	result, err := db.Exec(
		"INSERT INTO photos (filename, user_id, description) VALUES (?, ?, ?)",
		filename, userID, description,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func CreateTestNui(db *sql.DB, name string, userID int64) (int64, error) {
	result, err := db.Exec(
		"INSERT INTO nuis (name, user_id) VALUES (?, ?)",
		name, userID,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func LinkPhotoNui(db *sql.DB, photoID, nuiID int64) error {
	_, err := db.Exec(
		"INSERT INTO photo_nuis (photo_id, nui_id) VALUES (?, ?)",
		photoID, nuiID,
	)
	return err
}

func GetLikeCount(db *sql.DB, photoID int64) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM likes WHERE photo_id = ?", photoID).Scan(&count)
	return count, err
}

func GetFollowCount(db *sql.DB, followerID, followingID int64) (int, error) {
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM follows WHERE follower_id = ? AND following_id = ?",
		followerID, followingID,
	).Scan(&count)
	return count, err
}
