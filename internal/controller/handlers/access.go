package handlers

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// handleEnteringInviteCode обрабатывает ввод invite кода
// Примечание: для полной функциональности требуется AccessService
// В текущей реализации просто сообщаем пользователю использовать inline-кнопки
func (h *Handlers) handleEnteringInviteCode(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.Message.From.ID
	code := update.Message.Text

	h.logger.Info("User entered invite code",
		zap.Int64("telegram_id", telegramID),
		zap.String("code", code))

	// Очищаем состояние
	h.stateManager.ClearState(telegramID)

	// Отправляем инструкцию
	text := "🎟️ *Ввод кода приглашения*\n\n" +
		"Для ввода кода приглашения, пожалуйста:\n\n" +
		"1. Используйте команду /subjects\n" +
		"2. Выберите '🔍 Найти учителя'\n" +
		"3. Нажмите '🎟️ У меня есть код'\n" +
		"4. Введите код в появившемся поле\n\n" +
		"_Примечание: Обработка кодов происходит через специальный интерфейс бота._"

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	})
}

