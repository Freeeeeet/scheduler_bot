package controller

import (
	"context"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/handlers"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/state"
	"github.com/Freeeeeet/scheduler_bot/internal/service"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

type BotController struct {
	bot             *bot.Bot
	handlers        *handlers.Handlers
	callbackHandler *callbacks.Handler
	logger          *zap.Logger
}

func NewBotController(
	botInstance *bot.Bot,
	userService *service.UserService,
	bookingService *service.BookingService,
	teacherService *service.TeacherService,
	logger *zap.Logger,
) *BotController {
	// Создаём менеджер состояний
	stateManager := state.NewManager()

	// Создаём обработчики команд
	cmdHandlers := handlers.NewHandlers(
		userService,
		bookingService,
		teacherService,
		stateManager,
		logger,
	)

	// Создаём адаптер для callback handlers
	stateAdapter := state.NewAdapter(stateManager)

	// Создаём callback handler с зависимостями
	callbackHandler := callbacks.NewHandler(
		userService,
		bookingService,
		teacherService,
		stateAdapter,
		logger,
		cmdHandlers.HandleSubjects,
		cmdHandlers.HandleMySchedule,
	)

	return &BotController{
		bot:             botInstance,
		handlers:        cmdHandlers,
		callbackHandler: callbackHandler,
		logger:          logger,
	}
}

// RegisterHandlers регистрирует все обработчики команд
func (c *BotController) RegisterHandlers(ctx context.Context) error {
	// Регистрируем команды
	c.bot.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, c.handlers.HandleStart)
	c.bot.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, c.handlers.HandleHelp)
	c.bot.RegisterHandler(bot.HandlerTypeMessageText, "/subjects", bot.MatchTypeExact, c.handlers.HandleSubjects)
	c.bot.RegisterHandler(bot.HandlerTypeMessageText, "/mybookings", bot.MatchTypeExact, c.handlers.HandleMyBookings)
	c.bot.RegisterHandler(bot.HandlerTypeMessageText, "/cancel", bot.MatchTypeExact, c.handlers.HandleCancel)

	// Команды для учителей
	c.bot.RegisterHandler(bot.HandlerTypeMessageText, "/becometeacher", bot.MatchTypeExact, c.handlers.HandleBecomeTeacher)
	c.bot.RegisterHandler(bot.HandlerTypeMessageText, "/mysubjects", bot.MatchTypeExact, c.handlers.HandleMySubjects)
	c.bot.RegisterHandler(bot.HandlerTypeMessageText, "/myschedule", bot.MatchTypeExact, c.handlers.HandleMySchedule)
	c.bot.RegisterHandler(bot.HandlerTypeMessageText, "/createsubject", bot.MatchTypeExact, c.handlers.HandleCreateSubjectStart)

	// Обработчик текстовых сообщений (для диалогов с состояниями)
	c.bot.RegisterHandler(bot.HandlerTypeMessageText, "", bot.MatchTypePrefix, c.handlers.HandleTextMessage)

	// Обработчик нажатий на inline кнопки
	c.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "", bot.MatchTypePrefix, c.callbackHandler.HandleCallbackQuery)

	// Устанавливаем меню команд
	return c.setCommands(ctx)
}

// setCommands устанавливает список команд в меню бота
func (c *BotController) setCommands(ctx context.Context) error {
	commands := []models.BotCommand{
		{Command: "start", Description: "🚀 Начать работу с ботом"},
		{Command: "help", Description: "❓ Справка по командам"},
		{Command: "subjects", Description: "📚 Список всех предметов"},
		{Command: "mybookings", Description: "📅 Мои записи на занятия"},
		{Command: "becometeacher", Description: "🎓 Стать учителем"},
		{Command: "mysubjects", Description: "📝 Мои предметы (учитель)"},
		{Command: "myschedule", Description: "🗓 Моё расписание (учитель)"},
		{Command: "createsubject", Description: "➕ Создать предмет (учитель)"},
	}

	_, err := c.bot.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: commands,
	})

	if err != nil {
		c.logger.Error("Failed to set bot commands", zap.Error(err))
		return err
	}

	c.logger.Info("✅ Bot commands menu set")
	return nil
}

// Start запускает бота
func (c *BotController) Start(ctx context.Context) error {
	c.logger.Info("Starting bot...")
	c.bot.Start(ctx)
	return nil
}
