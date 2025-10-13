package common

import (
	"context"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// WithUser создаёт HandlerContext и загружает пользователя
// При ошибке автоматически отвечает пользователю и возвращает nil
func WithUser(
	ctx context.Context,
	b *bot.Bot,
	callback *models.CallbackQuery,
	h *callbacktypes.Handler,
	handler func(*HandlerContext),
) {
	hc := NewHandlerContext(ctx, b, callback, h)

	if err := hc.LoadUser(); err != nil {
		h.Logger.Error("Failed to load user",
			zap.Int64("telegram_id", hc.TelegramID),
			zap.Error(err))
		hc.AnswerAlert(ErrorMessage(err))
		return
	}

	handler(hc)
}

// WithTeacher создаёт HandlerContext и проверяет что пользователь - учитель
// При ошибке автоматически отвечает пользователю и возвращает nil
func WithTeacher(
	ctx context.Context,
	b *bot.Bot,
	callback *models.CallbackQuery,
	h *callbacktypes.Handler,
	handler func(*HandlerContext),
) {
	hc := NewHandlerContext(ctx, b, callback, h)

	if err := hc.RequireTeacher(); err != nil {
		h.Logger.Error("Teacher check failed",
			zap.Int64("telegram_id", hc.TelegramID),
			zap.Error(err))
		hc.AnswerAlert(ErrorMessage(err))
		return
	}

	handler(hc)
}

// WithSubjectOwner создаёт HandlerContext, проверяет что пользователь - владелец предмета
// При ошибке автоматически отвечает пользователю и возвращает nil
// При успехе передаёт HandlerContext и Subject в handler
func WithSubjectOwner(
	ctx context.Context,
	b *bot.Bot,
	callback *models.CallbackQuery,
	h *callbacktypes.Handler,
	subjectID int64,
	handler func(*HandlerContext, *model.Subject),
) {
	hc := NewHandlerContext(ctx, b, callback, h)

	subject, err := hc.RequireSubjectOwner(subjectID)
	if err != nil {
		h.Logger.Error("Subject owner check failed",
			zap.Int64("telegram_id", hc.TelegramID),
			zap.Int64("subject_id", subjectID),
			zap.Error(err))
		hc.AnswerAlert(ErrorMessage(err))
		return
	}

	handler(hc, subject)
}

// HandleError обрабатывает ошибку и отправляет ответ пользователю
func HandleError(hc *HandlerContext, err error, operation string) {
	hc.Handler.Logger.Error("Operation failed",
		zap.String("operation", operation),
		zap.Int64("telegram_id", hc.TelegramID),
		zap.Error(err))
	hc.AnswerAlert(ErrorMessage(err))
}

// LogAndAnswer логирует действие и отвечает на callback
func LogAndAnswer(hc *HandlerContext, message string, answer string) {
	hc.Handler.Logger.Info(message,
		zap.Int64("telegram_id", hc.TelegramID),
		zap.Int64("user_id", hc.User.ID))
	hc.Answer(answer)
}
