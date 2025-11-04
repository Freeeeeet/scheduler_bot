package callbacktypes

import (
	"context"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/Freeeeeet/scheduler_bot/internal/service"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// UserState представляет текущее состояние пользователя в диалоге
type UserState string

// StateManager интерфейс для управления состоянием пользователей
type StateManager interface {
	ClearState(telegramID int64)
	GetState(telegramID int64) UserState
	SetState(telegramID int64, state UserState)
	SetData(telegramID int64, key string, value interface{})
	GetData(telegramID int64, key string) (interface{}, bool)
	GetAllData(telegramID int64) map[string]interface{}
}

// Handler содержит общие зависимости для всех callback handlers
type Handler struct {
	UserService    *service.UserService
	BookingService *service.BookingService
	TeacherService *service.TeacherService
	AccessService  *service.StudentAccessService
	StateManager   StateManager
	Logger         *zap.Logger

	// Репозитории (для прямого доступа в некоторых handlers)
	UserRepo interface {
		GetByID(ctx context.Context, id int64) (*model.User, error)
		UpdatePublicStatus(ctx context.Context, userID int64, isPublic bool) error
	}
	InviteCodeRepo interface {
		GetByCode(ctx context.Context, code string) (*model.TeacherInviteCode, error)
		CountActiveCodesByTeacher(ctx context.Context, teacherID int64) (int, error)
	}
	AccessRepo interface {
		GetAccessInfo(ctx context.Context, studentID, teacherID int64) (*model.StudentTeacherAccess, error)
	}
	AccessRequestRepo interface {
		GetByID(ctx context.Context, id int64) (*model.AccessRequest, error)
	}

	// Функции-хэндлеры из основного контроллера
	HandleSubjects   func(ctx context.Context, b *bot.Bot, update *models.Update)
	HandleMySchedule func(ctx context.Context, b *bot.Bot, update *models.Update)
	HandleMySubjects func(ctx context.Context, b *bot.Bot, update *models.Update, messageID ...int)
}
