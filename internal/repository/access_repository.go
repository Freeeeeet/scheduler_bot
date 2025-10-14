package repository

import (
	"context"
	"fmt"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccessRepository struct {
	pool *pgxpool.Pool
}

func NewAccessRepository(pool *pgxpool.Pool) *AccessRepository {
	return &AccessRepository{pool: pool}
}

// HasAccess проверяет, есть ли у студента доступ к учителю
func (r *AccessRepository) HasAccess(ctx context.Context, studentID, teacherID int64) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM student_teacher_access
			WHERE student_id = $1 AND teacher_id = $2
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, studentID, teacherID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check access: %w", err)
	}

	return exists, nil
}

// GrantAccess предоставляет доступ студенту к учителю
func (r *AccessRepository) GrantAccess(ctx context.Context, studentID, teacherID int64, accessType string) error {
	query := `
		INSERT INTO student_teacher_access (student_id, teacher_id, access_type)
		VALUES ($1, $2, $3)
		ON CONFLICT (student_id, teacher_id) DO NOTHING
	`

	_, err := r.pool.Exec(ctx, query, studentID, teacherID, accessType)
	if err != nil {
		return fmt.Errorf("grant access: %w", err)
	}

	return nil
}

// RevokeAccess отзывает доступ студента к учителю
func (r *AccessRepository) RevokeAccess(ctx context.Context, studentID, teacherID int64) error {
	query := `
		DELETE FROM student_teacher_access
		WHERE student_id = $1 AND teacher_id = $2
	`

	result, err := r.pool.Exec(ctx, query, studentID, teacherID)
	if err != nil {
		return fmt.Errorf("revoke access: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("access record not found")
	}

	return nil
}

// GetStudentTeacherIDs получает ID всех учителей студента
func (r *AccessRepository) GetStudentTeacherIDs(ctx context.Context, studentID int64) ([]int64, error) {
	query := `
		SELECT teacher_id
		FROM student_teacher_access
		WHERE student_id = $1
		ORDER BY granted_at DESC
	`

	rows, err := r.pool.Query(ctx, query, studentID)
	if err != nil {
		return nil, fmt.Errorf("get student teachers: %w", err)
	}
	defer rows.Close()

	var teacherIDs []int64
	for rows.Next() {
		var teacherID int64
		if err := rows.Scan(&teacherID); err != nil {
			return nil, fmt.Errorf("scan teacher id: %w", err)
		}
		teacherIDs = append(teacherIDs, teacherID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate teacher ids: %w", err)
	}

	return teacherIDs, nil
}

// GetTeacherStudentIDs получает ID всех студентов учителя
func (r *AccessRepository) GetTeacherStudentIDs(ctx context.Context, teacherID int64) ([]int64, error) {
	query := `
		SELECT student_id
		FROM student_teacher_access
		WHERE teacher_id = $1
		ORDER BY granted_at DESC
	`

	rows, err := r.pool.Query(ctx, query, teacherID)
	if err != nil {
		return nil, fmt.Errorf("get teacher students: %w", err)
	}
	defer rows.Close()

	var studentIDs []int64
	for rows.Next() {
		var studentID int64
		if err := rows.Scan(&studentID); err != nil {
			return nil, fmt.Errorf("scan student id: %w", err)
		}
		studentIDs = append(studentIDs, studentID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate student ids: %w", err)
	}

	return studentIDs, nil
}

// GetAccessInfo получает информацию о доступе
func (r *AccessRepository) GetAccessInfo(ctx context.Context, studentID, teacherID int64) (*model.StudentTeacherAccess, error) {
	query := `
		SELECT id, student_id, teacher_id, access_type, granted_at
		FROM student_teacher_access
		WHERE student_id = $1 AND teacher_id = $2
	`

	var access model.StudentTeacherAccess
	err := r.pool.QueryRow(ctx, query, studentID, teacherID).Scan(
		&access.ID,
		&access.StudentID,
		&access.TeacherID,
		&access.AccessType,
		&access.GrantedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get access info: %w", err)
	}

	return &access, nil
}

// GetStudentAccessList получает полный список доступов студента
func (r *AccessRepository) GetStudentAccessList(ctx context.Context, studentID int64) ([]*model.StudentTeacherAccess, error) {
	query := `
		SELECT id, student_id, teacher_id, access_type, granted_at
		FROM student_teacher_access
		WHERE student_id = $1
		ORDER BY granted_at DESC
	`

	rows, err := r.pool.Query(ctx, query, studentID)
	if err != nil {
		return nil, fmt.Errorf("get student access list: %w", err)
	}
	defer rows.Close()

	var accessList []*model.StudentTeacherAccess
	for rows.Next() {
		var access model.StudentTeacherAccess
		err := rows.Scan(
			&access.ID,
			&access.StudentID,
			&access.TeacherID,
			&access.AccessType,
			&access.GrantedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan access: %w", err)
		}
		accessList = append(accessList, &access)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate access list: %w", err)
	}

	return accessList, nil
}

// CountTeacherStudents подсчитывает количество студентов у учителя
func (r *AccessRepository) CountTeacherStudents(ctx context.Context, teacherID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM student_teacher_access
		WHERE teacher_id = $1
	`

	var count int
	err := r.pool.QueryRow(ctx, query, teacherID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count teacher students: %w", err)
	}

	return count, nil
}
