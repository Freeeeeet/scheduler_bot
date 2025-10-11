package handlers

import (
	"context"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// requireUser проверяет что пользователь существует
// Возвращает user и true если OK, nil и false если нет
func (h *Handlers) requireUser(ctx context.Context, b *bot.Bot, update *models.Update) (*model.User, bool) {
	if update.Message == nil {
		return nil, false
	}

	telegramID := update.Message.From.ID
	user, err := h.userService.GetByTelegramID(ctx, telegramID)

	if err != nil {
		h.logger.Error("Failed to get user", zap.Int64("telegram_id", telegramID), zap.Error(err))
		h.sendError(ctx, b, update.Message.Chat.ID, "❌ Произошла ошибка. Попробуйте позже.")
		return nil, false
	}

	if user == nil {
		h.sendError(ctx, b, update.Message.Chat.ID, "❌ Пользователь не найден. Используйте /start для регистрации.")
		return nil, false
	}

	return user, true
}

// requireTeacher проверяет что пользователь является учителем
func (h *Handlers) requireTeacher(ctx context.Context, b *bot.Bot, update *models.Update) (*model.User, bool) {
	user, ok := h.requireUser(ctx, b, update)
	if !ok {
		return nil, false
	}

	if !user.IsTeacher {
		h.sendError(ctx, b, update.Message.Chat.ID, "❌ Эта команда доступна только учителям.\n\nСтать учителем: /becometeacher")
		return nil, false
	}

	return user, true
}

// sendError отправляет сообщение об ошибке и логирует если не удалось
func (h *Handlers) sendError(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
	if err != nil {
		h.logger.Error("Failed to send error message",
			zap.Int64("chat_id", chatID),
			zap.String("text", text),
			zap.Error(err),
		)
	}
}

// sendMessage отправляет сообщение и логирует если не удалось
func (h *Handlers) sendMessage(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
	if err != nil {
		h.logger.Error("Failed to send message",
			zap.Int64("chat_id", chatID),
			zap.Error(err),
		)
	}
}
