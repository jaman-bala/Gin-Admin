// Package config handles the loading and validation of application configuration.
package config

import (
	"fmt"
	"os"
	"strconv"
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
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
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
	_ = godotenv.Load() // Игнорируем ошибку, если .env файл не найден

	config := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  time.Second * time.Duration(getEnvAsInt("SERVER_READ_TIMEOUT", 10)),
			WriteTimeout: time.Second * time.Duration(getEnvAsInt("SERVER_WRITE_TIMEOUT", 10)),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", ""),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", ""),
			Password:        getEnv("DB_PASSWORD", ""),
			Name:            getEnv("DB_NAME", ""),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 25),
			ConnMaxLifetime: time.Hour * time.Duration(getEnvAsInt("DB_CONN_MAX_LIFETIME_HOURS", 1)),
		},
		JWT: JWTConfig{
			Secret:        getEnv("SECRET_KEY", ""),
			Expiry:        time.Hour * time.Duration(getEnvAsInt("JWT_EXPIRY_HOURS", 24)),
			RefreshExpiry: time.Hour * time.Duration(getEnvAsInt("JWT_REFRESH_EXPIRY_HOURS", 168)),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", ""),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Minio: MinioConfig{
			MinioHost:       getEnv("MINIO_HOST", ""),
			MinioPublicHost: getEnv("MINIO_PUBLIC_HOST", "localhost:9000"),
			MinioBucket:     getEnv("MINIO_BUCKET_NAME", ""),
			MinioPort:       getEnv("MINIO_PORT", "9000"),
			MinioAccessKey:  getEnv("MINIO_ACCESS_KEY", ""),
			MinioSecretKey:  getEnv("MINIO_SECRET_KEY", ""),
			MinioSSL:        getEnvAsBool("MINIO_SSL", false),
		},
	}

	// Валидация конфигурации
	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// validate проверяет корректность конфигурации
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
