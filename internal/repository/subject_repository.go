package repository

import (
	"context"
	"fmt"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type SubjectRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewSubjectRepository(pool *pgxpool.Pool, logger *zap.Logger) *SubjectRepository {
	return &SubjectRepository{
		pool:   pool,
		logger: logger,
	}
}

// Create создаёт новый предмет
func (r *SubjectRepository) Create(ctx context.Context, subject *model.Subject) error {
	r.logger.Info("SubjectRepository.Create called",
		zap.Int64("teacher_id", subject.TeacherID),
		zap.String("name", subject.Name),
		zap.String("description", subject.Description),
		zap.Int("price", subject.Price),
		zap.Int("duration", subject.Duration),
		zap.Bool("is_active", subject.IsActive),
		zap.Bool("requires_approval", subject.RequiresBookingApproval))

	query := `
		INSERT INTO subjects (teacher_id, name, description, price, duration, is_active, requires_booking_approval)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(
		ctx, query,
		subject.TeacherID,
		subject.Name,
		subject.Description,
		subject.Price,
		subject.Duration,
		subject.IsActive,
		subject.RequiresBookingApproval,
	).Scan(&subject.ID, &subject.CreatedAt)

	if err != nil {
		r.logger.Error("Failed to insert subject into DB",
			zap.Int64("teacher_id", subject.TeacherID),
			zap.String("name", subject.Name),
			zap.Error(err))
		return fmt.Errorf("create subject: %w", err)
	}

	r.logger.Info("Subject inserted successfully",
		zap.Int64("subject_id", subject.ID),
		zap.Int64("teacher_id", subject.TeacherID),
		zap.String("name", subject.Name))

	return nil
}

// GetByID получает предмет по ID
func (r *SubjectRepository) GetByID(ctx context.Context, id int64) (*model.Subject, error) {
	query := `
		SELECT id, teacher_id, name, description, price, duration, is_active, requires_booking_approval, created_at
		FROM subjects
		WHERE id = $1
	`

	var subject model.Subject
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&subject.ID,
		&subject.TeacherID,
		&subject.Name,
		&subject.Description,
		&subject.Price,
		&subject.Duration,
		&subject.IsActive,
		&subject.RequiresBookingApproval,
		&subject.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get subject by id: %w", err)
	}

	return &subject, nil
}

// GetByTeacherID получает все предметы учителя
func (r *SubjectRepository) GetByTeacherID(ctx context.Context, teacherID int64) ([]*model.Subject, error) {
	r.logger.Info("GetByTeacherID called",
		zap.Int64("teacher_id", teacherID))

	query := `
		SELECT id, teacher_id, name, description, price, duration, is_active, requires_booking_approval, created_at
		FROM subjects
		WHERE teacher_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, teacherID)
	if err != nil {
		r.logger.Error("Failed to query subjects",
			zap.Int64("teacher_id", teacherID),
			zap.Error(err))
		return nil, fmt.Errorf("get subjects by teacher: %w", err)
	}
	defer rows.Close()

	var subjects []*model.Subject
	for rows.Next() {
		var subject model.Subject
		err := rows.Scan(
			&subject.ID,
			&subject.TeacherID,
			&subject.Name,
			&subject.Description,
			&subject.Price,
			&subject.Duration,
			&subject.IsActive,
			&subject.RequiresBookingApproval,
			&subject.CreatedAt,
		)
		if err != nil {
			r.logger.Error("Failed to scan subject",
				zap.Error(err))
			return nil, fmt.Errorf("scan subject: %w", err)
		}
		subjects = append(subjects, &subject)
	}

	r.logger.Info("Retrieved subjects",
		zap.Int64("teacher_id", teacherID),
		zap.Int("count", len(subjects)))

	return subjects, nil
}

// GetActive получает все активные предметы
func (r *SubjectRepository) GetActive(ctx context.Context) ([]*model.Subject, error) {
	query := `
		SELECT id, teacher_id, name, description, price, duration, is_active, requires_booking_approval, created_at
		FROM subjects
		WHERE is_active = true
		ORDER BY name
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get active subjects: %w", err)
	}
	defer rows.Close()

	var subjects []*model.Subject
	for rows.Next() {
		var subject model.Subject
		err := rows.Scan(
			&subject.ID,
			&subject.TeacherID,
			&subject.Name,
			&subject.Description,
			&subject.Price,
			&subject.Duration,
			&subject.IsActive,
			&subject.RequiresBookingApproval,
			&subject.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan subject: %w", err)
		}
		subjects = append(subjects, &subject)
	}

	return subjects, nil
}

// Update обновляет предмет
func (r *SubjectRepository) Update(ctx context.Context, subject *model.Subject) error {
	query := `
		UPDATE subjects
		SET name = $1, description = $2, price = $3, duration = $4, is_active = $5, requires_booking_approval = $6
		WHERE id = $7
	`

	result, err := r.pool.Exec(
		ctx, query,
		subject.Name,
		subject.Description,
		subject.Price,
		subject.Duration,
		subject.IsActive,
		subject.RequiresBookingApproval,
		subject.ID,
	)

	if err != nil {
		return fmt.Errorf("update subject: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("subject not found")
	}

	return nil
}

// Delete удаляет предмет
func (r *SubjectRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM subjects WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete subject: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("subject not found")
	}

	return nil
}

// GetPublicActive получает активные предметы публичных учителей
func (r *SubjectRepository) GetPublicActive(ctx context.Context) ([]*model.Subject, error) {
	query := `
		SELECT s.id, s.teacher_id, s.name, s.description, s.price, s.duration, s.is_active, s.requires_booking_approval, s.created_at
		FROM subjects s
		INNER JOIN users u ON s.teacher_id = u.id
		WHERE s.is_active = true AND u.is_teacher = true AND u.is_public = true
		ORDER BY s.name
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get public active subjects: %w", err)
	}
	defer rows.Close()

	var subjects []*model.Subject
	for rows.Next() {
		var subject model.Subject
		err := rows.Scan(
			&subject.ID,
			&subject.TeacherID,
			&subject.Name,
			&subject.Description,
			&subject.Price,
			&subject.Duration,
			&subject.IsActive,
			&subject.RequiresBookingApproval,
			&subject.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan subject: %w", err)
		}
		subjects = append(subjects, &subject)
	}

	return subjects, nil
}

// GetActiveByTeacherIDs получает активные предметы списка учителей
func (r *SubjectRepository) GetActiveByTeacherIDs(ctx context.Context, teacherIDs []int64) ([]*model.Subject, error) {
	if len(teacherIDs) == 0 {
		return []*model.Subject{}, nil
	}

	query := `
		SELECT id, teacher_id, name, description, price, duration, is_active, requires_booking_approval, created_at
		FROM subjects
		WHERE teacher_id = ANY($1) AND is_active = true
		ORDER BY teacher_id, name
	`

	rows, err := r.pool.Query(ctx, query, teacherIDs)
	if err != nil {
		return nil, fmt.Errorf("get active subjects by teacher ids: %w", err)
	}
	defer rows.Close()

	var subjects []*model.Subject
	for rows.Next() {
		var subject model.Subject
		err := rows.Scan(
			&subject.ID,
			&subject.TeacherID,
			&subject.Name,
			&subject.Description,
			&subject.Price,
			&subject.Duration,
			&subject.IsActive,
			&subject.RequiresBookingApproval,
			&subject.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan subject: %w", err)
		}
		subjects = append(subjects, &subject)
	}

	return subjects, nil
}
