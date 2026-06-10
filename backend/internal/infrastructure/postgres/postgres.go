// Package postgres provides the implementation of the database repositories.
package postgres

import (
	"fmt"
	"log/slog"
	"os"

	"gin_auth_service/config"
	_ "gin_auth_service/docs" // swagger documentation

	"gin_auth_service/internal/pkg/hash"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // postgres driver
	goose "github.com/pressly/goose/v3"
)

// InitDB initializes the database connection and optionally applies migrations.
func InitDB(cfg *config.Config, runMigrations bool) (*sqlx.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to db: %w", err)
	}

	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	if runMigrations {
		if err := goose.SetDialect("postgres"); err != nil {
			return nil, fmt.Errorf("failed to set goose dialect: %w", err)
		}

		// Applying migrations
		slog.Info("Applying database migrations...")
		if err := goose.Up(db.DB, "migrations"); err != nil {
			return nil, fmt.Errorf("failed to apply migrations: %w", err)
		}
		slog.Info("Migrations applied successfully.")

		createDefaultAdmin(db)
	}

	return db, nil
}

func createDefaultAdmin(db *sqlx.DB) {
	phone := os.Getenv("ADMIN_DEFAULT_PHONE")
	if phone == "" {
		phone = "+996500500500"
	}
	password := os.Getenv("ADMIN_DEFAULT_PASSWORD")
	if password == "" {
		slog.Warn("ADMIN_DEFAULT_PASSWORD not set — skipping default admin creation")
		return
	}

	hashedPassword, err := hash.HashPassword(password)
	if err != nil {
		slog.Error("Error hashing admin password", "error", err)
		return
	}

	const query = `
		INSERT INTO users (phone, password, role, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (phone) DO NOTHING
	`
	result, err := db.Exec(query, phone, hashedPassword, "superuser", true)
	if err != nil {
		slog.Error("Error creating default admin", "error", err)
		return
	}

	if rows, _ := result.RowsAffected(); rows > 0 {
		slog.Info("Default admin created", "phone", phone)
	} else {
		slog.Info("Default admin already exists, skipping", "phone", phone)
	}
}
