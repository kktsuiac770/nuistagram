package main

import (
	"log"
	"net/http"
	"nuistagram/internal/database"
	"nuistagram/internal/handlers"
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
	if err := database.Init("data/nuistagram.db"); err != nil {
		log.Fatal("Failed to init database:", err)
	}

	handlers.Init()

	reactDist := "frontend/dist"
	indexPath := filepath.Join(reactDist, "index.html")

	if _, err := os.Stat(indexPath); err == nil {
		log.Println("Serving React app from", reactDist)

		mux := http.NewServeMux()

		mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
		mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(reactDist+"/assets"))))

		mux.HandleFunc("/api/photos", handlers.APIGetPhotos)
		mux.HandleFunc("/api/photo/{id}", handlers.APIGetPhoto)
		mux.HandleFunc("/api/photo/{id}/favorite", handlers.APIToggleFavorite)
		mux.HandleFunc("/api/me", handlers.APIGetMe)
		mux.HandleFunc("/api/nuis", handlers.APIGetNuis)

		mux.HandleFunc("POST /login", handlers.Login)
		mux.HandleFunc("POST /register", handlers.Register)
		mux.HandleFunc("/logout", handlers.Logout)
		mux.HandleFunc("POST /upload", handlers.Upload)
		mux.HandleFunc("POST /photo/{id}/delete", handlers.DeletePhoto)
		mux.HandleFunc("POST /photo/{id}/favorite", handlers.ToggleFavorite)
		mux.HandleFunc("/export", handlers.ExportPhotos)

		mux.Handle("/", &SPAHandler{staticPath: reactDist, indexPath: indexPath})

		log.Println("Server starting on :8080")
		log.Fatal(http.ListenAndServe(":8080", mux))
	} else {
		log.Println("React dist not found, serving HTML templates")
		if err := handlers.InitTemplates(); err != nil {
			log.Fatal("Failed to init templates:", err)
		}

		http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
		http.HandleFunc("/api/photos", handlers.APIGetPhotos)
		http.HandleFunc("/api/photo/{id}", handlers.APIGetPhoto)
		http.HandleFunc("/api/photo/{id}/favorite", handlers.APIToggleFavorite)
		http.HandleFunc("/api/me", handlers.APIGetMe)
		http.HandleFunc("/api/nuis", handlers.APIGetNuis)

		http.HandleFunc("/", handlers.Home)
		http.HandleFunc("/login", handlers.Login)
		http.HandleFunc("/register", handlers.Register)
		http.HandleFunc("/logout", handlers.Logout)
		http.HandleFunc("/upload", handlers.Upload)
		http.HandleFunc("/photo/{id}/delete", handlers.DeletePhoto)
		http.HandleFunc("/photo/{id}/favorite", handlers.ToggleFavorite)
		http.HandleFunc("/export", handlers.ExportPhotos)
		http.HandleFunc("/nui/{name}", handlers.NuiProfile)
		http.HandleFunc("/user/{username}", handlers.UserProfile)
		http.HandleFunc("/favorites", handlers.Favorites)
		http.HandleFunc("/photo/{id}", handlers.ViewPhoto)
		http.HandleFunc("/photo/{id}/edit", handlers.EditPhoto)
		http.HandleFunc("/albums", handlers.ListAlbums)
		http.HandleFunc("/albums/new", handlers.CreateAlbum)
		http.HandleFunc("/album/{id}", handlers.ViewAlbum)
		http.HandleFunc("/album/{id}/delete", handlers.DeleteAlbum)

		log.Println("Server starting on :8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}
}
