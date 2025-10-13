package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/Freeeeeet/scheduler_bot/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TeacherService struct {
	userRepo      *repository.UserRepository
	subjectRepo   *repository.SubjectRepository
	slotRepo      *repository.SlotRepository
	bookingRepo   *repository.BookingRepository
	recurringRepo *repository.RecurringScheduleRepository
	logger        *zap.Logger
}

func NewTeacherService(
	userRepo *repository.UserRepository,
	subjectRepo *repository.SubjectRepository,
	slotRepo *repository.SlotRepository,
	bookingRepo *repository.BookingRepository,
	recurringRepo *repository.RecurringScheduleRepository,
	logger *zap.Logger,
) *TeacherService {
	return &TeacherService{
		userRepo:      userRepo,
		subjectRepo:   subjectRepo,
		slotRepo:      slotRepo,
		bookingRepo:   bookingRepo,
		recurringRepo: recurringRepo,
		logger:        logger,
	}
}

// CreateSubject создаёт новый предмет для учителя
func (s *TeacherService) CreateSubject(ctx context.Context, teacherID int64, name, description string, price, duration int, requiresApproval bool) (*model.Subject, error) {
	s.logger.Info("CreateSubject called",
		zap.Int64("teacher_id", teacherID),
		zap.String("name", name),
		zap.String("description", description),
		zap.Int("price", price),
		zap.Int("duration", duration),
		zap.Bool("requires_approval", requiresApproval))

	// Проверяем что пользователь - учитель
	teacher, err := s.userRepo.GetByID(ctx, teacherID)
	if err != nil {
		s.logger.Error("Failed to get teacher",
			zap.Int64("teacher_id", teacherID),
			zap.Error(err))
		return nil, fmt.Errorf("get teacher: %w", err)
	}

	if teacher == nil {
		s.logger.Error("Teacher not found",
			zap.Int64("teacher_id", teacherID))
		return nil, fmt.Errorf("teacher not found")
	}

	if !teacher.IsTeacher {
		s.logger.Error("User is not a teacher",
			zap.Int64("teacher_id", teacherID),
			zap.Bool("is_teacher", teacher.IsTeacher))
		return nil, fmt.Errorf("user is not a teacher")
	}

	s.logger.Info("Teacher verified, creating subject",
		zap.Int64("teacher_id", teacherID),
		zap.String("teacher_name", teacher.FirstName))

	// Создаём предмет
	subject := &model.Subject{
		TeacherID:               teacherID,
		Name:                    name,
		Description:             description,
		Price:                   price,
		Duration:                duration,
		IsActive:                true,
		RequiresBookingApproval: requiresApproval,
	}

	s.logger.Info("Calling subjectRepo.Create",
		zap.Int64("teacher_id", teacherID),
		zap.String("name", name))

	err = s.subjectRepo.Create(ctx, subject)
	if err != nil {
		s.logger.Error("Failed to create subject in DB",
			zap.Int64("teacher_id", teacherID),
			zap.String("name", name),
			zap.Error(err))
		return nil, fmt.Errorf("create subject: %w", err)
	}

	s.logger.Info("Subject created successfully",
		zap.Int64("subject_id", subject.ID),
		zap.Int64("teacher_id", teacherID),
		zap.String("name", name),
		zap.Bool("requires_approval", requiresApproval),
	)

	return subject, nil
}

// GetTeacherSubjects получает все предметы учителя
func (s *TeacherService) GetTeacherSubjects(ctx context.Context, teacherID int64) ([]*model.Subject, error) {
	return s.subjectRepo.GetByTeacherID(ctx, teacherID)
}

// GetAllActiveSubjects получает все активные предметы
func (s *TeacherService) GetAllActiveSubjects(ctx context.Context) ([]*model.Subject, error) {
	return s.subjectRepo.GetActive(ctx)
}

// GetSubjectByID получает предмет по ID
func (s *TeacherService) GetSubjectByID(ctx context.Context, id int64) (*model.Subject, error) {
	return s.subjectRepo.GetByID(ctx, id)
}

// ToggleSubjectActive переключает активность предмета
func (s *TeacherService) ToggleSubjectActive(ctx context.Context, teacherID, subjectID int64) (*model.Subject, error) {
	subject, err := s.subjectRepo.GetByID(ctx, subjectID)
	if err != nil {
		return nil, fmt.Errorf("get subject: %w", err)
	}

	if subject == nil {
		return nil, fmt.Errorf("subject not found")
	}

	if subject.TeacherID != teacherID {
		return nil, fmt.Errorf("subject does not belong to teacher")
	}

	// Переключаем активность
	subject.IsActive = !subject.IsActive

	err = s.subjectRepo.Update(ctx, subject)
	if err != nil {
		return nil, fmt.Errorf("update subject: %w", err)
	}

	s.logger.Info("Subject active toggled",
		zap.Int64("subject_id", subjectID),
		zap.Bool("is_active", subject.IsActive),
	)

	return subject, nil
}

// UpdateSubject обновляет предмет
func (s *TeacherService) UpdateSubject(ctx context.Context, teacherID int64, subject *model.Subject) error {
	existing, err := s.subjectRepo.GetByID(ctx, subject.ID)
	if err != nil {
		return fmt.Errorf("get subject: %w", err)
	}

	if existing == nil {
		return fmt.Errorf("subject not found")
	}

	if existing.TeacherID != teacherID {
		return fmt.Errorf("subject does not belong to teacher")
	}

	err = s.subjectRepo.Update(ctx, subject)
	if err != nil {
		return fmt.Errorf("update subject: %w", err)
	}

	s.logger.Info("Subject updated",
		zap.Int64("subject_id", subject.ID),
	)

	return nil
}

// DeleteSubject удаляет предмет
func (s *TeacherService) DeleteSubject(ctx context.Context, teacherID, subjectID int64) error {
	subject, err := s.subjectRepo.GetByID(ctx, subjectID)
	if err != nil {
		return fmt.Errorf("get subject: %w", err)
	}

	if subject == nil {
		return fmt.Errorf("subject not found")
	}

	if subject.TeacherID != teacherID {
		return fmt.Errorf("subject does not belong to teacher")
	}

	// Удаляем предмет (слоты и бронирования удалятся каскадом)
	err = s.subjectRepo.Delete(ctx, subjectID)
	if err != nil {
		return fmt.Errorf("delete subject: %w", err)
	}

	s.logger.Info("Subject deleted",
		zap.Int64("subject_id", subjectID),
		zap.Int64("teacher_id", teacherID),
	)

	return nil
}

// CreateSlot создаёт временной слот
func (s *TeacherService) CreateSlot(ctx context.Context, teacherID, subjectID int64, startTime, endTime time.Time) (*model.ScheduleSlot, error) {
	// Проверяем что предмет принадлежит учителю
	subject, err := s.subjectRepo.GetByID(ctx, subjectID)
	if err != nil {
		return nil, fmt.Errorf("get subject: %w", err)
	}

	if subject == nil {
		return nil, fmt.Errorf("subject not found")
	}

	if subject.TeacherID != teacherID {
		return nil, fmt.Errorf("subject does not belong to teacher")
	}

	// Валидация времени
	if endTime.Before(startTime) || endTime.Equal(startTime) {
		return nil, fmt.Errorf("end time must be after start time")
	}

	if startTime.Before(time.Now()) {
		return nil, fmt.Errorf("cannot create slot in the past")
	}

	// Создаём слот
	slot := &model.ScheduleSlot{
		TeacherID: teacherID,
		SubjectID: subjectID,
		StartTime: startTime,
		EndTime:   endTime,
		Status:    model.SlotStatusFree,
		StudentID: nil,
	}

	err = s.slotRepo.Create(ctx, slot)
	if err != nil {
		return nil, fmt.Errorf("create slot: %w", err)
	}

	s.logger.Info("Slot created",
		zap.Int64("slot_id", slot.ID),
		zap.Int64("teacher_id", teacherID),
		zap.Time("start_time", startTime),
	)

	return slot, nil
}

// GetTeacherSchedule получает расписание учителя за период
func (s *TeacherService) GetTeacherSchedule(ctx context.Context, teacherID int64, from, to time.Time) ([]*model.ScheduleSlot, error) {
	return s.slotRepo.GetByTeacherID(ctx, teacherID, from, to)
}

// GetTeacherBookings получает все бронирования учителя
func (s *TeacherService) GetTeacherBookings(ctx context.Context, teacherID int64) ([]*model.Booking, error) {
	return s.bookingRepo.GetByTeacherID(ctx, teacherID)
}

// CreateWeeklySlots создаёт регулярное расписание (recurring schedule) и первичные слоты
// Этот метод устарел, используйте CreateWeeklySlotsGroup для создания группы расписаний
func (s *TeacherService) CreateWeeklySlots(ctx context.Context, teacherID, subjectID int64, weekday time.Weekday, startHour, startMinute, durationMinutes int) error {
	// Проверяем что предмет принадлежит учителю
	subject, err := s.subjectRepo.GetByID(ctx, subjectID)
	if err != nil {
		return fmt.Errorf("get subject: %w", err)
	}

	if subject == nil {
		return fmt.Errorf("subject not found")
	}

	if subject.TeacherID != teacherID {
		return fmt.Errorf("subject does not belong to teacher")
	}

	// Создаём recurring schedule (шаблон регулярного расписания)
	recurringSchedule := &model.RecurringSchedule{
		GroupID:         uuid.New(), // Генерируем уникальный group_id для одиночного расписания
		TeacherID:       teacherID,
		SubjectID:       subjectID,
		Weekday:         int(weekday),
		StartHour:       startHour,
		StartMinute:     startMinute,
		DurationMinutes: durationMinutes,
		IsActive:        true,
	}

	err = s.recurringRepo.Create(ctx, recurringSchedule)
	if err != nil {
		return fmt.Errorf("create recurring schedule: %w", err)
	}

	s.logger.Info("Recurring schedule created",
		zap.Int64("recurring_schedule_id", recurringSchedule.ID),
		zap.String("group_id", recurringSchedule.GroupID.String()),
		zap.Int64("teacher_id", teacherID),
		zap.Int64("subject_id", subjectID),
		zap.Int("weekday", int(weekday)),
	)

	// Создаём начальные слоты на следующие 4 недели
	count, err := s.generateSlotsForRecurringSchedule(ctx, recurringSchedule, 4)
	if err != nil {
		s.logger.Error("Failed to generate initial slots", zap.Error(err))
		// Не возвращаем ошибку, т.к. recurring schedule уже создан
	}

	s.logger.Info("Initial weekly slots created",
		zap.Int64("recurring_schedule_id", recurringSchedule.ID),
		zap.Int("count", count),
	)

	return nil
}

// CreateWeeklySlotsGroup создаёт группу регулярных расписаний с общим group_id
// weekdays - массив дней недели (0 = Sunday, 6 = Saturday)
// timeSlots - массив временных слотов (час и минута)
func (s *TeacherService) CreateWeeklySlotsGroup(ctx context.Context, teacherID, subjectID int64, weekdays []int, timeSlots []struct{ Hour, Minute int }, durationMinutes int) (uuid.UUID, error) {
	// Проверяем что предмет принадлежит учителю
	subject, err := s.subjectRepo.GetByID(ctx, subjectID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get subject: %w", err)
	}

	if subject == nil {
		return uuid.Nil, fmt.Errorf("subject not found")
	}

	if subject.TeacherID != teacherID {
		return uuid.Nil, fmt.Errorf("subject does not belong to teacher")
	}

	// Генерируем общий group_id для всей группы
	groupID := uuid.New()

	// Создаём recurring schedules для каждого дня и каждого времени
	createdCount := 0
	for _, weekday := range weekdays {
		for _, slot := range timeSlots {
			recurringSchedule := &model.RecurringSchedule{
				GroupID:         groupID,
				TeacherID:       teacherID,
				SubjectID:       subjectID,
				Weekday:         weekday,
				StartHour:       slot.Hour,
				StartMinute:     slot.Minute,
				DurationMinutes: durationMinutes,
				IsActive:        true,
			}

			err = s.recurringRepo.Create(ctx, recurringSchedule)
			if err != nil {
				s.logger.Error("Failed to create recurring schedule",
					zap.Error(err),
					zap.String("group_id", groupID.String()),
					zap.Int("weekday", weekday),
					zap.Int("hour", slot.Hour),
					zap.Int("minute", slot.Minute))
				continue
			}

			// Создаём начальные слоты на следующие 4 недели
			count, err := s.generateSlotsForRecurringSchedule(ctx, recurringSchedule, 4)
			if err != nil {
				s.logger.Error("Failed to generate initial slots",
					zap.Error(err),
					zap.Int64("recurring_schedule_id", recurringSchedule.ID))
			} else {
				s.logger.Debug("Generated initial slots",
					zap.Int64("recurring_schedule_id", recurringSchedule.ID),
					zap.Int("count", count))
			}

			createdCount++
		}
	}

	s.logger.Info("Recurring schedule group created",
		zap.String("group_id", groupID.String()),
		zap.Int64("teacher_id", teacherID),
		zap.Int64("subject_id", subjectID),
		zap.Int("weekdays_count", len(weekdays)),
		zap.Int("time_slots_count", len(timeSlots)),
		zap.Int("total_created", createdCount),
	)

	return groupID, nil
}

// generateSlotsForRecurringSchedule генерирует слоты для recurring schedule на указанное количество недель
func (s *TeacherService) generateSlotsForRecurringSchedule(ctx context.Context, schedule *model.RecurringSchedule, weeksAhead int) (int, error) {
	now := time.Now()
	location := now.Location()
	weekday := time.Weekday(schedule.Weekday)

	count := 0
	daysToCheck := weeksAhead * 7

	for i := 0; i < daysToCheck; i++ {
		date := now.AddDate(0, 0, i)

		if date.Weekday() == weekday {
			startTime := time.Date(date.Year(), date.Month(), date.Day(),
				schedule.StartHour, schedule.StartMinute, 0, 0, location)
			endTime := startTime.Add(time.Duration(schedule.DurationMinutes) * time.Minute)

			// Пропускаем прошедшие слоты
			if startTime.Before(now) {
				continue
			}

			// Проверяем, не существует ли уже такой слот
			exists, err := s.slotRepo.SlotExists(ctx, schedule.TeacherID, startTime)
			if err != nil {
				s.logger.Warn("Failed to check slot existence",
					zap.Error(err),
					zap.Time("start_time", startTime),
				)
				continue
			}

			if exists {
				s.logger.Debug("Slot already exists, skipping",
					zap.Time("start_time", startTime),
				)
				continue
			}

			slot := &model.ScheduleSlot{
				TeacherID: schedule.TeacherID,
				SubjectID: schedule.SubjectID,
				StartTime: startTime,
				EndTime:   endTime,
				Status:    model.SlotStatusFree,
				StudentID: nil,
			}

			err = s.slotRepo.Create(ctx, slot)
			if err != nil {
				s.logger.Warn("Failed to create slot",
					zap.Error(err),
					zap.Time("start_time", startTime),
				)
				continue
			}

			count++
		}
	}

	return count, nil
}

// GenerateSlotsForAllRecurringSchedules генерирует слоты для всех активных recurring schedules
// Эта функция будет вызываться периодически (например, раз в день)
func (s *TeacherService) GenerateSlotsForAllRecurringSchedules(ctx context.Context, weeksAhead int) error {
	schedules, err := s.recurringRepo.GetAllActive(ctx)
	if err != nil {
		return fmt.Errorf("get all active recurring schedules: %w", err)
	}

	totalCount := 0
	for _, schedule := range schedules {
		count, err := s.generateSlotsForRecurringSchedule(ctx, schedule, weeksAhead)
		if err != nil {
			s.logger.Error("Failed to generate slots for recurring schedule",
				zap.Error(err),
				zap.Int64("recurring_schedule_id", schedule.ID),
			)
			continue
		}
		totalCount += count
	}

	s.logger.Info("Generated slots for all recurring schedules",
		zap.Int("total_schedules", len(schedules)),
		zap.Int("total_slots_created", totalCount),
	)

	return nil
}

// GetRecurringSchedules возвращает все recurring schedules учителя
func (s *TeacherService) GetRecurringSchedules(ctx context.Context, teacherID int64) ([]*model.RecurringSchedule, error) {
	return s.recurringRepo.GetByTeacherID(ctx, teacherID)
}

// GetRecurringSchedulesBySubject возвращает recurring schedules для предмета
func (s *TeacherService) GetRecurringSchedulesBySubject(ctx context.Context, subjectID int64) ([]*model.RecurringSchedule, error) {
	return s.recurringRepo.GetBySubjectID(ctx, subjectID)
}

// GetRecurringScheduleByID возвращает recurring schedule по ID
func (s *TeacherService) GetRecurringScheduleByID(ctx context.Context, scheduleID int64) (*model.RecurringSchedule, error) {
	return s.recurringRepo.GetByID(ctx, scheduleID)
}

// DeactivateRecurringSchedule деактивирует recurring schedule
func (s *TeacherService) DeactivateRecurringSchedule(ctx context.Context, teacherID, scheduleID int64) error {
	schedule, err := s.recurringRepo.GetByID(ctx, scheduleID)
	if err != nil {
		return fmt.Errorf("get recurring schedule: %w", err)
	}

	if schedule == nil {
		return fmt.Errorf("recurring schedule not found")
	}

	if schedule.TeacherID != teacherID {
		return fmt.Errorf("recurring schedule does not belong to teacher")
	}

	return s.recurringRepo.Deactivate(ctx, scheduleID)
}

// DeleteRecurringSchedule удаляет recurring schedule
func (s *TeacherService) DeleteRecurringSchedule(ctx context.Context, teacherID, scheduleID int64) error {
	schedule, err := s.recurringRepo.GetByID(ctx, scheduleID)
	if err != nil {
		return fmt.Errorf("get recurring schedule: %w", err)
	}

	if schedule == nil {
		return fmt.Errorf("recurring schedule not found")
	}

	if schedule.TeacherID != teacherID {
		return fmt.Errorf("recurring schedule does not belong to teacher")
	}

	return s.recurringRepo.Delete(ctx, scheduleID)
}

// GetSlotByID получает слот по ID
func (s *TeacherService) GetSlotByID(ctx context.Context, slotID int64) (*model.ScheduleSlot, error) {
	return s.slotRepo.GetByID(ctx, slotID)
}

// CancelSlot отменяет свободный слот
func (s *TeacherService) CancelSlot(ctx context.Context, slotID int64) error {
	slot, err := s.slotRepo.GetByID(ctx, slotID)
	if err != nil {
		return fmt.Errorf("get slot: %w", err)
	}

	if slot == nil {
		return fmt.Errorf("slot not found")
	}

	if slot.Status != model.SlotStatusFree {
		return fmt.Errorf("can only cancel free slots")
	}

	err = s.slotRepo.Cancel(ctx, slotID)
	if err != nil {
		return fmt.Errorf("cancel slot: %w", err)
	}

	s.logger.Info("Slot canceled",
		zap.Int64("slot_id", slotID),
		zap.Int64("teacher_id", slot.TeacherID),
	)

	return nil
}

// RestoreSlot восстанавливает отменённый слот
func (s *TeacherService) RestoreSlot(ctx context.Context, slotID int64) error {
	slot, err := s.slotRepo.GetByID(ctx, slotID)
	if err != nil {
		return fmt.Errorf("get slot: %w", err)
	}

	if slot == nil {
		return fmt.Errorf("slot not found")
	}

	if slot.Status != model.SlotStatusCanceled {
		return fmt.Errorf("can only restore canceled slots")
	}

	// Восстанавливаем слот как свободный
	err = s.slotRepo.UpdateStatus(ctx, slotID, model.SlotStatusFree)
	if err != nil {
		return fmt.Errorf("restore slot: %w", err)
	}

	s.logger.Info("Slot restored",
		zap.Int64("slot_id", slotID),
		zap.Int64("teacher_id", slot.TeacherID),
	)

	return nil
}

// CancelBookingBySlot отменяет бронирование по слоту
func (s *TeacherService) CancelBookingBySlot(ctx context.Context, slotID int64, teacherID int64) error {
	slot, err := s.slotRepo.GetByID(ctx, slotID)
	if err != nil {
		return fmt.Errorf("get slot: %w", err)
	}

	if slot == nil {
		return fmt.Errorf("slot not found")
	}

	if slot.TeacherID != teacherID {
		return fmt.Errorf("slot does not belong to teacher")
	}

	if slot.Status != model.SlotStatusBooked {
		return fmt.Errorf("slot is not booked")
	}

	// Получаем активное бронирование для этого слота
	activeBooking, err := s.bookingRepo.GetBySlotID(ctx, slotID)
	if err != nil {
		return fmt.Errorf("get booking: %w", err)
	}

	if activeBooking == nil {
		return fmt.Errorf("no active booking found for this slot")
	}

	// Отменяем бронирование
	err = s.bookingRepo.UpdateStatus(ctx, activeBooking.ID, model.BookingStatusCanceled)
	if err != nil {
		return fmt.Errorf("update booking status: %w", err)
	}

	// Освобождаем слот
	err = s.slotRepo.Cancel(ctx, slotID)
	if err != nil {
		return fmt.Errorf("cancel slot: %w", err)
	}

	s.logger.Info("Booking canceled by teacher",
		zap.Int64("booking_id", activeBooking.ID),
		zap.Int64("slot_id", slotID),
		zap.Int64("teacher_id", teacherID),
	)

	return nil
}

// GetRecurringSchedulesByGroupID получает все recurring schedules по group_id
func (s *TeacherService) GetRecurringSchedulesByGroupID(ctx context.Context, groupID string) ([]*model.RecurringSchedule, error) {
	return s.recurringRepo.GetByGroupID(ctx, groupID)
}

// DeactivateRecurringScheduleGroup деактивирует всю группу recurring schedules
func (s *TeacherService) DeactivateRecurringScheduleGroup(ctx context.Context, teacherID int64, groupID string) error {
	// Проверяем что группа принадлежит учителю
	schedules, err := s.recurringRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("get recurring schedules by group_id: %w", err)
	}

	if len(schedules) == 0 {
		return fmt.Errorf("recurring schedule group not found")
	}

	if schedules[0].TeacherID != teacherID {
		return fmt.Errorf("recurring schedule group does not belong to teacher")
	}

	err = s.recurringRepo.DeactivateByGroupID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("deactivate recurring schedule group: %w", err)
	}

	s.logger.Info("Recurring schedule group deactivated",
		zap.String("group_id", groupID),
		zap.Int64("teacher_id", teacherID),
	)

	return nil
}

// DeleteRecurringScheduleGroup удаляет всю группу recurring schedules
func (s *TeacherService) DeleteRecurringScheduleGroup(ctx context.Context, teacherID int64, groupID string) error {
	// Проверяем что группа принадлежит учителю
	schedules, err := s.recurringRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("get recurring schedules by group_id: %w", err)
	}

	if len(schedules) == 0 {
		return fmt.Errorf("recurring schedule group not found")
	}

	if schedules[0].TeacherID != teacherID {
		return fmt.Errorf("recurring schedule group does not belong to teacher")
	}

	err = s.recurringRepo.DeleteByGroupID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("delete recurring schedule group: %w", err)
	}

	s.logger.Info("Recurring schedule group deleted",
		zap.String("group_id", groupID),
		zap.Int64("teacher_id", teacherID),
	)

	return nil
}
