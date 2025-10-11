package app

import (
	"context"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/service"
	"go.uber.org/zap"
)

// Scheduler управляет фоновыми задачами
type Scheduler struct {
	teacherService *service.TeacherService
	logger         *zap.Logger
	stopChan       chan struct{}
}

// NewScheduler создаёт новый планировщик
func NewScheduler(teacherService *service.TeacherService, logger *zap.Logger) *Scheduler {
	return &Scheduler{
		teacherService: teacherService,
		logger:         logger,
		stopChan:       make(chan struct{}),
	}
}

// Start запускает фоновые задачи
func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info("Starting background scheduler")

	// Запускаем задачу генерации слотов
	go s.runSlotGenerationTask(ctx)
}

// Stop останавливает фоновые задачи
func (s *Scheduler) Stop() {
	s.logger.Info("Stopping background scheduler")
	close(s.stopChan)
}

// runSlotGenerationTask периодически генерирует слоты для recurring schedules
func (s *Scheduler) runSlotGenerationTask(ctx context.Context) {
	// Первый запуск сразу при старте
	s.generateSlots(ctx)

	// Создаём ticker для периодического запуска (каждые 24 часа)
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.generateSlots(ctx)
		case <-s.stopChan:
			s.logger.Info("Slot generation task stopped")
			return
		case <-ctx.Done():
			s.logger.Info("Slot generation task cancelled")
			return
		}
	}
}

// generateSlots генерирует слоты для всех активных recurring schedules
func (s *Scheduler) generateSlots(ctx context.Context) {
	s.logger.Info("Starting automatic slot generation")

	// Генерируем слоты на 4 недели вперёд
	// Это означает, что слоты всегда будут доступны минимум на месяц вперёд
	err := s.teacherService.GenerateSlotsForAllRecurringSchedules(ctx, 4)
	if err != nil {
		s.logger.Error("Failed to generate slots", zap.Error(err))
		return
	}

	s.logger.Info("Automatic slot generation completed successfully")
}
