package common

import (
	"context"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// HandlerContext содержит общие данные для обработки callback
// Это избавляет от дублирования кода получения пользователя, сообщения и т.д.
type HandlerContext struct {
	Ctx        context.Context
	Bot        *bot.Bot
	Callback   *models.CallbackQuery
	Handler    *callbacktypes.Handler
	Message    *models.Message
	User       *model.User
	TelegramID int64
	ChatID     int64
}

// NewHandlerContext создаёт новый контекст обработчика
func NewHandlerContext(
	ctx context.Context,
	b *bot.Bot,
	callback *models.CallbackQuery,
	h *callbacktypes.Handler,
) *HandlerContext {
	msg := GetMessageFromCallback(callback)
	var chatID int64
	if msg != nil {
		chatID = msg.Chat.ID
	}

	return &HandlerContext{
		Ctx:        ctx,
		Bot:        b,
		Callback:   callback,
		Handler:    h,
		Message:    msg,
		TelegramID: callback.From.ID,
		ChatID:     chatID,
	}
}

// LoadUser загружает пользователя в контекст
func (hc *HandlerContext) LoadUser() error {
	user, err := hc.Handler.UserService.GetByTelegramID(hc.Ctx, hc.TelegramID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	hc.User = user
	return nil
}

// RequireUser проверяет что пользователь загружен
func (hc *HandlerContext) RequireUser() error {
	if hc.User == nil {
		return hc.LoadUser()
	}
	return nil
}

// RequireTeacher проверяет что пользователь является учителем
func (hc *HandlerContext) RequireTeacher() error {
	if err := hc.RequireUser(); err != nil {
		return err
	}
	if !hc.User.IsTeacher {
		return ErrNotATeacher
	}
	return nil
}

// RequireSubjectOwner проверяет что пользователь владеет предметом
func (hc *HandlerContext) RequireSubjectOwner(subjectID int64) (*model.Subject, error) {
	if err := hc.RequireTeacher(); err != nil {
		return nil, err
	}

	subject, err := hc.Handler.TeacherService.GetSubjectByID(hc.Ctx, subjectID)
	if err != nil {
		return nil, err
	}
	if subject == nil {
		return nil, ErrSubjectNotFound
	}
	if subject.TeacherID != hc.User.ID {
		return nil, ErrNotSubjectOwner
	}

	return subject, nil
}

// Answer отвечает на callback query
func (hc *HandlerContext) Answer(text string) {
	AnswerCallback(hc.Ctx, hc.Bot, hc.Callback.ID, text)
}

// AnswerAlert отвечает на callback query с alert
func (hc *HandlerContext) AnswerAlert(text string) {
	AnswerCallbackAlert(hc.Ctx, hc.Bot, hc.Callback.ID, text)
}

// EditMessage редактирует сообщение
func (hc *HandlerContext) EditMessage(text string, keyboard *models.InlineKeyboardMarkup) error {
	if hc.Message == nil {
		return ErrNoMessage
	}

	_, err := hc.Bot.EditMessageText(hc.Ctx, &bot.EditMessageTextParams{
		ChatID:      hc.ChatID,
		MessageID:   hc.Message.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	// Игнорируем ошибку "message is not modified" - это не настоящая ошибка
	if IsMessageNotModifiedError(err) {
		return nil
	}

	return err
}

// EditMessageText редактирует только текст сообщения
func (hc *HandlerContext) EditMessageText(text string) error {
	return hc.EditMessage(text, nil)
}

// DeleteMessage удаляет сообщение
func (hc *HandlerContext) DeleteMessage() error {
	if hc.Message == nil {
		return ErrNoMessage
	}

	_, err := hc.Bot.DeleteMessage(hc.Ctx, &bot.DeleteMessageParams{
		ChatID:    hc.ChatID,
		MessageID: hc.Message.ID,
	})

	return err
}

// SendMessage отправляет новое сообщение
func (hc *HandlerContext) SendMessage(text string, keyboard *models.InlineKeyboardMarkup) error {
	_, err := hc.Bot.SendMessage(hc.Ctx, &bot.SendMessageParams{
		ChatID:      hc.ChatID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	return err
}

// ClearState очищает состояние пользователя
func (hc *HandlerContext) ClearState() {
	hc.Handler.StateManager.ClearState(hc.TelegramID)
}

// SetState устанавливает состояние пользователя
func (hc *HandlerContext) SetState(state callbacktypes.UserState) {
	hc.Handler.StateManager.SetState(hc.TelegramID, state)
}

// SetData устанавливает данные в state
func (hc *HandlerContext) SetData(key string, value interface{}) {
	hc.Handler.StateManager.SetData(hc.TelegramID, key, value)
}

// GetData получает данные из state
func (hc *HandlerContext) GetData(key string) (interface{}, bool) {
	return hc.Handler.StateManager.GetData(hc.TelegramID, key)
}
