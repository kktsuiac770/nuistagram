package main

import (
	"fmt"
	"net/http"
	"nuistagram/internal/config"
	"nuistagram/internal/database"
	"nuistagram/internal/handlers"
	"nuistagram/internal/logging"
	"nuistagram/internal/metrics"
	"nuistagram/internal/middleware"
	"nuistagram/internal/tracing"
	"os"
	"path/filepath"
)

type SPAHandler struct {
	staticPath string
	indexPath  string
}

func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(h.staticPath, r.URL.Path)
	_, err := os.Stat(path)
	if os.IsNotExist(err) || err != nil {
		http.ServeFile(w, r, h.indexPath)
		return
	}
	http.ServeFile(w, r, path)
}

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to load config:", err)
		os.Exit(1)
	}

	env := "development"
	if cfg.Environment != "" {
		env = cfg.Environment
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

	if err := database.Init(cfg.Database.Path, pool); err != nil {
		logging.Error("Failed to init database", "error", err)
		os.Exit(1)
	}
	logging.Info("Database initialized")

	handlers.Init()
	handlers.SetSecureCookie(cfg.Security.SecureCookie)

	reactDist := cfg.Paths.ReactDist
	staticDir := cfg.Paths.Static
	indexPath := filepath.Join(reactDist, "index.html")
	addr := fmt.Sprintf(":%d", cfg.Server.Port)

	if _, err := os.Stat(indexPath); err == nil {
		logging.Info("Serving React app", "path", reactDist)
		startReactServer(addr, reactDist, staticDir, indexPath)
	} else {
		logging.Info("React dist not found, serving HTML templates")
		startTemplateServer(addr, staticDir)
	}
}

func chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

func startReactServer(addr, reactDist, staticDir, indexPath string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", handlers.Healthz)
	mux.HandleFunc("/readyz", handlers.Readyz)
	mux.HandleFunc("/metrics", metrics.Handler)

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(reactDist+"/assets"))))

	mux.HandleFunc("/api/photos", handlers.APIGetPhotos)
	mux.HandleFunc("/api/photo/{id}", handlers.APIGetPhoto)
	mux.HandleFunc("/api/photo/{id}/favorite", handlers.CSRFMiddleware(handlers.APIToggleFavorite))
	mux.HandleFunc("/api/photo/{id}/like", handlers.CSRFMiddleware(handlers.APIToggleLike))
	mux.HandleFunc("/api/photo/{id}/likers", handlers.APIGetLikers)
	mux.HandleFunc("/api/photo/{id}/comments", handlers.APIGetComments)
	mux.HandleFunc("POST /api/photo/{id}/comment", handlers.CSRFMiddleware(handlers.APICreateComment))
	mux.HandleFunc("/api/comment/{id}", handlers.CSRFMiddleware(handlers.APIDeleteComment))
	mux.HandleFunc("/api/me", handlers.APIGetMe)
	mux.HandleFunc("/api/nuis", handlers.APIGetNuis)
	mux.HandleFunc("/api/csrf-token", handlers.GetCSRFToken)
	mux.HandleFunc("/api/user/{username}", handlers.APIGetUser)
	mux.HandleFunc("/api/user/{username}/photos", handlers.APIGetUserPhotos)
	mux.HandleFunc("POST /api/user/{username}/follow", handlers.CSRFMiddleware(handlers.APIFollowByUsername))
	mux.HandleFunc("POST /api/user/{username}/unfollow", handlers.CSRFMiddleware(handlers.APIUnfollowByUsername))
	mux.HandleFunc("/api/search/users", handlers.APISearchUsers)
	mux.HandleFunc("POST /api/profile", handlers.CSRFMiddleware(handlers.APIUpdateProfile))
	mux.HandleFunc("POST /api/avatar", handlers.CSRFMiddleware(handlers.APIUploadAvatar))
	mux.HandleFunc("/api/notifications", handlers.APIGetNotifications)
	mux.HandleFunc("/api/notifications/unread", handlers.APIUnreadCount)
	mux.HandleFunc("POST /api/notifications/{id}/read", handlers.CSRFMiddleware(handlers.APIMarkNotificationRead))
	mux.HandleFunc("POST /api/notifications/read-all", handlers.CSRFMiddleware(handlers.APIMarkAllNotificationsRead))

	mux.HandleFunc("POST /login", handlers.RateLimitLogin(handlers.Login))
	mux.HandleFunc("POST /register", handlers.RateLimitLogin(handlers.Register))
	mux.HandleFunc("/logout", handlers.Logout)
	mux.HandleFunc("POST /upload", handlers.CSRFMiddleware(handlers.Upload))
	mux.HandleFunc("POST /photo/{id}/delete", handlers.CSRFMiddleware(handlers.DeletePhoto))
	mux.HandleFunc("POST /photo/{id}/favorite", handlers.CSRFMiddleware(handlers.ToggleFavorite))
	mux.HandleFunc("/export", handlers.ExportPhotos)

	mux.Handle("/", &SPAHandler{staticPath: reactDist, indexPath: indexPath})

	handler := chain(mux,
		middleware.Recovery,
		middleware.RequestID,
		middleware.Logging,
		metrics.Middleware,
		middleware.Tracing,
	)

	logging.Info("Server starting", "addr", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		logging.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

func startTemplateServer(addr, staticDir string) {
	if err := handlers.InitTemplates(); err != nil {
		logging.Error("Failed to init templates", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", handlers.Healthz)
	mux.HandleFunc("/readyz", handlers.Readyz)
	mux.HandleFunc("/metrics", metrics.Handler)

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	mux.HandleFunc("/api/photos", handlers.APIGetPhotos)
	mux.HandleFunc("/api/photo/{id}", handlers.APIGetPhoto)
	mux.HandleFunc("/api/photo/{id}/favorite", handlers.CSRFMiddleware(handlers.APIToggleFavorite))
	mux.HandleFunc("/api/photo/{id}/like", handlers.CSRFMiddleware(handlers.APIToggleLike))
	mux.HandleFunc("/api/photo/{id}/likers", handlers.APIGetLikers)
	mux.HandleFunc("/api/photo/{id}/comments", handlers.APIGetComments)
	mux.HandleFunc("/api/me", handlers.APIGetMe)
	mux.HandleFunc("/api/nuis", handlers.APIGetNuis)
	mux.HandleFunc("/api/user/{username}", handlers.APIGetUser)
	mux.HandleFunc("/api/user/{username}/photos", handlers.APIGetUserPhotos)
	mux.HandleFunc("/api/user/{username}/follow-status", handlers.APIGetFollowStatus)

	mux.HandleFunc("/", handlers.Home)
	mux.HandleFunc("/login", handlers.RateLimitLogin(handlers.Login))
	mux.HandleFunc("/register", handlers.RateLimitLogin(handlers.Register))
	mux.HandleFunc("/logout", handlers.Logout)
	mux.HandleFunc("/upload", handlers.CSRFMiddleware(handlers.Upload))
	mux.HandleFunc("/photo/{id}/delete", handlers.CSRFMiddleware(handlers.DeletePhoto))
	mux.HandleFunc("/photo/{id}/favorite", handlers.CSRFMiddleware(handlers.ToggleFavorite))
	mux.HandleFunc("/export", handlers.ExportPhotos)
	mux.HandleFunc("/nui/{name}", handlers.NuiProfile)
	mux.HandleFunc("/user/{username}", handlers.UserProfile)
	mux.HandleFunc("/favorites", handlers.Favorites)
	mux.HandleFunc("/photo/{id}", handlers.ViewPhoto)
	mux.HandleFunc("/photo/{id}/edit", handlers.CSRFMiddleware(handlers.EditPhoto))
	mux.HandleFunc("/albums", handlers.ListAlbums)
	mux.HandleFunc("/albums/new", handlers.CSRFMiddleware(handlers.CreateAlbum))
	mux.HandleFunc("/album/{id}", handlers.ViewAlbum)
	mux.HandleFunc("/album/{id}/delete", handlers.CSRFMiddleware(handlers.DeleteAlbum))

	handler := chain(mux,
		middleware.Recovery,
		middleware.RequestID,
		middleware.Logging,
		metrics.Middleware,
		middleware.Tracing,
	)

	logging.Info("Server starting", "addr", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		logging.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
