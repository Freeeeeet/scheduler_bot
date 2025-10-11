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
