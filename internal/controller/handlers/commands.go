package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/state"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// HandleStart обрабатывает команду /start
func (h *Handlers) HandleStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	user := update.Message.From

	// Регистрируем пользователя
	registeredUser, err := h.userService.RegisterUser(
		ctx,
		user.ID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.LanguageCode,
	)

	if err != nil {
		h.logger.Error("Failed to register user", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Произошла ошибка при регистрации. Попробуйте позже.",
		})
		return
	}

	welcomeText := fmt.Sprintf(
		"👋 Привет, %s!\n\n"+
			"Добро пожаловать в Scheduler Bot - бот для записи на занятия к учителям.\n\n"+
			"Доступные команды:\n"+
			"/subjects - Посмотреть все предметы\n"+
			"/mybookings - Мои записи\n"+
			"/help - Справка\n\n"+
			"Для учителей:\n"+
			"/becometeacher - Стать учителем\n"+
			"/mysubjects - Мои предметы\n"+
			"/myschedule - Моё расписание",
		registeredUser.FirstName,
	)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   welcomeText,
	})
}

// HandleHelp обрабатывает команду /help
func (h *Handlers) HandleHelp(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	helpText := "📚 Справка по командам:\n\n" +
		"Для студентов:\n" +
		"/start - Начать работу с ботом\n" +
		"/subjects - Список всех предметов\n" +
		"/mybookings - Мои записи на занятия\n" +
		"/help - Показать эту справку\n\n" +
		"Для учителей:\n" +
		"/becometeacher - Зарегистрироваться как учитель\n" +
		"/mysubjects - Управление своими предметами\n" +
		"/myschedule - Посмотреть расписание\n\n" +
		"Для записи на занятие выберите предмет из списка /subjects"

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   helpText,
	})
}

// HandleCancel обрабатывает команду /cancel - отмена текущего диалога
func (h *Handlers) HandleCancel(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	telegramID := update.Message.From.ID
	currentState := h.stateManager.GetState(telegramID)

	if currentState == state.StateNone {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Нет активных операций для отмены.",
		})
		return
	}

	// Очищаем состояние
	h.stateManager.ClearState(telegramID)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "✅ Операция отменена.\n\nИспользуйте /help для просмотра доступных команд.",
	})
}

// HandleTextMessage обрабатывает текстовые сообщения в зависимости от состояния пользователя
func (h *Handlers) HandleTextMessage(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}

	// Игнорируем команды (они обрабатываются другими handlers)
	if strings.HasPrefix(update.Message.Text, "/") {
		return
	}

	telegramID := update.Message.From.ID
	currentState := h.stateManager.GetState(telegramID)

	h.logger.Info("HandleTextMessage called",
		zap.Int64("telegram_id", telegramID),
		zap.String("text", update.Message.Text),
		zap.String("state", string(currentState)))

	// Если нет активного состояния, игнорируем
	if currentState == state.StateNone {
		h.logger.Debug("No active state, ignoring message",
			zap.Int64("telegram_id", telegramID))
		return
	}

	// Обрабатываем в зависимости от состояния
	switch currentState {
	case state.StateCreateSubjectName:
		h.logger.Info("Handling create subject name step",
			zap.Int64("telegram_id", telegramID))
		h.handleCreateSubjectNameStep(ctx, b, update)
	case state.StateCreateSubjectDescription:
		h.logger.Info("Handling create subject description step",
			zap.Int64("telegram_id", telegramID))
		h.handleCreateSubjectDescriptionStep(ctx, b, update)
	case state.StateCreateSubjectPrice:
		h.logger.Info("Handling create subject price step",
			zap.Int64("telegram_id", telegramID))
		h.handleCreateSubjectPriceStep(ctx, b, update)
	case state.StateCreateSubjectDuration:
		h.logger.Info("Handling create subject duration step",
			zap.Int64("telegram_id", telegramID))
		h.handleCreateSubjectDurationStep(ctx, b, update)
	case state.StateEditSubjectName:
		h.handleEditSubjectName(ctx, b, update)
	case state.StateEditSubjectDescription:
		h.handleEditSubjectDescription(ctx, b, update)
	case state.StateEditSubjectPrice:
		h.handleEditSubjectPrice(ctx, b, update)
	case state.StateEditSubjectDuration:
		h.handleEditSubjectDuration(ctx, b, update)
	default:
		h.logger.Warn("Unknown state", zap.String("state", string(currentState)))
	}
}
