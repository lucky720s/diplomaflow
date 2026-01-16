package main

import (
	"flag"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var (
		command = flag.String("cmd", "up", "Migration command: up, down, force, version")
		steps   = flag.Int("steps", 0, "Number of migrations to run (0 = all)")
		version = flag.Int("version", 0, "Force version (use with -cmd=force)")
		dbURL   = flag.String("db", "", "Database URL (or use DATABASE_URL env)")
		path    = flag.String("path", "file:///app/db/migrations", "Path to migrations (file:///abs/path)")
	)
	flag.Parse()

	if *dbURL == "" {
		*dbURL = os.Getenv("DATABASE_URL")
	}
	if *dbURL == "" {
		log.Fatal("DATABASE_URL is required (DSN format is not supported here)")
	}

	log.Printf("migrate: cmd=%s steps=%d version=%d", *command, *steps, *version)
	log.Printf("migrate: source=%s", *path)

	m, err := migrate.New(*path, *dbURL)
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
		log.Println("✅ Migrations applied successfully")

	case "down":
		if *steps > 0 {
			err = m.Steps(-*steps)
		} else {
			err = m.Down()
		}
		if err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration down failed: %v", err)
		}
		log.Println("✅ Migrations rolled back successfully")

	case "force":
		if *version == 0 {
			log.Fatal("Version required for force command")
		}
		err = m.Force(*version)
		if err != nil {
			log.Fatalf("Force version failed: %v", err)
		}
		log.Printf("✅ Forced to version %d\n", *version)

	case "version":
		ver, dirty, err2 := m.Version()
		if err2 != nil {
			log.Fatalf("Get version failed: %v", err2)
		}
		log.Printf("Current version: %d (dirty: %v)\n", ver, dirty)

	default:
		log.Fatalf("Unknown command: %s", *command)
	}
}
