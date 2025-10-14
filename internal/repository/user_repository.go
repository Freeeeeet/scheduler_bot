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
		INSERT INTO users (telegram_id, username, first_name, last_name, language_code, is_teacher, is_public)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
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
		user.IsPublic,
	).Scan(&user.ID, &user.CreatedAt)

	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

// GetByTelegramID получает пользователя по Telegram ID
func (r *UserRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*model.User, error) {
	query := `
		SELECT id, telegram_id, username, first_name, last_name, language_code, is_teacher, is_public, auto_approve_bookings, created_at
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
		&user.IsPublic,
		&user.AutoApproveBookings,
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
		SELECT id, telegram_id, username, first_name, last_name, language_code, is_teacher, is_public, auto_approve_bookings, created_at
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
		&user.IsPublic,
		&user.AutoApproveBookings,
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
		SET username = $1, first_name = $2, last_name = $3, language_code = $4, is_teacher = $5, is_public = $6, auto_approve_bookings = $7
		WHERE id = $8
	`

	result, err := r.pool.Exec(
		ctx, query,
		user.Username,
		user.FirstName,
		user.LastName,
		user.LanguageCode,
		user.IsTeacher,
		user.IsPublic,
		user.AutoApproveBookings,
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

// UpdatePublicStatus обновляет публичность учителя
func (r *UserRepository) UpdatePublicStatus(ctx context.Context, userID int64, isPublic bool) error {
	query := `
		UPDATE users
		SET is_public = $1
		WHERE id = $2 AND is_teacher = true
	`

	result, err := r.pool.Exec(ctx, query, isPublic, userID)
	if err != nil {
		return fmt.Errorf("update public status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("teacher not found")
	}

	return nil
}

// GetPublicTeachers получает список публичных учителей
func (r *UserRepository) GetPublicTeachers(ctx context.Context) ([]*model.User, error) {
	query := `
		SELECT id, telegram_id, username, first_name, last_name, language_code, is_teacher, is_public, auto_approve_bookings, created_at
		FROM users
		WHERE is_teacher = true AND is_public = true
		ORDER BY first_name, last_name
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get public teachers: %w", err)
	}
	defer rows.Close()

	var teachers []*model.User
	for rows.Next() {
		var teacher model.User
		err := rows.Scan(
			&teacher.ID,
			&teacher.TelegramID,
			&teacher.Username,
			&teacher.FirstName,
			&teacher.LastName,
			&teacher.LanguageCode,
			&teacher.IsTeacher,
			&teacher.IsPublic,
			&teacher.AutoApproveBookings,
			&teacher.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan public teacher: %w", err)
		}
		teachers = append(teachers, &teacher)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate public teachers: %w", err)
	}

	return teachers, nil
}

// GetByIDs получает пользователей по списку ID
func (r *UserRepository) GetByIDs(ctx context.Context, ids []int64) ([]*model.User, error) {
	if len(ids) == 0 {
		return []*model.User{}, nil
	}

	query := `
		SELECT id, telegram_id, username, first_name, last_name, language_code, is_teacher, is_public, auto_approve_bookings, created_at
		FROM users
		WHERE id = ANY($1)
		ORDER BY first_name, last_name
	`

	rows, err := r.pool.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("get users by ids: %w", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var user model.User
		err := rows.Scan(
			&user.ID,
			&user.TelegramID,
			&user.Username,
			&user.FirstName,
			&user.LastName,
			&user.LanguageCode,
			&user.IsTeacher,
			&user.IsPublic,
			&user.AutoApproveBookings,
			&user.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return users, nil
}
