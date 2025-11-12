package init_database

import (
	"2025_2_a4code/internal/config"
	"database/sql"
	"fmt"
	"log/slog"

	e "2025_2_a4code/internal/lib/wrapper"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v4/stdlib"
)

type Storage struct {
	db *sql.DB
}

func NewDbConnection(dbConfig *config.DBConfig) (*sql.DB, error) {
	const op = "app.newDbConnection"

	connectionString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.Name, dbConfig.SSLMode,
	)

	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		return nil, e.Wrap(op, err)
	}

	if err = db.Ping(); err != nil {
		return nil, e.Wrap(op, err)
	}

	slog.Info("Connected to postgresql successfully")

	return db, nil
}

func RunMigrations(db *sql.DB, migrationsDir string) error {
	const op = "app.runMigrations"

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return e.Wrap(op, err)
	}

	m, err := migrate.NewWithDatabaseInstance(migrationsDir, "postgres", driver)
	if err != nil {
		return e.Wrap(op, err)
	}

	slog.Info("Applying migrations...")
	err = m.Up()
	if err != nil {
		return e.Wrap(op, err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		slog.Info("Migrations are already successfully applied")
	} else {
		slog.Info("Migrations are successfully applied")
	}

	return nil
}
