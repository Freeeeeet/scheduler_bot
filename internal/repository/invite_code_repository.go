package repository

import (
	"context"
	"fmt"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InviteCodeRepository struct {
	pool *pgxpool.Pool
}

func NewInviteCodeRepository(pool *pgxpool.Pool) *InviteCodeRepository {
	return &InviteCodeRepository{pool: pool}
}

// Create создает новый invite-код
func (r *InviteCodeRepository) Create(ctx context.Context, code *model.TeacherInviteCode) error {
	query := `
		INSERT INTO teacher_invite_codes (teacher_id, code, max_uses, expires_at, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, current_uses, created_at
	`

	err := r.pool.QueryRow(
		ctx, query,
		code.TeacherID,
		code.Code,
		code.MaxUses,
		code.ExpiresAt,
		code.IsActive,
	).Scan(&code.ID, &code.CurrentUses, &code.CreatedAt)

	if err != nil {
		return fmt.Errorf("create invite code: %w", err)
	}

	return nil
}

// GetByCode получает код по строке
func (r *InviteCodeRepository) GetByCode(ctx context.Context, code string) (*model.TeacherInviteCode, error) {
	query := `
		SELECT id, teacher_id, code, max_uses, current_uses, expires_at, is_active, created_at
		FROM teacher_invite_codes
		WHERE code = $1
	`

	var inviteCode model.TeacherInviteCode
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&inviteCode.ID,
		&inviteCode.TeacherID,
		&inviteCode.Code,
		&inviteCode.MaxUses,
		&inviteCode.CurrentUses,
		&inviteCode.ExpiresAt,
		&inviteCode.IsActive,
		&inviteCode.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get invite code by code: %w", err)
	}

	return &inviteCode, nil
}

// GetByID получает код по ID
func (r *InviteCodeRepository) GetByID(ctx context.Context, id int64) (*model.TeacherInviteCode, error) {
	query := `
		SELECT id, teacher_id, code, max_uses, current_uses, expires_at, is_active, created_at
		FROM teacher_invite_codes
		WHERE id = $1
	`

	var inviteCode model.TeacherInviteCode
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&inviteCode.ID,
		&inviteCode.TeacherID,
		&inviteCode.Code,
		&inviteCode.MaxUses,
		&inviteCode.CurrentUses,
		&inviteCode.ExpiresAt,
		&inviteCode.IsActive,
		&inviteCode.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get invite code by id: %w", err)
	}

	return &inviteCode, nil
}

// GetByTeacherID получает все коды учителя
func (r *InviteCodeRepository) GetByTeacherID(ctx context.Context, teacherID int64) ([]*model.TeacherInviteCode, error) {
	query := `
		SELECT id, teacher_id, code, max_uses, current_uses, expires_at, is_active, created_at
		FROM teacher_invite_codes
		WHERE teacher_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, teacherID)
	if err != nil {
		return nil, fmt.Errorf("get invite codes by teacher: %w", err)
	}
	defer rows.Close()

	var codes []*model.TeacherInviteCode
	for rows.Next() {
		var code model.TeacherInviteCode
		err := rows.Scan(
			&code.ID,
			&code.TeacherID,
			&code.Code,
			&code.MaxUses,
			&code.CurrentUses,
			&code.ExpiresAt,
			&code.IsActive,
			&code.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan invite code: %w", err)
		}
		codes = append(codes, &code)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invite codes: %w", err)
	}

	return codes, nil
}

// GetActiveByTeacherID получает активные коды учителя
func (r *InviteCodeRepository) GetActiveByTeacherID(ctx context.Context, teacherID int64) ([]*model.TeacherInviteCode, error) {
	query := `
		SELECT id, teacher_id, code, max_uses, current_uses, expires_at, is_active, created_at
		FROM teacher_invite_codes
		WHERE teacher_id = $1 AND is_active = true
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, teacherID)
	if err != nil {
		return nil, fmt.Errorf("get active invite codes: %w", err)
	}
	defer rows.Close()

	var codes []*model.TeacherInviteCode
	for rows.Next() {
		var code model.TeacherInviteCode
		err := rows.Scan(
			&code.ID,
			&code.TeacherID,
			&code.Code,
			&code.MaxUses,
			&code.CurrentUses,
			&code.ExpiresAt,
			&code.IsActive,
			&code.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan invite code: %w", err)
		}
		codes = append(codes, &code)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invite codes: %w", err)
	}

	return codes, nil
}

// UseCode использует код (инкремент current_uses)
func (r *InviteCodeRepository) UseCode(ctx context.Context, codeID int64) error {
	query := `
		UPDATE teacher_invite_codes
		SET current_uses = current_uses + 1
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, codeID)
	if err != nil {
		return fmt.Errorf("use invite code: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("invite code not found")
	}

	return nil
}

// Deactivate деактивирует код
func (r *InviteCodeRepository) Deactivate(ctx context.Context, codeID int64) error {
	query := `
		UPDATE teacher_invite_codes
		SET is_active = false
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, codeID)
	if err != nil {
		return fmt.Errorf("deactivate invite code: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("invite code not found")
	}

	return nil
}

// Delete удаляет код
func (r *InviteCodeRepository) Delete(ctx context.Context, codeID int64) error {
	query := `
		DELETE FROM teacher_invite_codes
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, codeID)
	if err != nil {
		return fmt.Errorf("delete invite code: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("invite code not found")
	}

	return nil
}

// IsValid проверяет валидность кода (активен, не истек, не израсходован)
func (r *InviteCodeRepository) IsValid(ctx context.Context, code string) (bool, error) {
	inviteCode, err := r.GetByCode(ctx, code)
	if err != nil {
		return false, err
	}

	if inviteCode == nil {
		return false, nil
	}

	return inviteCode.IsValid(), nil
}

// CodeExists проверяет, существует ли код с такой строкой
func (r *InviteCodeRepository) CodeExists(ctx context.Context, code string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM teacher_invite_codes
			WHERE code = $1
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, code).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check code exists: %w", err)
	}

	return exists, nil
}

// CountActiveCodesByTeacher подсчитывает активные коды учителя
func (r *InviteCodeRepository) CountActiveCodesByTeacher(ctx context.Context, teacherID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM teacher_invite_codes
		WHERE teacher_id = $1 AND is_active = true
	`

	var count int
	err := r.pool.QueryRow(ctx, query, teacherID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active codes: %w", err)
	}

	return count, nil
}
