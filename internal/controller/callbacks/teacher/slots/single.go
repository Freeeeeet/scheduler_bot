package slots

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// HandleSingleTimeAuto создает единичный слот с автоматически выбранным временем
func HandleSingleTimeAuto(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleSingleTimeAuto called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: single_time_auto:123:2024-01-15:15:30
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 5 {
		h.Logger.Error("Invalid callback format", zap.String("data", callback.Data))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	subjectID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		h.Logger.Error("Failed to parse subject ID", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный ID")
		return
	}

	dateStr := parts[2]
	timeStr := parts[3] + ":" + parts[4]

	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	// Парсим дату и время
	dateTimeStr := fmt.Sprintf("%s %s", dateStr, timeStr)
	startTime, err := time.Parse("2006-01-02 15:04", dateTimeStr)
	if err != nil {
		h.Logger.Error("Failed to parse datetime", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверная дата/время")
		return
	}

	endTime := startTime.Add(time.Duration(subject.Duration) * time.Minute)

	// Создаем слот
	slot, err := h.TeacherService.CreateSlot(ctx, user.ID, subjectID, startTime, endTime)
	if err != nil {
		h.Logger.Error("Failed to create slot", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось создать слот")
		return
	}

	h.Logger.Info("Slot created successfully",
		zap.Int64("slot_id", slot.ID),
		zap.Time("start_time", startTime))

	msg := common.GetMessageFromCallback(callback)
	if msg != nil {
		text := fmt.Sprintf("✅ <b>Слот создан!</b>\n\n"+
			"📚 Предмет: %s\n"+
			"📅 Дата: %s\n"+
			"🕐 Время: %s - %s\n"+
			"⏱ Длительность: %d мин\n\n"+
			"Посмотреть расписание: /myschedule",
			subject.Name,
			startTime.Format("02.01.2006 (Monday)"),
			startTime.Format("15:04"),
			endTime.Format("15:04"),
			subject.Duration)

		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    msg.Chat.ID,
			MessageID: msg.ID,
			Text:      text,
			ParseMode: models.ParseModeHTML,
		})
	}

	common.AnswerCallback(ctx, b, callback.ID, "✅ Слот создан!")
}

// HandleCustomTime начинает процесс ввода пользовательского времени
func HandleCustomTime(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleCustomTime called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: custom_time:123:2024-01-15
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 3 {
		h.Logger.Error("Invalid callback format", zap.String("data", callback.Data))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	subjectID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		h.Logger.Error("Failed to parse subject ID", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный ID")
		return
	}

	dateStr := parts[2]

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	telegramID := callback.From.ID

	// Устанавливаем состояние для ожидания ввода времени
	h.StateManager.SetState(telegramID, "custom_slot_time")
	h.StateManager.SetData(telegramID, "subject_id", subjectID)
	h.StateManager.SetData(telegramID, "date_str", dateStr)

	text := "⌨️ <b>Ввод времени вручную</b>\n\n" +
		"Введите время начала занятия в формате <b>ЧЧ:ММ</b>\n\n" +
		"Примеры:\n" +
		"• 09:30\n" +
		"• 14:45\n" +
		"• 18:00\n\n" +
		"Отправьте /cancel для отмены."

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("single_day_date:%d:%s", subjectID, dateStr)}},
		},
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleCustomTimeInput обрабатывает введенное время (вызывается из обработчика текстовых сообщений)
func HandleCustomTimeInput(ctx context.Context, b *bot.Bot, update *models.Update, h *callbacktypes.Handler, timeText string, subjectID int64, dateStr string) {
	// Проверяем формат времени (ЧЧ:ММ)
	timeRegex := regexp.MustCompile(`^([0-1][0-9]|2[0-3]):([0-5][0-9])$`)
	if !timeRegex.MatchString(timeText) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text: "❌ Неверный формат времени!\n\n" +
				"Используйте формат <b>ЧЧ:ММ</b> (например, 09:30 или 14:45)\n\n" +
				"Попробуйте еще раз или отправьте /cancel для отмены.",
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	user, err := h.UserService.GetByTelegramID(ctx, update.Message.From.ID)
	if err != nil || user == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Пользователь не найден",
		})
		return
	}

	// Получаем предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Предмет не найден",
		})
		return
	}

	// Парсим дату и время
	dateTimeStr := fmt.Sprintf("%s %s", dateStr, timeText)
	startTime, err := time.Parse("2006-01-02 15:04", dateTimeStr)
	if err != nil {
		h.Logger.Error("Failed to parse datetime", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Не удалось обработать дату/время",
		})
		return
	}

	endTime := startTime.Add(time.Duration(subject.Duration) * time.Minute)

	// Создаем слот
	slot, err := h.TeacherService.CreateSlot(ctx, user.ID, subjectID, startTime, endTime)
	if err != nil {
		h.Logger.Error("Failed to create slot", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("❌ Не удалось создать слот: %v", err),
		})
		return
	}

	h.Logger.Info("Slot created successfully via custom time",
		zap.Int64("slot_id", slot.ID),
		zap.Time("start_time", startTime))

	// Очищаем состояние
	h.StateManager.ClearState(update.Message.From.ID)

	text := fmt.Sprintf("✅ <b>Слот создан!</b>\n\n"+
		"📚 Предмет: %s\n"+
		"📅 Дата: %s\n"+
		"🕐 Время: %s - %s\n"+
		"⏱ Длительность: %d мин\n\n"+
		"Посмотреть расписание: /myschedule",
		subject.Name,
		startTime.Format("02.01.2006 (Monday)"),
		startTime.Format("15:04"),
		endTime.Format("15:04"),
		subject.Duration)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
}
