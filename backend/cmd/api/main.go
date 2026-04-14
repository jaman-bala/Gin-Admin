// Package main is the entry point for the server application.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gin_auth_service/config"
	infraHttp "gin_auth_service/internal/infrastructure/http"
	"gin_auth_service/internal/infrastructure/postgres"
	"gin_auth_service/internal/pkg/logger"
	"gin_auth_service/internal/pkg/validator"

	_ "gin_auth_service/docs"
)

// @title AUTH SERVICE API
// @version 1.0
// @description API for managing users and authentication in the Auth Service application.
// @host localhost:8082
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter: Bearer {your_jwt_token} (e.g., Bearer eyJhbGciOiJIUzI1NiIs...)
func main() {
	// Initialize logger
	isProd := os.Getenv("GIN_MODE") == "release"
	logger.InitLogger(isProd)

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Error loading config", "error", err)
		os.Exit(1)
	}

	// Connect to Database
	db, err := postgres.InitDB(cfg, true)
	if err != nil {
		slog.Error("Error connecting to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		slog.Info("Closing database connection")
		_ = db.Close()
	}()

	// Init custom validators
	validator.InitCustomValidators()

	// Setup HTTP routing
	router := infraHttp.SetupRoutes(db, cfg)

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Run server in a goroutine
	go func() {
		slog.Info("Server starting", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server listen error", "error", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server gracefully...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exiting")
}
