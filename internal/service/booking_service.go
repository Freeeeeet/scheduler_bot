package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/Freeeeeet/scheduler_bot/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type BookingService struct {
	pool        *pgxpool.Pool
	userRepo    *repository.UserRepository
	subjectRepo *repository.SubjectRepository
	slotRepo    *repository.SlotRepository
	bookingRepo *repository.BookingRepository
	logger      *zap.Logger
}

func NewBookingService(
	pool *pgxpool.Pool,
	userRepo *repository.UserRepository,
	subjectRepo *repository.SubjectRepository,
	slotRepo *repository.SlotRepository,
	bookingRepo *repository.BookingRepository,
	logger *zap.Logger,
) *BookingService {
	return &BookingService{
		pool:        pool,
		userRepo:    userRepo,
		subjectRepo: subjectRepo,
		slotRepo:    slotRepo,
		bookingRepo: bookingRepo,
		logger:      logger,
	}
}

// BookSlot бронирует слот для студента
func (s *BookingService) BookSlot(ctx context.Context, studentID, slotID int64) (*model.Booking, error) {
	// Начинаем транзакцию
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Получаем информацию о слоте
	slot, err := s.slotRepo.GetByID(ctx, slotID)
	if err != nil {
		return nil, fmt.Errorf("get slot: %w", err)
	}

	if slot == nil {
		return nil, fmt.Errorf("slot not found")
	}

	// Проверяем что слот свободен
	if slot.Status != model.SlotStatusFree {
		return nil, fmt.Errorf("slot is not available")
	}

	// Проверяем что слот в будущем
	if slot.StartTime.Before(time.Now()) {
		return nil, fmt.Errorf("slot is in the past")
	}

	// Получаем информацию о предмете
	subject, err := s.subjectRepo.GetByID(ctx, slot.SubjectID)
	if err != nil {
		return nil, fmt.Errorf("get subject: %w", err)
	}

	if subject == nil {
		return nil, fmt.Errorf("subject not found")
	}

	// Проверяем что предмет активен
	if !subject.IsActive {
		return nil, fmt.Errorf("subject is not active")
	}

	// Определяем статус бронирования в зависимости от настроек предмета
	bookingStatus := model.BookingStatusConfirmed
	if subject.RequiresBookingApproval {
		bookingStatus = model.BookingStatusPending
	}

	// Бронируем слот (временно, до подтверждения)
	err = s.slotRepo.Book(ctx, slotID, studentID)
	if err != nil {
		return nil, fmt.Errorf("book slot: %w", err)
	}

	// Создаём запись о бронировании
	booking := &model.Booking{
		StudentID: studentID,
		TeacherID: slot.TeacherID,
		SubjectID: slot.SubjectID,
		SlotID:    slotID,
		Status:    bookingStatus,
	}

	err = s.bookingRepo.Create(ctx, booking)
	if err != nil {
		return nil, fmt.Errorf("create booking: %w", err)
	}

	// Коммитим транзакцию
	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	s.logger.Info("Slot booked",
		zap.Int64("booking_id", booking.ID),
		zap.Int64("student_id", studentID),
		zap.Int64("slot_id", slotID),
		zap.String("subject", subject.Name),
		zap.String("status", string(bookingStatus)),
	)

	// Возвращаем бронирование с заполненными данными для уведомлений
	booking.Subject = subject
	booking.Slot = slot

	return booking, nil
}

// GetPendingBookings получает все pending бронирования учителя
func (s *BookingService) GetPendingBookings(ctx context.Context, teacherID int64) ([]*model.Booking, error) {
	return s.bookingRepo.GetPendingByTeacherID(ctx, teacherID)
}

// ApproveBooking одобряет бронирование
func (s *BookingService) ApproveBooking(ctx context.Context, bookingID, teacherID int64) error {
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("get booking: %w", err)
	}

	if booking == nil {
		return fmt.Errorf("booking not found")
	}

	if booking.TeacherID != teacherID {
		return fmt.Errorf("no permission to approve this booking")
	}

	if booking.Status != model.BookingStatusPending {
		return fmt.Errorf("booking is not pending")
	}

	err = s.bookingRepo.UpdateStatus(ctx, bookingID, model.BookingStatusConfirmed)
	if err != nil {
		return fmt.Errorf("update booking status: %w", err)
	}

	s.logger.Info("Booking approved",
		zap.Int64("booking_id", bookingID),
		zap.Int64("teacher_id", teacherID),
	)

	return nil
}

// RejectBooking отклоняет бронирование
func (s *BookingService) RejectBooking(ctx context.Context, bookingID, teacherID int64) error {
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("get booking: %w", err)
	}

	if booking == nil {
		return fmt.Errorf("booking not found")
	}

	if booking.TeacherID != teacherID {
		return fmt.Errorf("no permission to reject this booking")
	}

	if booking.Status != model.BookingStatusPending {
		return fmt.Errorf("booking is not pending")
	}

	// Начинаем транзакцию
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Обновляем статус
	err = s.bookingRepo.UpdateStatus(ctx, bookingID, model.BookingStatusRejected)
	if err != nil {
		return fmt.Errorf("update booking status: %w", err)
	}

	// Освобождаем слот
	err = s.slotRepo.Cancel(ctx, booking.SlotID)
	if err != nil {
		return fmt.Errorf("cancel slot: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	s.logger.Info("Booking rejected",
		zap.Int64("booking_id", bookingID),
		zap.Int64("teacher_id", teacherID),
	)

	return nil
}

// GetBookingsBySubject получает все активные бронирования для предмета
func (s *BookingService) GetBookingsBySubject(ctx context.Context, subjectID int64) ([]*model.Booking, error) {
	return s.bookingRepo.GetBySubjectID(ctx, subjectID)
}

// GetByID получает бронирование по ID
func (s *BookingService) GetByID(ctx context.Context, bookingID int64) (*model.Booking, error) {
	return s.bookingRepo.GetByID(ctx, bookingID)
}

// GetAvailableSlots получает доступные слоты для предмета
func (s *BookingService) GetAvailableSlots(ctx context.Context, subjectID int64, from, to time.Time) ([]*model.ScheduleSlot, error) {
	return s.slotRepo.GetFreeSlots(ctx, subjectID, from, to)
}

// GetStudentBookings получает все бронирования студента
func (s *BookingService) GetStudentBookings(ctx context.Context, studentID int64) ([]*model.Booking, error) {
	return s.bookingRepo.GetByStudentID(ctx, studentID)
}

// CancelBooking отменяет бронирование
func (s *BookingService) CancelBooking(ctx context.Context, bookingID, userID int64) error {
	// Получаем бронирование
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("get booking: %w", err)
	}

	if booking == nil {
		return fmt.Errorf("booking not found")
	}

	// Проверяем что пользователь имеет право отменить
	if booking.StudentID != userID && booking.TeacherID != userID {
		return fmt.Errorf("no permission to cancel this booking")
	}

	// Проверяем что бронирование активно
	if booking.Status != model.BookingStatusConfirmed && booking.Status != model.BookingStatusPending {
		return fmt.Errorf("booking is not active")
	}

	// Начинаем транзакцию
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Обновляем статус бронирования
	err = s.bookingRepo.UpdateStatus(ctx, bookingID, model.BookingStatusCanceled)
	if err != nil {
		return fmt.Errorf("update booking status: %w", err)
	}

	// Освобождаем слот
	err = s.slotRepo.Cancel(ctx, booking.SlotID)
	if err != nil {
		return fmt.Errorf("cancel slot: %w", err)
	}

	// Коммитим транзакцию
	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	s.logger.Info("Booking canceled",
		zap.Int64("booking_id", bookingID),
		zap.Int64("user_id", userID),
	)

	return nil
}
