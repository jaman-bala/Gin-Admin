// Package config handles the loading and validation of application configuration.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Redis    RedisConfig
	Minio    MinioConfig
}

type ServerConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	MaxBodyBytes    int64
	RunMigrations   bool
	CORSOrigins     []string
	MetricsToken    string
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type JWTConfig struct {
	Secret        string
	Expiry        time.Duration
	RefreshExpiry time.Duration
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type MinioConfig struct {
	MinioHost       string
	MinioPublicHost string // Public endpoint for browser URLs (e.g., localhost:9000)
	MinioBucket     string
	MinioPort       string
	MinioAccessKey  string
	MinioSecretKey  string
	MinioSSL        bool
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	config := &Config{
		Server: ServerConfig{
			Port:        getEnv("SERVER_PORT", "8080"),
			ReadTimeout: time.Second * time.Duration(getEnvAsInt("SERVER_READ_TIMEOUT", 60)),
			// WriteTimeout must stay above the pprof CPU-profile window
			// (/debug/pprof/profile?seconds=30 by default).
			WriteTimeout:    time.Second * time.Duration(getEnvAsInt("SERVER_WRITE_TIMEOUT", 60)),
			ShutdownTimeout: time.Second * time.Duration(getEnvAsInt("SERVER_SHUTDOWN_TIMEOUT", 30)),
			MaxBodyBytes:    int64(getEnvAsInt("SERVER_MAX_BODY_MB", 10)) << 20,
			RunMigrations:   getEnvAsBool("RUN_MIGRATIONS", false),
			CORSOrigins:     splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000,http://localhost:8085")),
			MetricsToken:    readSecret("METRICS_TOKEN", ""),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", ""),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", ""),
			Password:        readSecret("DB_PASSWORD", ""),
			Name:            getEnv("DB_NAME", ""),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 25),
			ConnMaxLifetime: time.Hour * time.Duration(getEnvAsInt("DB_CONN_MAX_LIFETIME_HOURS", 1)),
		},
		JWT: JWTConfig{
			Secret: readSecret("SECRET_KEY", ""),
			// Access tokens are short-lived; clients refresh them via
			// /auth/refresh. JWT_EXPIRY_HOURS is kept for backward compat.
			Expiry:        jwtAccessExpiry(),
			RefreshExpiry: time.Hour * time.Duration(getEnvAsInt("JWT_REFRESH_EXPIRY_HOURS", 168)),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", ""),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: readSecret("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Minio: MinioConfig{
			MinioHost:       getEnv("MINIO_HOST", ""),
			MinioPublicHost: getEnv("MINIO_PUBLIC_HOST", "localhost:9000"),
			MinioBucket:     getEnv("MINIO_BUCKET_NAME", ""),
			MinioPort:       getEnv("MINIO_PORT", "9000"),
			MinioAccessKey:  readSecret("MINIO_ACCESS_KEY", ""),
			MinioSecretKey:  readSecret("MINIO_SECRET_KEY", ""),
			MinioSSL:        getEnvAsBool("MINIO_SSL", false),
		},
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("SECRET_KEY is required")
	}
	if c.Minio.MinioAccessKey == "" {
		return fmt.Errorf("MINIO_ACCESS_KEY is required")
	}
	if c.Minio.MinioSecretKey == "" {
		return fmt.Errorf("MINIO_SECRET_KEY is required")
	}
	return nil
}

// readSecret reads a sensitive value from a Docker secret file if available,
// then falls back to the environment variable, then to the default.
// Secret path: /run/secrets/<lowercase_key> (e.g. DB_PASSWORD → /run/secrets/db_password)
func readSecret(key, fallback string) string {
	secretPath := "/run/secrets/" + strings.ToLower(key)
	if data, err := os.ReadFile(secretPath); err == nil {
		return strings.TrimSpace(string(data))
	}
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// jwtAccessExpiry prefers JWT_EXPIRY_MINUTES; falls back to the legacy
// JWT_EXPIRY_HOURS if set, then to 15 minutes.
func jwtAccessExpiry() time.Duration {
	if m := getEnvAsInt("JWT_EXPIRY_MINUTES", 0); m > 0 {
		return time.Minute * time.Duration(m)
	}
	if h := getEnvAsInt("JWT_EXPIRY_HOURS", 0); h > 0 {
		return time.Hour * time.Duration(h)
	}
	return 15 * time.Minute
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return fallback
}