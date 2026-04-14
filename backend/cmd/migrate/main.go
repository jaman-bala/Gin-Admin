package main

import (
	"flag"
	"fmt"
	"gin_auth_service/config"
	"gin_auth_service/internal/infrastructure/postgres"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

const (
	defaultMigrationDir = "migrations"
)

func main() {
	// 1. Setup flags
	flag.Usage = func() {
		fmt.Printf("Usage: %s [options] <command> [arguments]\n", os.Args[0])
		fmt.Printf("Commands:\n")
		fmt.Printf("  up                   Migrate the database to the most recent version available\n")
		fmt.Printf("  up-by-one            Migrate the database up by 1\n")
		fmt.Printf("  up-to VERSION        Migrate the database to a specific VERSION\n")
		fmt.Printf("  down                 Roll back the version by 1\n")
		fmt.Printf("  down-to VERSION      Roll back to a specific VERSION\n")
		fmt.Printf("  redo                 Re-run the latest migration\n")
		fmt.Printf("  reset                Roll back all migrations\n")
		fmt.Printf("  status               Dump the migration status for the current DB\n")
		fmt.Printf("  version              Print the current version of the database\n")
		fmt.Printf("  create NAME [type]   Creates new migration file with the current timestamp\n")
		fmt.Printf("  fix                  Apply sequential corrections to migrations (fixes existing filenames)\n")
		fmt.Printf("Options:\n")
		flag.PrintDefaults()
	}

	dir := flag.String("dir", defaultMigrationDir, "directory with migration files")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	command := args[0]

	// 2. Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// 3. Connect to database (using InitDB without auto-migrations)
	db, err := postgres.InitDB(cfg, false)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// 4. Setup goose
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("failed to set goose dialect: %v", err)
	}

	// 5. Execute command
	if err := goose.Run(command, db.DB, *dir, args[1:]...); err != nil {
		log.Fatalf("goose run: %v", err)
	}
}