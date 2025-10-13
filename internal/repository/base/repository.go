package base

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository базовый репозиторий с общими методами
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository создаёт новый базовый репозиторий
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Pool возвращает пул соединений
func (r *Repository) Pool() *pgxpool.Pool {
	return r.pool
}

// QueryRow выполняет запрос и возвращает одну строку
func (r *Repository) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return r.pool.QueryRow(ctx, query, args...)
}

// Query выполняет запрос и возвращает множество строк
func (r *Repository) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return r.pool.Query(ctx, query, args...)
}

// ExecAffected выполняет команду и возвращает количество затронутых строк
func (r *Repository) ExecAffected(ctx context.Context, query string, args ...interface{}) (int64, error) {
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// IsNotFound проверяет является ли ошибка "строка не найдена"
func IsNotFound(err error) bool {
	return err == pgx.ErrNoRows
}

