package repository

import (
	"context"
	"fmt"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BookingRepository struct {
	pool *pgxpool.Pool
}

func NewBookingRepository(pool *pgxpool.Pool) *BookingRepository {
	return &BookingRepository{pool: pool}
}

// Create создаёт новое бронирование
func (r *BookingRepository) Create(ctx context.Context, booking *model.Booking) error {
	query := `
		INSERT INTO bookings (student_id, teacher_id, subject_id, slot_id, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(
		ctx, query,
		booking.StudentID,
		booking.TeacherID,
		booking.SubjectID,
		booking.SlotID,
		booking.Status,
	).Scan(&booking.ID, &booking.CreatedAt, &booking.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create booking: %w", err)
	}

	return nil
}

// GetByID получает бронирование по ID
func (r *BookingRepository) GetByID(ctx context.Context, id int64) (*model.Booking, error) {
	query := `
		SELECT id, student_id, teacher_id, subject_id, slot_id, status, created_at, updated_at
		FROM bookings
		WHERE id = $1
	`

	var booking model.Booking
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&booking.ID,
		&booking.StudentID,
		&booking.TeacherID,
		&booking.SubjectID,
		&booking.SlotID,
		&booking.Status,
		&booking.CreatedAt,
		&booking.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get booking by id: %w", err)
	}

	return &booking, nil
}

// GetByStudentID получает все бронирования студента
func (r *BookingRepository) GetByStudentID(ctx context.Context, studentID int64) ([]*model.Booking, error) {
	query := `
		SELECT id, student_id, teacher_id, subject_id, slot_id, status, created_at, updated_at
		FROM bookings
		WHERE student_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, studentID)
	if err != nil {
		return nil, fmt.Errorf("get bookings by student: %w", err)
	}
	defer rows.Close()

	var bookings []*model.Booking
	for rows.Next() {
		var booking model.Booking
		err := rows.Scan(
			&booking.ID,
			&booking.StudentID,
			&booking.TeacherID,
			&booking.SubjectID,
			&booking.SlotID,
			&booking.Status,
			&booking.CreatedAt,
			&booking.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		bookings = append(bookings, &booking)
	}

	return bookings, nil
}

// GetByTeacherID получает все бронирования для учителя
func (r *BookingRepository) GetByTeacherID(ctx context.Context, teacherID int64) ([]*model.Booking, error) {
	query := `
		SELECT id, student_id, teacher_id, subject_id, slot_id, status, created_at, updated_at
		FROM bookings
		WHERE teacher_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, teacherID)
	if err != nil {
		return nil, fmt.Errorf("get bookings by teacher: %w", err)
	}
	defer rows.Close()

	var bookings []*model.Booking
	for rows.Next() {
		var booking model.Booking
		err := rows.Scan(
			&booking.ID,
			&booking.StudentID,
			&booking.TeacherID,
			&booking.SubjectID,
			&booking.SlotID,
			&booking.Status,
			&booking.CreatedAt,
			&booking.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		bookings = append(bookings, &booking)
	}

	return bookings, nil
}

// UpdateStatus обновляет статус бронирования
func (r *BookingRepository) UpdateStatus(ctx context.Context, id int64, status model.BookingStatus) error {
	query := `
		UPDATE bookings
		SET status = $1
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("update booking status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("booking not found")
	}

	return nil
}

// GetBySlotID получает активное бронирование для слота
func (r *BookingRepository) GetBySlotID(ctx context.Context, slotID int64) (*model.Booking, error) {
	query := `
		SELECT id, student_id, teacher_id, subject_id, slot_id, status, created_at, updated_at
		FROM bookings
		WHERE slot_id = $1 AND (status = 'confirmed' OR status = 'pending')
		LIMIT 1
	`

	var booking model.Booking
	err := r.pool.QueryRow(ctx, query, slotID).Scan(
		&booking.ID,
		&booking.StudentID,
		&booking.TeacherID,
		&booking.SubjectID,
		&booking.SlotID,
		&booking.Status,
		&booking.CreatedAt,
		&booking.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get booking by slot: %w", err)
	}

	return &booking, nil
}

// GetPendingByTeacherID получает все pending бронирования учителя
func (r *BookingRepository) GetPendingByTeacherID(ctx context.Context, teacherID int64) ([]*model.Booking, error) {
	query := `
		SELECT id, student_id, teacher_id, subject_id, slot_id, status, created_at, updated_at
		FROM bookings
		WHERE teacher_id = $1 AND status = 'pending'
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, teacherID)
	if err != nil {
		return nil, fmt.Errorf("get pending bookings by teacher: %w", err)
	}
	defer rows.Close()

	var bookings []*model.Booking
	for rows.Next() {
		var booking model.Booking
		err := rows.Scan(
			&booking.ID,
			&booking.StudentID,
			&booking.TeacherID,
			&booking.SubjectID,
			&booking.SlotID,
			&booking.Status,
			&booking.CreatedAt,
			&booking.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		bookings = append(bookings, &booking)
	}

	return bookings, nil
}

// GetBySubjectID получает все активные бронирования для предмета
func (r *BookingRepository) GetBySubjectID(ctx context.Context, subjectID int64) ([]*model.Booking, error) {
	query := `
		SELECT id, student_id, teacher_id, subject_id, slot_id, status, created_at, updated_at
		FROM bookings
		WHERE subject_id = $1 AND (status = 'confirmed' OR status = 'pending')
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, subjectID)
	if err != nil {
		return nil, fmt.Errorf("get bookings by subject: %w", err)
	}
	defer rows.Close()

	var bookings []*model.Booking
	for rows.Next() {
		var booking model.Booking
		err := rows.Scan(
			&booking.ID,
			&booking.StudentID,
			&booking.TeacherID,
			&booking.SubjectID,
			&booking.SlotID,
			&booking.Status,
			&booking.CreatedAt,
			&booking.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		bookings = append(bookings, &booking)
	}

	return bookings, nil
}

// Delete удаляет бронирование
func (r *BookingRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM bookings WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete booking: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("booking not found")
	}

	return nil
}
