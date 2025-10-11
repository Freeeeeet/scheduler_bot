package teacher

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// HandleEditFieldName устанавливает state для редактирования названия
func HandleEditFieldName(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleEditFieldName called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	subjectID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		h.Logger.Error("Failed to parse subject ID", zap.Error(err), zap.String("data", callback.Data))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	telegramID := callback.From.ID
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Устанавливаем state для редактирования названия
	h.StateManager.SetState(telegramID, "edit_subject_name")
	h.StateManager.SetData(telegramID, "subject_id", subjectID)

	h.Logger.Info("Set state for editing name",
		zap.Int64("telegram_id", telegramID),
		zap.Int64("subject_id", subjectID))

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
		Text:      "📝 Введите новое название предмета:\n\nДля отмены используйте /cancel",
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleEditFieldDesc устанавливает state для редактирования описания
func HandleEditFieldDesc(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	subjectID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	telegramID := callback.From.ID
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	h.StateManager.SetState(telegramID, "edit_subject_description")
	h.StateManager.SetData(telegramID, "subject_id", subjectID)

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
		Text:      "📄 Введите новое описание предмета:\n\nДля отмены используйте /cancel",
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleEditFieldPrice устанавливает state для редактирования цены
func HandleEditFieldPrice(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	subjectID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	telegramID := callback.From.ID
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	h.StateManager.SetState(telegramID, "edit_subject_price")
	h.StateManager.SetData(telegramID, "subject_id", subjectID)

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
		Text:      "💰 Введите новую цену в рублях (например: 1500):\n\nДля отмены используйте /cancel",
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleEditFieldDuration показывает кнопки для выбора длительности
func HandleEditFieldDuration(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	subjectID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "30 минут", CallbackData: fmt.Sprintf("set_duration:%d:30", subjectID)},
				{Text: "1 час (60 мин)", CallbackData: fmt.Sprintf("set_duration:%d:60", subjectID)},
			},
			{
				{Text: "1.5 часа (90 мин)", CallbackData: fmt.Sprintf("set_duration:%d:90", subjectID)},
				{Text: "2 часа (120 мин)", CallbackData: fmt.Sprintf("set_duration:%d:120", subjectID)},
			},
			{
				{Text: "2.5 часа (150 мин)", CallbackData: fmt.Sprintf("set_duration:%d:150", subjectID)},
				{Text: "3 часа (180 мин)", CallbackData: fmt.Sprintf("set_duration:%d:180", subjectID)},
			},
			{
				{Text: "✏️ Свой интервал", CallbackData: fmt.Sprintf("edit_duration_custom:%d", subjectID)},
			},
			{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("edit_subject:%d", subjectID)},
			},
		},
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        "⏱ Выберите новую длительность занятия:",
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleSetDuration устанавливает выбранную длительность
func HandleSetDuration(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	// Формат: set_duration:123:60
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 3 {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	subjectID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный ID")
		return
	}

	duration, err := strconv.Atoi(parts[2])
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверная длительность")
		return
	}

	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем текущий предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	// Обновляем только длительность
	subject.Duration = duration
	err = h.TeacherService.UpdateSubject(ctx, user.ID, subject)
	if err != nil {
		h.Logger.Error("Failed to update subject duration", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось обновить")
		return
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, fmt.Sprintf("✅ Длительность изменена на %d минут", duration))

	// Возвращаемся к меню редактирования
	HandleEditSubject(ctx, b, callback, h)
}

// HandleEditDurationCustom устанавливает state для ручного ввода длительности
func HandleEditDurationCustom(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	subjectID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	telegramID := callback.From.ID
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	h.StateManager.SetState(telegramID, "edit_subject_duration")
	h.StateManager.SetData(telegramID, "subject_id", subjectID)

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
		Text:      "⏱ Введите длительность в минутах (например: 45):\n\nДля отмены используйте /cancel",
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleToggleApproval переключает требование одобрения
func HandleToggleApproval(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	subjectID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем текущий предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	// Переключаем
	subject.RequiresBookingApproval = !subject.RequiresBookingApproval
	err = h.TeacherService.UpdateSubject(ctx, user.ID, subject)
	if err != nil {
		h.Logger.Error("Failed to toggle approval requirement", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось обновить")
		return
	}

	statusText := "включено"
	if !subject.RequiresBookingApproval {
		statusText = "выключено"
	}
	common.AnswerCallbackAlert(ctx, b, callback.ID, fmt.Sprintf("✅ Требование одобрения %s", statusText))

	// Обновляем меню
	HandleEditSubject(ctx, b, callback, h)
}
