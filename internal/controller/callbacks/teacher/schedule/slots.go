package schedule

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common/formatting"
	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// ========================
// Slot Management Handlers
// ========================

// HandleViewSlotDetails показывает детальную информацию о слоте с управлением
func HandleViewSlotDetails(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleViewSlotDetails called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: view_slot_details:slot_id:weekOffset
	parts := strings.Split(callback.Data, ":")
	if len(parts) < 2 {
		h.Logger.Error("Invalid callback format", zap.String("data", callback.Data))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	slotID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		h.Logger.Error("Failed to parse slot ID", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный ID слота")
		return
	}

	weekOffset := 0
	if len(parts) >= 3 {
		weekOffset, err = strconv.Atoi(parts[2])
		if err != nil {
			h.Logger.Warn("Failed to parse week offset, using 0", zap.Error(err))
			weekOffset = 0
		}
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

	// Получаем слот
	slot, err := h.TeacherService.GetSlotByID(ctx, slotID)
	if err != nil || slot == nil {
		h.Logger.Error("Slot not found", zap.Int64("slot_id", slotID), zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Слот не найден")
		return
	}

	// Проверяем что пользователь - владелец слота
	if slot.TeacherID != user.ID {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ У вас нет доступа к этому слоту")
		return
	}

	// Получаем предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, slot.SubjectID)
	if err != nil || subject == nil {
		h.Logger.Error("Subject not found", zap.Int64("subject_id", slot.SubjectID), zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	// Формируем текст с деталями
	statusEmoji := "🟢"
	statusText := "Свободен"
	switch slot.Status {
	case model.SlotStatusBooked:
		statusEmoji = "🔴"
		statusText = "Забронирован"
	case model.SlotStatusCanceled:
		statusEmoji = "⚫️"
		statusText = "Отменён"
	}

	duration := slot.EndTime.Sub(slot.StartTime).Minutes()

	text := fmt.Sprintf("📋 <b>Детали слота</b>\n\n"+
		"📚 <b>Предмет:</b> %s\n"+
		"📅 <b>Дата:</b> %s, %s\n"+
		"🕐 <b>Время:</b> %s - %s\n"+
		"⏱ <b>Длительность:</b> %.0f мин\n"+
		"%s <b>Статус:</b> %s\n",
		subject.Name,
		slot.StartTime.Format("02.01.2006"),
		formatting.GetWeekdayName(int(slot.StartTime.Weekday())),
		slot.StartTime.Format("15:04"),
		slot.EndTime.Format("15:04"),
		duration,
		statusEmoji,
		statusText)

	var buttons [][]models.InlineKeyboardButton

	// Если слот забронирован, показываем информацию о студенте
	if slot.Status == model.SlotStatusBooked && slot.StudentID != nil {
		student, err := h.UserService.GetByID(ctx, *slot.StudentID)
		if err == nil && student != nil {
			fullName := student.FirstName
			if student.LastName != "" {
				fullName += " " + student.LastName
			}
			text += fmt.Sprintf("\n👤 <b>Студент:</b> %s\n", fullName)
			if student.Username != "" {
				text += fmt.Sprintf("📱 <b>Контакт:</b> @%s\n", student.Username)
			}
		}

		// Кнопки для забронированного слота
		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: "❌ Отменить запись студента", CallbackData: fmt.Sprintf("cancel_booking_from_slot:%d:%d", slotID, weekOffset)},
		})
	} else if slot.Status == model.SlotStatusFree {
		// Кнопки для свободного слота
		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: "🗑 Отменить слот", CallbackData: fmt.Sprintf("cancel_slot:%d:%d", slotID, weekOffset)},
		})
	} else if slot.Status == model.SlotStatusCanceled {
		// Кнопки для отменённого слота
		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: "♻️ Восстановить слот", CallbackData: fmt.Sprintf("restore_slot:%d:%d", slotID, weekOffset)},
		})
	}

	// Кнопка "Назад"
	dateStr := slot.StartTime.Format("2006-01-02")
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⬅️ Назад к дню", CallbackData: fmt.Sprintf("view_schedule_week_day:%d:%s", weekOffset, dateStr)},
	})

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
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

// HandleCancelSlot отменяет свободный слот
func HandleCancelSlot(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleCancelSlot called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: cancel_slot:slot_id:weekOffset
	parts := strings.Split(callback.Data, ":")
	if len(parts) < 2 {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	slotID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный ID слота")
		return
	}

	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем слот
	slot, err := h.TeacherService.GetSlotByID(ctx, slotID)
	if err != nil || slot == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Слот не найден")
		return
	}

	// Проверяем владельца
	if slot.TeacherID != user.ID {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ У вас нет доступа к этому слоту")
		return
	}

	// Проверяем что слот свободен
	if slot.Status != model.SlotStatusFree {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Можно отменить только свободный слот")
		return
	}

	// Отменяем слот
	err = h.TeacherService.CancelSlot(ctx, slotID)
	if err != nil {
		h.Logger.Error("Failed to cancel slot", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось отменить слот")
		return
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Слот отменён")

	// Обновляем экран с деталями
	HandleViewSlotDetails(ctx, b, callback, h)
}

// HandleRestoreSlot восстанавливает отменённый слот
func HandleRestoreSlot(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleRestoreSlot called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: restore_slot:slot_id:weekOffset
	parts := strings.Split(callback.Data, ":")
	if len(parts) < 2 {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	slotID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный ID слота")
		return
	}

	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем слот
	slot, err := h.TeacherService.GetSlotByID(ctx, slotID)
	if err != nil || slot == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Слот не найден")
		return
	}

	// Проверяем владельца
	if slot.TeacherID != user.ID {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ У вас нет доступа к этому слоту")
		return
	}

	// Проверяем что слот отменён
	if slot.Status != model.SlotStatusCanceled {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Можно восстановить только отменённый слот")
		return
	}

	// Восстанавливаем слот
	err = h.TeacherService.RestoreSlot(ctx, slotID)
	if err != nil {
		h.Logger.Error("Failed to restore slot", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось восстановить слот")
		return
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Слот восстановлен")

	// Обновляем экран с деталями
	HandleViewSlotDetails(ctx, b, callback, h)
}

// HandleCancelBookingFromSlot отменяет запись студента на слот
func HandleCancelBookingFromSlot(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleCancelBookingFromSlot called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: cancel_booking_from_slot:slot_id:weekOffset
	parts := strings.Split(callback.Data, ":")
	if len(parts) < 2 {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	slotID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный ID слота")
		return
	}

	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем слот
	slot, err := h.TeacherService.GetSlotByID(ctx, slotID)
	if err != nil || slot == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Слот не найден")
		return
	}

	// Проверяем владельца
	if slot.TeacherID != user.ID {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ У вас нет доступа к этому слоту")
		return
	}

	// Проверяем что слот забронирован
	if slot.Status != model.SlotStatusBooked {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Слот не забронирован")
		return
	}

	// Отменяем бронирование (освобождаем слот)
	err = h.TeacherService.CancelBookingBySlot(ctx, slotID, user.ID)
	if err != nil {
		h.Logger.Error("Failed to cancel booking", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось отменить запись")
		return
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Запись студента отменена")

	// Обновляем экран с деталями
	HandleViewSlotDetails(ctx, b, callback, h)
}

// HandleAddSlots начинает процесс добавления слотов
func HandleAddSlots(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	text := "🕐 Добавление временных слотов\n\n" +
		"Для добавления слотов используйте команду:\n" +
		"/addslots\n\n" +
		"Или создайте слоты через API."

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
		Text:      text,
	})

	common.AnswerCallback(ctx, b, callback.ID, "Добавление слотов")
}
