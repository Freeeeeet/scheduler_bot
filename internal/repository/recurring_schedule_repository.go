package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// RecurringScheduleRepository управляет recurring расписаниями в базе данных
type RecurringScheduleRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewRecurringScheduleRepository создаёт новый репозиторий
func NewRecurringScheduleRepository(pool *pgxpool.Pool, logger *zap.Logger) *RecurringScheduleRepository {
	return &RecurringScheduleRepository{
		pool:   pool,
		logger: logger,
	}
}

// Create создаёт новый recurring schedule
func (r *RecurringScheduleRepository) Create(ctx context.Context, schedule *model.RecurringSchedule) error {
	query := `
		INSERT INTO recurring_schedules (group_id, teacher_id, subject_id, weekday, start_hour, start_minute, duration_minutes, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(
		ctx,
		query,
		schedule.GroupID,
		schedule.TeacherID,
		schedule.SubjectID,
		schedule.Weekday,
		schedule.StartHour,
		schedule.StartMinute,
		schedule.DurationMinutes,
		schedule.IsActive,
	).Scan(&schedule.ID, &schedule.CreatedAt, &schedule.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create recurring schedule: %w", err)
	}

	return nil
}

// GetByID получает recurring schedule по ID
func (r *RecurringScheduleRepository) GetByID(ctx context.Context, id int64) (*model.RecurringSchedule, error) {
	query := `
		SELECT id, group_id, teacher_id, subject_id, weekday, start_hour, start_minute, duration_minutes, is_active, created_at, updated_at
		FROM recurring_schedules
		WHERE id = $1
	`

	schedule := &model.RecurringSchedule{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&schedule.ID,
		&schedule.GroupID,
		&schedule.TeacherID,
		&schedule.SubjectID,
		&schedule.Weekday,
		&schedule.StartHour,
		&schedule.StartMinute,
		&schedule.DurationMinutes,
		&schedule.IsActive,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get recurring schedule by id: %w", err)
	}

	return schedule, nil
}

// GetByTeacherID получает все recurring schedules учителя
func (r *RecurringScheduleRepository) GetByTeacherID(ctx context.Context, teacherID int64) ([]*model.RecurringSchedule, error) {
	query := `
		SELECT id, group_id, teacher_id, subject_id, weekday, start_hour, start_minute, duration_minutes, is_active, created_at, updated_at
		FROM recurring_schedules
		WHERE teacher_id = $1
		ORDER BY weekday, start_hour, start_minute
	`

	rows, err := r.pool.Query(ctx, query, teacherID)
	if err != nil {
		return nil, fmt.Errorf("get recurring schedules by teacher: %w", err)
	}
	defer rows.Close()

	var schedules []*model.RecurringSchedule
	for rows.Next() {
		schedule := &model.RecurringSchedule{}
		err := rows.Scan(
			&schedule.ID,
			&schedule.GroupID,
			&schedule.TeacherID,
			&schedule.SubjectID,
			&schedule.Weekday,
			&schedule.StartHour,
			&schedule.StartMinute,
			&schedule.DurationMinutes,
			&schedule.IsActive,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan recurring schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// GetBySubjectID получает все recurring schedules для предмета
func (r *RecurringScheduleRepository) GetBySubjectID(ctx context.Context, subjectID int64) ([]*model.RecurringSchedule, error) {
	query := `
		SELECT id, group_id, teacher_id, subject_id, weekday, start_hour, start_minute, duration_minutes, is_active, created_at, updated_at
		FROM recurring_schedules
		WHERE subject_id = $1
		ORDER BY weekday, start_hour, start_minute
	`

	rows, err := r.pool.Query(ctx, query, subjectID)
	if err != nil {
		return nil, fmt.Errorf("get recurring schedules by subject: %w", err)
	}
	defer rows.Close()

	var schedules []*model.RecurringSchedule
	for rows.Next() {
		schedule := &model.RecurringSchedule{}
		err := rows.Scan(
			&schedule.ID,
			&schedule.GroupID,
			&schedule.TeacherID,
			&schedule.SubjectID,
			&schedule.Weekday,
			&schedule.StartHour,
			&schedule.StartMinute,
			&schedule.DurationMinutes,
			&schedule.IsActive,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan recurring schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// GetAllActive получает все активные recurring schedules
func (r *RecurringScheduleRepository) GetAllActive(ctx context.Context) ([]*model.RecurringSchedule, error) {
	query := `
		SELECT id, group_id, teacher_id, subject_id, weekday, start_hour, start_minute, duration_minutes, is_active, created_at, updated_at
		FROM recurring_schedules
		WHERE is_active = true
		ORDER BY weekday, start_hour, start_minute
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get all active recurring schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*model.RecurringSchedule
	for rows.Next() {
		schedule := &model.RecurringSchedule{}
		err := rows.Scan(
			&schedule.ID,
			&schedule.GroupID,
			&schedule.TeacherID,
			&schedule.SubjectID,
			&schedule.Weekday,
			&schedule.StartHour,
			&schedule.StartMinute,
			&schedule.DurationMinutes,
			&schedule.IsActive,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan recurring schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// Update обновляет recurring schedule
func (r *RecurringScheduleRepository) Update(ctx context.Context, schedule *model.RecurringSchedule) error {
	query := `
		UPDATE recurring_schedules
		SET weekday = $2, start_hour = $3, start_minute = $4, duration_minutes = $5, is_active = $6
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.pool.QueryRow(
		ctx,
		query,
		schedule.ID,
		schedule.Weekday,
		schedule.StartHour,
		schedule.StartMinute,
		schedule.DurationMinutes,
		schedule.IsActive,
	).Scan(&schedule.UpdatedAt)

	if err != nil {
		return fmt.Errorf("update recurring schedule: %w", err)
	}

	return nil
}

// Deactivate деактивирует recurring schedule
func (r *RecurringScheduleRepository) Deactivate(ctx context.Context, id int64) error {
	query := `UPDATE recurring_schedules SET is_active = false WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deactivate recurring schedule: %w", err)
	}

	return nil
}

// Delete удаляет recurring schedule
func (r *RecurringScheduleRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM recurring_schedules WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete recurring schedule: %w", err)
	}

	return nil
}

// GetSchedulesNeedingSlots возвращает recurring schedules, для которых нужно создать слоты
// на указанную дату
func (r *RecurringScheduleRepository) GetSchedulesNeedingSlots(ctx context.Context, date time.Time) ([]*model.RecurringSchedule, error) {
	weekday := int(date.Weekday())

	query := `
		SELECT id, group_id, teacher_id, subject_id, weekday, start_hour, start_minute, duration_minutes, is_active, created_at, updated_at
		FROM recurring_schedules
		WHERE is_active = true AND weekday = $1
	`

	rows, err := r.pool.Query(ctx, query, weekday)
	if err != nil {
		return nil, fmt.Errorf("get schedules needing slots: %w", err)
	}
	defer rows.Close()

	var schedules []*model.RecurringSchedule
	for rows.Next() {
		schedule := &model.RecurringSchedule{}
		err := rows.Scan(
			&schedule.ID,
			&schedule.GroupID,
			&schedule.TeacherID,
			&schedule.SubjectID,
			&schedule.Weekday,
			&schedule.StartHour,
			&schedule.StartMinute,
			&schedule.DurationMinutes,
			&schedule.IsActive,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan recurring schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// GetByGroupID получает все recurring schedules по group_id
func (r *RecurringScheduleRepository) GetByGroupID(ctx context.Context, groupID string) ([]*model.RecurringSchedule, error) {
	query := `
		SELECT id, group_id, teacher_id, subject_id, weekday, start_hour, start_minute, duration_minutes, is_active, created_at, updated_at
		FROM recurring_schedules
		WHERE group_id = $1
		ORDER BY weekday, start_hour, start_minute
	`

	rows, err := r.pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("get recurring schedules by group_id: %w", err)
	}
	defer rows.Close()

	var schedules []*model.RecurringSchedule
	for rows.Next() {
		schedule := &model.RecurringSchedule{}
		err := rows.Scan(
			&schedule.ID,
			&schedule.GroupID,
			&schedule.TeacherID,
			&schedule.SubjectID,
			&schedule.Weekday,
			&schedule.StartHour,
			&schedule.StartMinute,
			&schedule.DurationMinutes,
			&schedule.IsActive,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan recurring schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// DeactivateByGroupID деактивирует все recurring schedules в группе
func (r *RecurringScheduleRepository) DeactivateByGroupID(ctx context.Context, groupID string) error {
	query := `UPDATE recurring_schedules SET is_active = false WHERE group_id = $1`

	_, err := r.pool.Exec(ctx, query, groupID)
	if err != nil {
		return fmt.Errorf("deactivate recurring schedules by group_id: %w", err)
	}

	return nil
}

// DeleteByGroupID удаляет все recurring schedules в группе
func (r *RecurringScheduleRepository) DeleteByGroupID(ctx context.Context, groupID string) error {
	query := `DELETE FROM recurring_schedules WHERE group_id = $1`

	_, err := r.pool.Exec(ctx, query, groupID)
	if err != nil {
		return fmt.Errorf("delete recurring schedules by group_id: %w", err)
	}

	return nil
}
