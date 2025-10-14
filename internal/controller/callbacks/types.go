package callbacks

import (
	"context"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/Freeeeeet/scheduler_bot/internal/service"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// ========================
// Handler with Dependencies
// ========================

// Handler обертка для callbacktypes.Handler с методами
type Handler struct {
	*callbacktypes.Handler
}

// StateManager интерфейс для управления состоянием пользователей
type StateManager = callbacktypes.StateManager

// UserState представляет текущее состояние пользователя в диалоге
type UserState = callbacktypes.UserState

const (
	StateNone                     UserState = "" // Нет активного состояния
	StateCreateSubjectName        UserState = "create_subject_name"
	StateCreateSubjectDescription UserState = "create_subject_description"
	StateCreateSubjectPrice       UserState = "create_subject_price"
	StateCreateSubjectDuration    UserState = "create_subject_duration"
)

// NewHandler создаёт новый обработчик callbacks с зависимостями
func NewHandler(
	userService *service.UserService,
	bookingService *service.BookingService,
	teacherService *service.TeacherService,
	accessService *service.StudentAccessService,
	userRepo interface {
		GetByID(ctx context.Context, id int64) (*model.User, error)
		UpdatePublicStatus(ctx context.Context, userID int64, isPublic bool) error
	},
	inviteCodeRepo interface {
		GetByCode(ctx context.Context, code string) (*model.TeacherInviteCode, error)
		CountActiveCodesByTeacher(ctx context.Context, teacherID int64) (int, error)
	},
	accessRepo interface {
		GetAccessInfo(ctx context.Context, studentID, teacherID int64) (*model.StudentTeacherAccess, error)
	},
	accessRequestRepo interface {
		GetByID(ctx context.Context, id int64) (*model.AccessRequest, error)
	},
	stateManager callbacktypes.StateManager,
	logger *zap.Logger,
	handleSubjects func(ctx context.Context, b *bot.Bot, update *models.Update),
	handleMySchedule func(ctx context.Context, b *bot.Bot, update *models.Update),
	handleMySubjects func(ctx context.Context, b *bot.Bot, update *models.Update),
) *Handler {
	inner := &callbacktypes.Handler{
		UserService:       userService,
		BookingService:    bookingService,
		TeacherService:    teacherService,
		AccessService:     accessService,
		UserRepo:          userRepo,
		InviteCodeRepo:    inviteCodeRepo,
		AccessRepo:        accessRepo,
		AccessRequestRepo: accessRequestRepo,
		StateManager:      stateManager,
		Logger:            logger,
		HandleSubjects:    handleSubjects,
		HandleMySchedule:  handleMySchedule,
		HandleMySubjects:  handleMySubjects,
	}
	return &Handler{Handler: inner}
}

// HandleCallbackQuery - главный обработчик callback queries
func (h *Handler) HandleCallbackQuery(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	callback := update.CallbackQuery
	data := callback.Data

	h.Logger.Info("Callback received",
		zap.String("data", data),
		zap.Int64("user_id", callback.From.ID),
	)

	// Вызываем роутер
	Route(ctx, b, callback, h.Handler)
}
