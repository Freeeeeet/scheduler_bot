package repository

import (
	"context"
	"fmt"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create создаёт нового пользователя
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (telegram_id, username, first_name, last_name, language_code, is_teacher)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(
		ctx, query,
		user.TelegramID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.LanguageCode,
		user.IsTeacher,
	).Scan(&user.ID, &user.CreatedAt)

	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

// GetByTelegramID получает пользователя по Telegram ID
func (r *UserRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*model.User, error) {
	query := `
		SELECT id, telegram_id, username, first_name, last_name, language_code, is_teacher, created_at
		FROM users
		WHERE telegram_id = $1
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, telegramID).Scan(
		&user.ID,
		&user.TelegramID,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.LanguageCode,
		&user.IsTeacher,
		&user.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Пользователь не найден
		}
		return nil, fmt.Errorf("get user by telegram id: %w", err)
	}

	return &user, nil
}

// GetByID получает пользователя по ID
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	query := `
		SELECT id, telegram_id, username, first_name, last_name, language_code, is_teacher, created_at
		FROM users
		WHERE id = $1
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.TelegramID,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.LanguageCode,
		&user.IsTeacher,
		&user.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return &user, nil
}

// Update обновляет данные пользователя
func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users
		SET username = $1, first_name = $2, last_name = $3, language_code = $4, is_teacher = $5
		WHERE id = $6
	`

	result, err := r.pool.Exec(
		ctx, query,
		user.Username,
		user.FirstName,
		user.LastName,
		user.LanguageCode,
		user.IsTeacher,
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
