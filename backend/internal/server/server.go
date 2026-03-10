package server

import (
	"database/sql"
	"fmt"
	"net/http"

	"nuistagram/internal/cache"
	"nuistagram/internal/config"
	"nuistagram/internal/database"
	jwtpkg "nuistagram/internal/jwt"
	"nuistagram/internal/middleware"
	"nuistagram/internal/monitoring/logging"
	"nuistagram/internal/monitoring/metrics"
	"nuistagram/internal/monitoring/tracing"
	"nuistagram/internal/ratelimit"
	"nuistagram/internal/repository"
	"nuistagram/internal/storage"
)

type Server struct {
	Repos   *repository.Repositories
	JWT     *jwtpkg.Manager
	Config  *config.Config
	DB      *sql.DB
	Storage storage.Storage
	Limits  *ratelimit.Limiter
}

func New(cfg *config.Config) (*Server, error) {
	env := cfg.Environment
	if env == "" {
		env = "development"
	}

	logging.Init(env, nil)
	logging.Info("Starting NUIstagram", "env", env)

	metrics.Init()
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

	c := cache.New(5 * 1000_000_000)
	repos := repository.NewRepositories(db, c)

	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required but not set")
	}
	jwtMgr, err := jwtpkg.NewManager(cfg.JWT)
	if err != nil {
		return nil, fmt.Errorf("init jwt: %w", err)
	}

	stor, err := storage.New(cfg.Storage)
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
		Limits:  ratelimit.New(),
	}, nil
}

func (s *Server) Run() error {
	addr := fmt.Sprintf(":%d", s.Config.Server.Port)

	mux := http.NewServeMux()
	s.registerHealthRoutes(mux)

	s.registerAPIRoutes(mux)
	s.registerFormRoutes(mux)
	s.registerAdminRoutes(mux)

	handler := middleware.Chain(mux,
		middleware.Recovery,
		middleware.RequestID,
		middleware.Metrics,
		middleware.Logging,
		middleware.Tracing,
	)

	logging.Info("Server starting", "addr", addr)
	return http.ListenAndServe(addr, handler)
}
