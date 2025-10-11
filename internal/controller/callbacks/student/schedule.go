package student

import (
	"context"
	"fmt"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// HandleViewScheduleSubject показывает доступные слоты для предмета
func HandleViewScheduleSubject(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleViewScheduleSubject called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	subjectID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		h.Logger.Error("Failed to parse subject ID", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	h.Logger.Info("Parsed subject ID", zap.Int64("subject_id", subjectID))

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Получаем предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		h.Logger.Error("Subject not found",
			zap.Int64("subject_id", subjectID),
			zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	h.Logger.Info("Subject found",
		zap.Int64("subject_id", subjectID),
		zap.String("subject_name", subject.Name))

	// Получаем доступные слоты на следующие 14 дней
	now := time.Now()
	endDate := now.AddDate(0, 0, 14)

	h.Logger.Info("Fetching available slots",
		zap.Int64("subject_id", subjectID),
		zap.Time("from", now),
		zap.Time("to", endDate))

	slots, err := h.BookingService.GetAvailableSlots(ctx, subjectID, now, endDate)
	if err != nil {
		h.Logger.Error("Failed to get available slots",
			zap.Int64("subject_id", subjectID),
			zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось загрузить слоты")
		return
	}

	h.Logger.Info("Available slots retrieved",
		zap.Int64("subject_id", subjectID),
		zap.Int("count", len(slots)))

	if len(slots) == 0 {
		text := fmt.Sprintf("📅 Расписание: **%s**\n\n"+
			"К сожалению, сейчас нет доступных слотов на ближайшие 2 недели.\n\n"+
			"Попробуйте позже или выберите другой предмет.",
			subject.Name)

		keyboard := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: "⬅️ К списку предметов", CallbackData: "book_another"}},
			},
		}

		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      msg.Chat.ID,
			MessageID:   msg.ID,
			Text:        text,
			ParseMode:   models.ParseModeMarkdown,
			ReplyMarkup: keyboard,
		})
		common.AnswerCallback(ctx, b, callback.ID, "")
		return
	}

	// Группируем слоты по дням
	slotsByDate := make(map[string][]*model.ScheduleSlot)
	for _, slot := range slots {
		dateKey := slot.StartTime.Format("2006-01-02")
		slotsByDate[dateKey] = append(slotsByDate[dateKey], slot)
	}

	// Формируем текст и кнопки
	text := fmt.Sprintf("📅 **Расписание: %s**\n\n"+
		"💰 Цена: %.2f ₽\n"+
		"⏱ Длительность: %d мин\n\n"+
		"Доступные слоты на ближайшие 2 недели:\n\n",
		subject.Name,
		float64(subject.Price)/100,
		subject.Duration)

	var buttons [][]models.InlineKeyboardButton
	count := 0

	// Сортируем даты и выводим по 10 слотов максимум
	for _, slot := range slots {
		if count >= 10 {
			text += "\n💡 Показаны первые 10 слотов"
			break
		}

		dateStr := slot.StartTime.Format("02.01 (Mon)")
		timeStr := slot.StartTime.Format("15:04")

		buttonText := fmt.Sprintf("📅 %s • 🕐 %s", dateStr, timeStr)

		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: buttonText, CallbackData: fmt.Sprintf("book_lesson:%d", slot.ID)},
		})
		count++
	}

	// Добавляем кнопки навигации
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⬅️ К деталям предмета", CallbackData: fmt.Sprintf("view_subject:%d", subjectID)},
	})
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "📚 К списку предметов", CallbackData: "book_another"},
	})

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}
