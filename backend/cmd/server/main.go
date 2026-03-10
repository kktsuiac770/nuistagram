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
	// Load environment variables from .env file (optional in containerised deployments)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
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
