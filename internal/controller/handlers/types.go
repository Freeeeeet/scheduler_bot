package handlers

import (
	"github.com/Freeeeeet/scheduler_bot/internal/controller/state"
	"github.com/Freeeeeet/scheduler_bot/internal/service"
	"go.uber.org/zap"
)

// Handlers содержит все зависимости для обработки команд
type Handlers struct {
	userService    *service.UserService
	bookingService *service.BookingService
	teacherService *service.TeacherService
	stateManager   *state.Manager
	logger         *zap.Logger
}

// NewHandlers создаёт новый обработчик команд
func NewHandlers(
	userService *service.UserService,
	bookingService *service.BookingService,
	teacherService *service.TeacherService,
	stateManager *state.Manager,
	logger *zap.Logger,
) *Handlers {
	return &Handlers{
		userService:    userService,
		bookingService: bookingService,
		teacherService: teacherService,
		stateManager:   stateManager,
		logger:         logger,
	}
}
