package main

import (
	"fmt"
	"log"
	"nuistagram/internal/config"
	"nuistagram/internal/monitoring/logging"
	"nuistagram/internal/server"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	cfg, err := config.Load("config.yaml")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to load config:", err)
		os.Exit(1)
	}

	srv, err := server.New(cfg)
	if err != nil {
		logging.Error("Failed to initialize server", "error", err)
		os.Exit(1)
	}

	if err := srv.Run(); err != nil {
		logging.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
