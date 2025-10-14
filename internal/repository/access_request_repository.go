package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccessRequestRepository struct {
	pool *pgxpool.Pool
}

func NewAccessRequestRepository(pool *pgxpool.Pool) *AccessRequestRepository {
	return &AccessRequestRepository{pool: pool}
}

// Create создает заявку
func (r *AccessRequestRepository) Create(ctx context.Context, req *model.AccessRequest) error {
	query := `
		INSERT INTO access_requests (student_id, teacher_id, status, message)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(
		ctx, query,
		req.StudentID,
		req.TeacherID,
		req.Status,
		req.Message,
	).Scan(&req.ID, &req.CreatedAt)

	if err != nil {
		return fmt.Errorf("create access request: %w", err)
	}

	return nil
}

// GetByID получает заявку по ID
func (r *AccessRequestRepository) GetByID(ctx context.Context, id int64) (*model.AccessRequest, error) {
	query := `
		SELECT id, student_id, teacher_id, status, message, teacher_response, created_at, updated_at
		FROM access_requests
		WHERE id = $1
	`

	var req model.AccessRequest
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&req.ID,
		&req.StudentID,
		&req.TeacherID,
		&req.Status,
		&req.Message,
		&req.TeacherResponse,
		&req.CreatedAt,
		&req.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get access request: %w", err)
	}

	return &req, nil
}

// GetPendingByTeacher получает pending заявки учителя
func (r *AccessRequestRepository) GetPendingByTeacher(ctx context.Context, teacherID int64) ([]*model.AccessRequest, error) {
	query := `
		SELECT id, student_id, teacher_id, status, message, teacher_response, created_at, updated_at
		FROM access_requests
		WHERE teacher_id = $1 AND status = $2
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, teacherID, model.RequestStatusPending)
	if err != nil {
		return nil, fmt.Errorf("get pending requests: %w", err)
	}
	defer rows.Close()

	var requests []*model.AccessRequest
	for rows.Next() {
		var req model.AccessRequest
		err := rows.Scan(
			&req.ID,
			&req.StudentID,
			&req.TeacherID,
			&req.Status,
			&req.Message,
			&req.TeacherResponse,
			&req.CreatedAt,
			&req.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan access request: %w", err)
		}
		requests = append(requests, &req)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate requests: %w", err)
	}

	return requests, nil
}

// GetByStudent получает заявки студента
func (r *AccessRequestRepository) GetByStudent(ctx context.Context, studentID int64) ([]*model.AccessRequest, error) {
	query := `
		SELECT id, student_id, teacher_id, status, message, teacher_response, created_at, updated_at
		FROM access_requests
		WHERE student_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, studentID)
	if err != nil {
		return nil, fmt.Errorf("get student requests: %w", err)
	}
	defer rows.Close()

	var requests []*model.AccessRequest
	for rows.Next() {
		var req model.AccessRequest
		err := rows.Scan(
			&req.ID,
			&req.StudentID,
			&req.TeacherID,
			&req.Status,
			&req.Message,
			&req.TeacherResponse,
			&req.CreatedAt,
			&req.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan access request: %w", err)
		}
		requests = append(requests, &req)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate requests: %w", err)
	}

	return requests, nil
}

// GetByStudentAndStatus получает заявки студента по статусу
func (r *AccessRequestRepository) GetByStudentAndStatus(ctx context.Context, studentID int64, status string) ([]*model.AccessRequest, error) {
	query := `
		SELECT id, student_id, teacher_id, status, message, teacher_response, created_at, updated_at
		FROM access_requests
		WHERE student_id = $1 AND status = $2
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, studentID, status)
	if err != nil {
		return nil, fmt.Errorf("get student requests by status: %w", err)
	}
	defer rows.Close()

	var requests []*model.AccessRequest
	for rows.Next() {
		var req model.AccessRequest
		err := rows.Scan(
			&req.ID,
			&req.StudentID,
			&req.TeacherID,
			&req.Status,
			&req.Message,
			&req.TeacherResponse,
			&req.CreatedAt,
			&req.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan access request: %w", err)
		}
		requests = append(requests, &req)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate requests: %w", err)
	}

	return requests, nil
}

// HasPendingRequest проверяет, есть ли активная заявка
func (r *AccessRequestRepository) HasPendingRequest(ctx context.Context, studentID, teacherID int64) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM access_requests
			WHERE student_id = $1 AND teacher_id = $2 AND status = $3
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, studentID, teacherID, model.RequestStatusPending).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check pending request: %w", err)
	}

	return exists, nil
}

// UpdateStatus обновляет статус заявки
func (r *AccessRequestRepository) UpdateStatus(ctx context.Context, id int64, status, response string) error {
	query := `
		UPDATE access_requests
		SET status = $1, teacher_response = $2, updated_at = $3
		WHERE id = $4
	`

	now := time.Now()
	result, err := r.pool.Exec(ctx, query, status, response, now, id)
	if err != nil {
		return fmt.Errorf("update request status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("request not found")
	}

	return nil
}

// CountPendingByTeacher подсчитывает количество pending заявок учителя
func (r *AccessRequestRepository) CountPendingByTeacher(ctx context.Context, teacherID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM access_requests
		WHERE teacher_id = $1 AND status = $2
	`

	var count int
	err := r.pool.QueryRow(ctx, query, teacherID, model.RequestStatusPending).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count pending requests: %w", err)
	}

	return count, nil
}

// Delete удаляет заявку (может быть полезно для очистки старых заявок)
func (r *AccessRequestRepository) Delete(ctx context.Context, id int64) error {
	query := `
		DELETE FROM access_requests
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete request: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("request not found")
	}

	return nil
}

// GetPendingRequest получает конкретную pending заявку студента к учителю
func (r *AccessRequestRepository) GetPendingRequest(ctx context.Context, studentID, teacherID int64) (*model.AccessRequest, error) {
	query := `
		SELECT id, student_id, teacher_id, status, message, teacher_response, created_at, updated_at
		FROM access_requests
		WHERE student_id = $1 AND teacher_id = $2 AND status = $3
	`

	var req model.AccessRequest
	err := r.pool.QueryRow(ctx, query, studentID, teacherID, model.RequestStatusPending).Scan(
		&req.ID,
		&req.StudentID,
		&req.TeacherID,
		&req.Status,
		&req.Message,
		&req.TeacherResponse,
		&req.CreatedAt,
		&req.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get pending request: %w", err)
	}

	return &req, nil
}
