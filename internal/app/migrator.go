package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Migrator обёртка над goose
type Migrator struct {
	pool           *pgxpool.Pool
	db             *sql.DB
	migrationsPath string
}

// NewMigrator создаёт новый мигратор
func NewMigrator(pool *pgxpool.Pool, migrationsPath string) (*Migrator, error) {
	// Устанавливаем диалект для PostgreSQL
	if err := goose.SetDialect("postgres"); err != nil {
		return nil, fmt.Errorf("set goose dialect: %w", err)
	}

	// Goose работает с *sql.DB, поэтому создаём его из конфига пула
	db := stdlib.OpenDBFromPool(pool)

	return &Migrator{
		pool:           pool,
		db:             db,
		migrationsPath: migrationsPath,
	}, nil
}

// Run применяет все pending миграции
func (mg *Migrator) Run(ctx context.Context) error {
	log.Println("🔄 Applying database migrations...")

	err := goose.UpContext(ctx, mg.db, mg.migrationsPath)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	log.Println("✅ Migrations applied successfully")
	return nil
}

// Version показывает текущую версию миграций
func (mg *Migrator) Version(ctx context.Context) (int64, error) {
	version, err := goose.GetDBVersionContext(ctx, mg.db)
	if err != nil {
		return 0, fmt.Errorf("get version: %w", err)
	}
	return version, nil
}

// Close закрывает соединение мигратора
func (mg *Migrator) Close() error {
	// Закрываем sql.DB, но не пул (он управляется в main)
	if mg.db != nil {
		return mg.db.Close()
	}
	return nil
}
