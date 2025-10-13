package common

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Helper functions для всех callback handlers

// AnswerCallback отвечает на callback query (без alert)
func AnswerCallback(ctx context.Context, b *bot.Bot, callbackID string, text string) {
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
		ShowAlert:       false,
	})
}

// AnswerCallbackAlert отвечает на callback query с alert (всплывающее окно)
func AnswerCallbackAlert(ctx context.Context, b *bot.Bot, callbackID string, text string) {
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
		ShowAlert:       true,
	})
}

// GetMessageFromCallback извлекает сообщение из callback query
func GetMessageFromCallback(callback *models.CallbackQuery) *models.Message {
	if callback.Message.Message != nil {
		return callback.Message.Message
	}
	return nil
}

// ParseIDFromCallback извлекает ID из callback data
// Например: "edit_subject:123" -> 123
func ParseIDFromCallback(data string) (int64, error) {
	parts := strings.Split(data, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid callback data format")
	}
	return strconv.ParseInt(parts[1], 10, 64)
}

// ParseMultiIDFromCallback извлекает несколько ID из callback data
// Например: "request_recurring_confirm:123:456" с префиксом "request_recurring_confirm:" -> [123, 456]
func ParseMultiIDFromCallback(callbackData string, prefix string) []int64 {
	// Убираем префикс
	data := strings.TrimPrefix(callbackData, prefix)
	parts := strings.Split(data, ":")

	var ids []int64
	for _, part := range parts {
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}

	return ids
}

// IsMessageNotModifiedError проверяет является ли ошибка "message is not modified"
// Это не настоящая ошибка - просто сообщение уже имеет нужное содержимое
func IsMessageNotModifiedError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "message is not modified")
}
