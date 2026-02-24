package main

import (
	"log"
	"net/http"
	"nuistagram/internal/database"
	"nuistagram/internal/handlers"
)

func main() {
	if err := database.Init("data/nuistagram.db"); err != nil {
		log.Fatal("Failed to init database:", err)
	}

	if err := handlers.InitTemplates(); err != nil {
		log.Fatal("Failed to init templates:", err)
	}

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", handlers.Home)
	http.HandleFunc("/nui/{name}", handlers.NuiProfile)
	http.HandleFunc("/user/{username}", handlers.UserProfile)
	http.HandleFunc("/favorites", handlers.Favorites)
	http.HandleFunc("/photo/{id}", handlers.ViewPhoto)
	http.HandleFunc("/photo/{id}/edit", handlers.EditPhoto)
	http.HandleFunc("/photo/{id}/delete", handlers.DeletePhoto)
	http.HandleFunc("/photo/{id}/favorite", handlers.ToggleFavorite)
	http.HandleFunc("/upload", handlers.Upload)
	http.HandleFunc("/login", handlers.Login)
	http.HandleFunc("/register", handlers.Register)
	http.HandleFunc("/logout", handlers.Logout)
	http.HandleFunc("/albums", handlers.ListAlbums)
	http.HandleFunc("/albums/new", handlers.CreateAlbum)
	http.HandleFunc("/album/{id}", handlers.ViewAlbum)
	http.HandleFunc("/album/{id}/delete", handlers.DeleteAlbum)
	http.HandleFunc("/export", handlers.ExportPhotos)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
