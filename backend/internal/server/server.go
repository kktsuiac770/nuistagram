package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"nuistagram/internal/cache"
	"nuistagram/internal/config"
	"nuistagram/internal/database"
	jwtpkg "nuistagram/internal/jwt"
	"nuistagram/internal/middleware"
	"nuistagram/internal/monitoring/logging"
	"nuistagram/internal/monitoring/tracing"
	"nuistagram/internal/repository"
	"nuistagram/internal/storage"
)

type Server struct {
	Repos   *repository.Repositories
	JWT     *jwtpkg.Manager
	Config  *config.Config
	DB      *sql.DB
	Storage storage.Storage
}

func New(cfg *config.Config) (*Server, error) {
	env := cfg.Environment
	if env == "" {
		env = "development"
	}

	logging.Init(env, nil)
	logging.Info("Starting NUIstagram", "env", env)

	tracing.Init("nuistagram", nil)

	pool := database.PoolConfig{
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
	}

	db, err := database.Init(cfg.Database.URL, pool)
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}
	logging.Info("Database initialized")

	c := cache.New(5 * 60 * 1000000000)
	repos := repository.NewRepositories(db, c)

	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required but not set")
	}
	jwtMgr, err := jwtpkg.NewManager(cfg.JWT.Secret, cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("init jwt: %w", err)
	}

	stor, err := storage.New(
		cfg.Storage.Provider,
		cfg.Storage.UploadDir,
		cfg.Storage.CloudName,
		cfg.Storage.APIKey,
		cfg.Storage.APISecret,
		cfg.Storage.Folder,
	)
	if err != nil {
		return nil, fmt.Errorf("init storage: %w", err)
	}
	logging.Info("Storage initialized", "provider", cfg.Storage.Provider)

	return &Server{
		Repos:   repos,
		JWT:     jwtMgr,
		Config:  cfg,
		DB:      db,
		Storage: stor,
	}, nil
}

func (s *Server) Run() error {
	reactDist := s.Config.Paths.ReactDist
	staticDir := s.Config.Paths.Static
	indexPath := filepath.Join(reactDist, "index.html")
	addr := fmt.Sprintf(":%d", s.Config.Server.Port)

	mux := http.NewServeMux()
	s.registerHealthRoutes(mux)

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	s.registerAPIRoutes(mux)
	s.registerFormRoutes(mux)

	logging.Info("Serving React app", "path", reactDist)
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(reactDist+"/assets"))))
	mux.Handle("/", &spaHandler{staticPath: reactDist, indexPath: indexPath})

	handler := middleware.Chain(mux,
		middleware.Recovery,
		middleware.RequestID,
		middleware.Logging,
		middleware.Tracing,
	)

	logging.Info("Server starting", "addr", addr)
	return http.ListenAndServe(addr, handler)
}

type spaHandler struct {
	staticPath string
	indexPath  string
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(h.staticPath, r.URL.Path)
	_, err := os.Stat(path)
	if os.IsNotExist(err) || err != nil {
		http.ServeFile(w, r, h.indexPath)
		return
	}
	http.ServeFile(w, r, path)
}
