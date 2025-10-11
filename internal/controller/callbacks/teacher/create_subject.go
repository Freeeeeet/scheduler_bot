package teacher

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/state"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// HandleCreateSubjectSetDuration обрабатывает выбор длительности кнопками
func HandleCreateSubjectSetDuration(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleCreateSubjectSetDuration called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	telegramID := callback.From.ID
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Парсим длительность из callback data
	// Формат: create_subject_set_duration:90 или create_subject_set_duration:custom
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 2 {
		h.Logger.Error("Invalid callback format", zap.String("data", callback.Data))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	durationStr := parts[1]

	// Если "custom" - просим ввести вручную
	if durationStr == "custom" {
		h.Logger.Info("User chose custom duration", zap.Int64("telegram_id", telegramID))

		// Остаемся в том же state
		h.StateManager.SetState(telegramID, callbacktypes.UserState(state.StateCreateSubjectDuration))

		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    msg.Chat.ID,
			MessageID: msg.ID,
			Text:      "✏️ Введите длительность занятия в минутах (например: 45, 75, 105):\n\nДля отмены используйте /cancel",
		})

		common.AnswerCallback(ctx, b, callback.ID, "")
		return
	}

	// Парсим выбранную длительность
	duration, err := strconv.Atoi(durationStr)
	if err != nil {
		h.Logger.Error("Failed to parse duration", zap.Error(err), zap.String("duration_str", durationStr))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверная длительность")
		return
	}

	h.Logger.Info("Duration selected via button",
		zap.Int64("telegram_id", telegramID),
		zap.Int("duration", duration))

	// Получаем все данные
	allData := h.StateManager.GetAllData(telegramID)
	name, _ := allData["name"].(string)
	description, _ := allData["description"].(string)
	price, _ := allData["price"].(int)

	// Сохраняем длительность и переходим к одобрению
	h.StateManager.SetData(telegramID, "duration", duration)
	h.StateManager.SetState(telegramID, callbacktypes.UserState(state.StateCreateSubjectApproval))

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "✅ Да, требуется одобрение", CallbackData: "create_subject_approval_yes"},
			},
			{
				{Text: "❌ Нет, записываться свободно", CallbackData: "create_subject_approval_no"},
			},
		},
	}

	priceInRubles := float64(price) / 100

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
		Text: fmt.Sprintf("✅ Название: %s\n"+
			"✅ Описание: %s\n"+
			"✅ Цена: %.2f ₽\n"+
			"✅ Длительность: %d минут\n\n"+
			"Шаг 5 из 5: Требуется ли ваше одобрение для записи?\n\n"+
			"• 🟢 Да - студенты отправляют запрос, вы одобряете\n"+
			"• 🔵 Нет - студенты записываются автоматически",
			name, description, priceInRubles, duration),
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}
