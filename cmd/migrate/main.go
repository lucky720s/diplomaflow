package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var (
		command    = flag.String("cmd", "up", "Migration command: up, down, force, version")
		steps      = flag.Int("steps", 0, "Number of migrations to run (0 = all)")
		version    = flag.Int("version", 0, "Force version (use with -cmd=force)")
		dbURL      = flag.String("db", "", "Database URL (or use DATABASE_URL env)")
		migrations = flag.String("path", "file://db/migrations", "Path to migrations")
	)
	flag.Parse()

	if *dbURL == "" {
		*dbURL = os.Getenv("DATABASE_URL")
		if *dbURL == "" {
			// Fallback to DSN format
			dsn := os.Getenv("DATABASE_DSN")
			if dsn == "" {
				log.Fatal("Database URL required: use -db flag or DATABASE_URL env")
			}
			// Convert DSN to URL format
			*dbURL = fmt.Sprintf("postgres://%s", dsn)
		}
	}

	m, err := migrate.New(*migrations, *dbURL)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	switch *command {
	case "up":
		if *steps > 0 {
			err = m.Steps(*steps)
		} else {
			err = m.Up()
		}
		if err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration up failed: %v", err)
		}
		fmt.Println("✅ Migrations applied successfully")

	case "down":
		if *steps > 0 {
			err = m.Steps(-*steps)
		} else {
			err = m.Down()
		}
		if err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration down failed: %v", err)
		}
		fmt.Println("✅ Migrations rolled back successfully")

	case "force":
		if *version == 0 {
			log.Fatal("Version required for force command")
		}
		err = m.Force(*version)
		if err != nil {
			log.Fatalf("Force version failed: %v", err)
		}
		fmt.Printf("✅ Forced to version %d\n", *version)

	case "version":
		ver, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("Get version failed: %v", err)
		}
		fmt.Printf("Current version: %d (dirty: %v)\n", ver, dirty)

	case "create":
		name := flag.Arg(0)
		if name == "" {
			log.Fatal("Migration name required")
		}
		// Just print instructions
		fmt.Printf("Create migration files:\n")
		fmt.Printf("  db/migrations/XXXXXX_%s.up.sql\n", name)
		fmt.Printf("  db/migrations/XXXXXX_%s.down.sql\n", name)

	default:
		log.Fatalf("Unknown command: %s", *command)
	}
}
