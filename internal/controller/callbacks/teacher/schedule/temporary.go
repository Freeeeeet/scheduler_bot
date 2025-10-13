package schedule

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

// ========================
// Temporary Schedule Management Handlers
// ========================

// HandleManageTemporary показывает управление временными расписаниями
func HandleManageTemporary(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleManageTemporary called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: manage_temporary:subject_id
	subjectID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		h.Logger.Error("Failed to parse subject ID", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		h.Logger.Error("Subject not found", zap.Int64("subject_id", subjectID), zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	// Получаем все слоты предмета на следующие 30 дней
	now := time.Now()
	endDate := now.AddDate(0, 0, 30)
	allSlots, err := h.TeacherService.GetTeacherSchedule(ctx, user.ID, now, endDate)
	if err != nil {
		h.Logger.Error("Failed to get schedule", zap.Error(err))
		allSlots = []*model.ScheduleSlot{}
	}

	// Фильтруем только слоты этого предмета
	var subjectSlots []*model.ScheduleSlot
	for _, slot := range allSlots {
		if slot.SubjectID == subjectID {
			subjectSlots = append(subjectSlots, slot)
		}
	}

	text := fmt.Sprintf("📅 <b>Временные расписания</b>\n\n<b>Предмет:</b> %s\n\n", subject.Name)

	if len(subjectSlots) == 0 {
		text += "У вас пока нет временных слотов для этого предмета.\n\n"
		text += "Временные слоты — это разовые занятия, созданные вручную."
	} else {
		// Подсчитываем статистику
		totalSlots := len(subjectSlots)
		bookedCount := 0
		freeCount := 0
		canceledCount := 0

		for _, slot := range subjectSlots {
			switch slot.Status {
			case model.SlotStatusBooked:
				bookedCount++
			case model.SlotStatusFree:
				freeCount++
			case model.SlotStatusCanceled:
				canceledCount++
			}
		}

		text += fmt.Sprintf("📊 <b>Всего слотов (30 дней):</b> %d\n"+
			"🟢 Свободно: %d\n"+
			"🔴 Забронировано: %d\n"+
			"⚫️ Отменено: %d\n\n",
			totalSlots, freeCount, bookedCount, canceledCount)

		text += "<b>Ближайшие 5 слотов:</b>\n"
		for i, slot := range subjectSlots {
			if i >= 5 {
				text += fmt.Sprintf("... и еще %d слотов\n", len(subjectSlots)-5)
				break
			}
			statusEmoji := "🟢"
			switch slot.Status {
			case model.SlotStatusBooked:
				statusEmoji = "🔴"
			case model.SlotStatusCanceled:
				statusEmoji = "⚫️"
			}
			text += fmt.Sprintf("%s %s в %s\n",
				statusEmoji,
				slot.StartTime.Format("02.01 (Mon)"),
				slot.StartTime.Format("15:04"))
		}
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "➕ Создать временный слот", CallbackData: fmt.Sprintf("create_slots:%d", subjectID)},
			},
			{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("subject_schedule:%d", subjectID)},
			},
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
