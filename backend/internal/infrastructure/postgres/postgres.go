// Package postgres provides the implementation of the database repositories.
package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"gin_auth_service/config"
	_ "gin_auth_service/docs" // swagger documentation

	"gin_auth_service/internal/pkg/hash"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // postgres driver
	goose "github.com/pressly/goose/v3"
)

// TransactionManager provides transaction management for database operations.
type TransactionManager struct {
	db *sqlx.DB
}

// NewTransactionManager creates a new transaction manager.
func NewTransactionManager(db *sqlx.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

// BeginTx starts a new transaction.
func (tm *TransactionManager) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return tm.db.BeginTxx(ctx, nil)
}

// WithTx executes a function within a transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, it is committed.
func (tm *TransactionManager) WithTx(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	tx, err := tm.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p) // re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx error: %v, rollback error: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit error: %w", err)
	}

	return nil
}

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
	hashedPassword, err := hash.HashPassword("Password123")
	if err != nil {
		slog.Error("Error hashing password", "error", err)
		return
	}

	query := `
		INSERT INTO users (phone, password, role, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (phone) DO UPDATE SET password = EXCLUDED.password
	`
	_, err = db.Exec(query, "+996500500500", hashedPassword, "superuser", true)
	if err != nil {
		slog.Error("Error creating admin", "error", err)
	}
}
