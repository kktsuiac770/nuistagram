package main

import (
	"bufio"
	"fmt"
	"nuistagram/internal/config"
	"nuistagram/internal/monitoring/logging"
	"nuistagram/internal/server"
	"os"
	"strings"
)

// loadDotEnv reads a .env file and sets environment variables for any key
// that is not already set in the process environment. Lines starting with #
// and blank lines are ignored. Values may be optionally quoted with " or '.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // .env is optional
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// Strip surrounding quotes.
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0] {
			val = val[1 : len(val)-1]
		}
		// Don't override values already set in the environment.
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func main() {
	// Load .env before config so env overrides pick up the values.
	loadDotEnv(".env")

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
