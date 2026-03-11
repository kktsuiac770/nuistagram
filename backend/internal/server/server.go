package server

import (
	"database/sql"
	"fmt"
	"net/http"

	"nuistagram/internal/admin"
	"nuistagram/internal/auth"
	"nuistagram/internal/cache"
	"nuistagram/internal/comments"
	"nuistagram/internal/config"
	"nuistagram/internal/database"
	"nuistagram/internal/follows"
	"nuistagram/internal/health"
	jwtpkg "nuistagram/internal/jwt"
	"nuistagram/internal/likes"
	"nuistagram/internal/middleware"
	"nuistagram/internal/monitoring/logging"
	"nuistagram/internal/monitoring/metrics"
	"nuistagram/internal/monitoring/tracing"
	"nuistagram/internal/notifications"
	"nuistagram/internal/photos"
	"nuistagram/internal/ratelimit"
	"nuistagram/internal/repository"
	"nuistagram/internal/storage"
	"nuistagram/internal/users"
)

// Server is the composition root: it wires all domain handler packages together
// and owns the HTTP server lifecycle.
type Server struct {
	auth          *auth.Handler
	photos        *photos.Handler
	users         *users.Handler
	comments      *comments.Handler
	likes         *likes.Handler
	follows       *follows.Handler
	notifications *notifications.Handler
	admin         *admin.Handler
	health        *health.Handler
	config        *config.Config
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

	limits := ratelimit.New()

	authHandler := auth.New(repos.Users, jwtMgr)

	return &Server{
		auth: authHandler,
		photos: photos.New(
			authHandler, repos.Users, repos.Photos, repos.Nuis,
			repos.Albums, repos.Favorites, stor, cfg, limits,
		),
		users: users.New(
			authHandler, repos.Users, repos.Photos, stor, cfg, limits,
		),
		comments: comments.New(
			authHandler, repos.Comments, repos.Photos, repos.Notifications, cfg, limits,
		),
		likes: likes.New(
			authHandler, repos.Likes, repos.Photos, repos.Notifications, cfg, limits,
		),
		follows: follows.New(
			authHandler, repos.Users, repos.Follows, repos.Notifications,
		),
		notifications: notifications.New(authHandler, repos.Notifications),
		admin:         admin.New(authHandler, repos.Users),
		health:        health.New(db),
		config:        cfg,
	}, nil
}

// NewForTest creates a Server with pre-initialised dependencies, intended for
// integration / E2E tests that already have their own DB and JWT manager.
func NewForTest(db *sql.DB, jwtMgr *jwtpkg.Manager, cfg *config.Config) *Server {
	c := cache.New(5 * 60 * 1000_000_000)
	repos := repository.NewRepositories(db, c)
	limits := ratelimit.New()
	stor, _ := storage.New(cfg.Storage) // defaults to local; errors surfaced on first use
	authHandler := auth.New(repos.Users, jwtMgr)
	return &Server{
		auth: authHandler,
		photos: photos.New(
			authHandler, repos.Users, repos.Photos, repos.Nuis,
			repos.Albums, repos.Favorites, stor, cfg, limits,
		),
		users: users.New(
			authHandler, repos.Users, repos.Photos, stor, cfg, limits,
		),
		comments: comments.New(
			authHandler, repos.Comments, repos.Photos, repos.Notifications, cfg, limits,
		),
		likes: likes.New(
			authHandler, repos.Likes, repos.Photos, repos.Notifications, cfg, limits,
		),
		follows: follows.New(
			authHandler, repos.Users, repos.Follows, repos.Notifications,
		),
		notifications: notifications.New(authHandler, repos.Notifications),
		admin:         admin.New(authHandler, repos.Users),
		health:        health.New(db),
		config:        cfg,
	}
}

// Auth exposes the auth handler (used by E2E test setup to register routes).
func (s *Server) Auth() *auth.Handler { return s.auth }

// Photos exposes the photos handler.
func (s *Server) Photos() *photos.Handler { return s.photos }

// Users exposes the users handler.
func (s *Server) Users() *users.Handler { return s.users }

// Likes exposes the likes handler.
func (s *Server) Likes() *likes.Handler { return s.likes }

func (s *Server) Run() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)

	mux := http.NewServeMux()
	s.health.RegisterRoutes(mux)
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
