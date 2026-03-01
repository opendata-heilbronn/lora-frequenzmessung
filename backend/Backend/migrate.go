package main

import (
	"embed"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func runMigrations(dsn string) {
	d, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		log.Fatalf("failed to load migrations: %v", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", d, dsn)
	if err != nil {
		log.Fatalf("failed to create migrator: %v", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migration failed: %v", err)
	}
	log.Println("Migrations applied successfully")
}
