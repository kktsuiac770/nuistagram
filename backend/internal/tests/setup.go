package tests

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"nuistagram/internal/cache"
	"nuistagram/internal/database"
	jwtpkg "nuistagram/internal/jwt"
	"nuistagram/internal/repository"
	"nuistagram/internal/server"

	_ "github.com/lib/pq"
)

type TestServer struct {
	DB     *sql.DB
	Repos  *repository.Repositories
	Srv    *server.Server
	Server *httptest.Server
	Client *http.Client
}

func SetupTestServer(mux *http.ServeMux) (*TestServer, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required for E2E tests")
	}

	pool := database.PoolConfig{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,
	}
	db, err := database.Init(dbURL, pool)
	if err != nil {
		return nil, err
	}

	// Clean all tables before each test run
	if _, err := db.Exec(`TRUNCATE TABLE notifications, album_photos, albums, comments, likes, favorites, photo_nuis, photos, follows, nuis, users RESTART IDENTITY CASCADE`); err != nil {
		db.Close()
		return nil, fmt.Errorf("truncate tables: %w", err)
	}

	c := cache.New(5 * 60 * 1000000000)
	repos := repository.NewRepositories(db, c)

	jwtMgr, err := jwtpkg.NewManager("test-secret-key-for-e2e-tests-32b", 15*time.Minute, 7*24*time.Hour)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("init jwt: %w", err)
	}

	srv := &server.Server{
		Repos: repos,
		JWT:   jwtMgr,
		DB:    db,
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
	ts.DB.Exec(`TRUNCATE TABLE notifications, album_photos, albums, comments, likes, favorites, photo_nuis, photos, follows, nuis, users RESTART IDENTITY CASCADE`)
	ts.DB.Close()
}

func (ts *TestServer) URL() string {
	return ts.Server.URL
}

func CreateTestUser(db *sql.DB, username, passwordHash string) (int64, error) {
	var id int64
	err := db.QueryRow(
		"INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id",
		username, passwordHash,
	).Scan(&id)
	return id, err
}

func CreateTestPhoto(db *sql.DB, userID int64, filename, description string) (int64, error) {
	var id int64
	err := db.QueryRow(
		"INSERT INTO photos (filename, user_id, description) VALUES ($1, $2, $3) RETURNING id",
		filename, userID, description,
	).Scan(&id)
	return id, err
}

func CreateTestNui(db *sql.DB, name string, userID int64) (int64, error) {
	var id int64
	err := db.QueryRow(
		"INSERT INTO nuis (name, user_id) VALUES ($1, $2) RETURNING id",
		name, userID,
	).Scan(&id)
	return id, err
}

func LinkPhotoNui(db *sql.DB, photoID, nuiID int64) error {
	_, err := db.Exec(
		"INSERT INTO photo_nuis (photo_id, nui_id) VALUES ($1, $2)",
		photoID, nuiID,
	)
	return err
}

func GetLikeCount(db *sql.DB, photoID int64) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM likes WHERE photo_id = $1", photoID).Scan(&count)
	return count, err
}

func GetFollowCount(db *sql.DB, followerID, followingID int64) (int, error) {
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM follows WHERE follower_id = $1 AND following_id = $2",
		followerID, followingID,
	).Scan(&count)
	return count, err
}
