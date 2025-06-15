package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"reconciliation-service/internal/config"
	"reconciliation-service/internal/database"
	"reconciliation-service/internal/handlers"
)

func main() {
	migrateCmd := flag.String("migrate", "", "Migration command (up/down/version)")
	steps := flag.Int("steps", 0, "Number of migration steps (0 means all)")
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	db, err := database.NewConnection(cfg)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	if *migrateCmd != "" {
		handleMigration(cfg, *migrateCmd, *steps)
		return
	}

	router := handlers.SetupRouter(db, cfg)

	srv := &http.Server{
		Addr:         cfg.ServerAddress,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("Server is running on %s", cfg.ServerAddress)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	log.Println("Server exited gracefully")
}

func handleMigration(cfg *config.Config, command string, steps int) {
	db, err := database.NewConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to ensure database exists: %v", err)
	}
	db.Close()

	m, err := migrate.New(
		fmt.Sprintf("file://%s", cfg.Migration.Dir),
		cfg.GetMigrationDBURL(),
	)
	if err != nil {
		if strings.Contains(err.Error(), "no change") {
			log.Printf("No migration changes to apply")
			return
		}
		log.Fatalf("Failed to initialize migrate: %v", err)
	}
	defer m.Close()

	switch command {
	case "up":
		if steps > 0 {
			err = m.Steps(steps)
		} else {
			err = m.Up()
		}
	case "down":
		if steps > 0 {
			err = m.Steps(-steps)
		} else {
			err = m.Down()
		}
	case "version":
		version, dirty, verErr := m.Version()
		if verErr != nil {
			if verErr == migrate.ErrNilVersion {
				log.Printf("No migrations have been applied yet")
				return
			}
			log.Fatalf("Failed to get version: %v", verErr)
		}
		fmt.Printf("Current migration version: %d (dirty: %v)\n", version, dirty)
		return
	default:
		log.Fatalf("Invalid migration command: %s", command)
	}

	if err != nil {
		if err == migrate.ErrNoChange {
			log.Printf("No migration changes to apply")
			return
		}
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration completed successfully")
}
