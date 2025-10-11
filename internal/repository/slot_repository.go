package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SlotRepository struct {
	pool *pgxpool.Pool
}

func NewSlotRepository(pool *pgxpool.Pool) *SlotRepository {
	return &SlotRepository{pool: pool}
}

// Create создаёт новый слот
func (r *SlotRepository) Create(ctx context.Context, slot *model.ScheduleSlot) error {
	query := `
		INSERT INTO schedule_slots (teacher_id, subject_id, start_time, end_time, status, student_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(
		ctx, query,
		slot.TeacherID,
		slot.SubjectID,
		slot.StartTime,
		slot.EndTime,
		slot.Status,
		slot.StudentID,
	).Scan(&slot.ID, &slot.CreatedAt)

	if err != nil {
		return fmt.Errorf("create slot: %w", err)
	}

	return nil
}

// GetByID получает слот по ID
func (r *SlotRepository) GetByID(ctx context.Context, id int64) (*model.ScheduleSlot, error) {
	query := `
		SELECT id, teacher_id, subject_id, start_time, end_time, status, student_id, created_at
		FROM schedule_slots
		WHERE id = $1
	`

	var slot model.ScheduleSlot
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&slot.ID,
		&slot.TeacherID,
		&slot.SubjectID,
		&slot.StartTime,
		&slot.EndTime,
		&slot.Status,
		&slot.StudentID,
		&slot.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get slot by id: %w", err)
	}

	return &slot, nil
}

// GetFreeSlots получает свободные слоты для предмета в заданном диапазоне времени
func (r *SlotRepository) GetFreeSlots(ctx context.Context, subjectID int64, from, to time.Time) ([]*model.ScheduleSlot, error) {
	query := `
		SELECT id, teacher_id, subject_id, start_time, end_time, status, student_id, created_at
		FROM schedule_slots
		WHERE subject_id = $1
		  AND status = 'free'
		  AND start_time >= $2
		  AND start_time < $3
		ORDER BY start_time
	`

	rows, err := r.pool.Query(ctx, query, subjectID, from, to)
	if err != nil {
		return nil, fmt.Errorf("get free slots: %w", err)
	}
	defer rows.Close()

	var slots []*model.ScheduleSlot
	for rows.Next() {
		var slot model.ScheduleSlot
		err := rows.Scan(
			&slot.ID,
			&slot.TeacherID,
			&slot.SubjectID,
			&slot.StartTime,
			&slot.EndTime,
			&slot.Status,
			&slot.StudentID,
			&slot.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan slot: %w", err)
		}
		slots = append(slots, &slot)
	}

	return slots, nil
}

// GetByTeacherID получает все слоты учителя
func (r *SlotRepository) GetByTeacherID(ctx context.Context, teacherID int64, from, to time.Time) ([]*model.ScheduleSlot, error) {
	query := `
		SELECT id, teacher_id, subject_id, start_time, end_time, status, student_id, created_at
		FROM schedule_slots
		WHERE teacher_id = $1
		  AND start_time >= $2
		  AND start_time < $3
		ORDER BY start_time
	`

	rows, err := r.pool.Query(ctx, query, teacherID, from, to)
	if err != nil {
		return nil, fmt.Errorf("get slots by teacher: %w", err)
	}
	defer rows.Close()

	var slots []*model.ScheduleSlot
	for rows.Next() {
		var slot model.ScheduleSlot
		err := rows.Scan(
			&slot.ID,
			&slot.TeacherID,
			&slot.SubjectID,
			&slot.StartTime,
			&slot.EndTime,
			&slot.Status,
			&slot.StudentID,
			&slot.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan slot: %w", err)
		}
		slots = append(slots, &slot)
	}

	return slots, nil
}

// Book бронирует слот для студента
func (r *SlotRepository) Book(ctx context.Context, slotID, studentID int64) error {
	query := `
		UPDATE schedule_slots
		SET status = 'booked', student_id = $1
		WHERE id = $2 AND status = 'free'
	`

	result, err := r.pool.Exec(ctx, query, studentID, slotID)
	if err != nil {
		return fmt.Errorf("book slot: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("slot not available or already booked")
	}

	return nil
}

// Cancel отменяет бронирование слота
func (r *SlotRepository) Cancel(ctx context.Context, slotID int64) error {
	query := `
		UPDATE schedule_slots
		SET status = 'canceled', student_id = NULL
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, slotID)
	if err != nil {
		return fmt.Errorf("cancel slot: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("slot not found")
	}

	return nil
}

// UpdateStatus обновляет статус слота
func (r *SlotRepository) UpdateStatus(ctx context.Context, slotID int64, status model.SlotStatus) error {
	query := `
		UPDATE schedule_slots
		SET status = $1
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, status, slotID)
	if err != nil {
		return fmt.Errorf("update slot status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("slot not found")
	}

	return nil
}

// SlotExists проверяет существование слота для учителя в указанное время
func (r *SlotRepository) SlotExists(ctx context.Context, teacherID int64, startTime time.Time) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM schedule_slots
			WHERE teacher_id = $1 AND start_time = $2
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, teacherID, startTime).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check slot exists: %w", err)
	}

	return exists, nil
}
