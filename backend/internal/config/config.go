package config

import (
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Environment string         `yaml:"environment"`
	Server      ServerConfig   `yaml:"server"`
	Database    DatabaseConfig `yaml:"database"`
	Security    SecurityConfig `yaml:"security"`
	Paths       PathsConfig    `yaml:"paths"`
	Storage     StorageConfig  `yaml:"storage"`
	JWT         JWTConfig      `yaml:"jwt"`
}

type JWTConfig struct {
	Secret          string        `yaml:"secret"`
	AccessTokenTTL  time.Duration `yaml:"access_token_ttl"`
	RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type DatabaseConfig struct {
	URL             string        `yaml:"url"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
}

type SecurityConfig struct {
	SecureCookie bool `yaml:"secure_cookie"`
}

type PathsConfig struct {
	ReactDist string `yaml:"react_dist"`
	Static    string `yaml:"static"`
}

type StorageConfig struct {
	// Provider selects the storage backend: "local" (default) or "cloudinary".
	Provider string `yaml:"provider"`
	// UploadDir is the local filesystem directory for uploads (local only).
	UploadDir string `yaml:"upload_dir"`
	// Cloudinary credentials (cloudinary provider only).
	CloudName string `yaml:"cloud_name"`
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
	// Folder is the Cloudinary path prefix for all uploaded assets.
	Folder string `yaml:"folder"`
}

var defaultConfig = Config{
	Environment: "development",
	Server: ServerConfig{
		Port: 8080,
	},
	Database: DatabaseConfig{
		URL:             "",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,
	},
	Security: SecurityConfig{
		SecureCookie: false,
	},
	Paths: PathsConfig{
		ReactDist: "frontend/dist",
		Static:    "static",
	},
	Storage: StorageConfig{
		Provider:  "local",
		UploadDir: "static/uploads",
		Folder:    "nuistagram",
	},
	JWT: JWTConfig{
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	},
}

func Load(path string) (*Config, error) {
	cfg := defaultConfig

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			applyEnvOverrides(&cfg)
			return &cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	applyEnvOverrides(&cfg)

	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if env := os.Getenv("ENV"); env != "" {
		cfg.Environment = env
	}

	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.Database.URL = dbURL
	}

	if secureCookie := os.Getenv("SECURE_COOKIE"); secureCookie != "" {
		if val, err := strconv.ParseBool(secureCookie); err == nil {
			cfg.Security.SecureCookie = val
		}
	}

	if provider := os.Getenv("STORAGE_PROVIDER"); provider != "" {
		cfg.Storage.Provider = provider
	}
	if uploadDir := os.Getenv("STORAGE_UPLOAD_DIR"); uploadDir != "" {
		cfg.Storage.UploadDir = uploadDir
	}
	if cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME"); cloudName != "" {
		cfg.Storage.CloudName = cloudName
	}
	if apiKey := os.Getenv("CLOUDINARY_API_KEY"); apiKey != "" {
		cfg.Storage.APIKey = apiKey
	}
	if apiSecret := os.Getenv("CLOUDINARY_API_SECRET"); apiSecret != "" {
		cfg.Storage.APISecret = apiSecret
	}
	if folder := os.Getenv("CLOUDINARY_FOLDER"); folder != "" {
		cfg.Storage.Folder = folder
	}

	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.JWT.Secret = secret
	}
	if ttl := os.Getenv("JWT_ACCESS_TTL"); ttl != "" {
		if d, err := time.ParseDuration(ttl); err == nil {
			cfg.JWT.AccessTokenTTL = d
		}
	}
	if ttl := os.Getenv("JWT_REFRESH_TTL"); ttl != "" {
		if d, err := time.ParseDuration(ttl); err == nil {
			cfg.JWT.RefreshTokenTTL = d
		}
	}
}
