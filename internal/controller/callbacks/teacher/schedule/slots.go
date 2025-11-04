package schedule

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common/formatting"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/state"
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

// HandleSlotAction показывает экран выбора действия для слота
func HandleSlotAction(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleSlotAction called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: slot_action:slotID:subjectID:date
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
	if err != nil || user == nil || !user.IsTeacher {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Доступ запрещен")
		return
	}

	// Получаем информацию о слоте
	slot, err := h.TeacherService.GetSlotByID(ctx, slotID)
	if err != nil || slot == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Слот не найден")
		return
	}

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Формируем текст и кнопки для выбора действия
	timeStr := fmt.Sprintf("%s - %s", slot.StartTime.Format("15:04"), slot.EndTime.Format("15:04"))
	text := fmt.Sprintf("🕐 <b>Слот: %s</b>\n\n"+
		"Выберите действие:\n\n"+
		"📌 Пометить занятым - отметить слот как занятый\n"+
		"👤 Закрепить за учеником - назначить слот конкретному ученику",
		timeStr)

	var buttons [][]models.InlineKeyboardButton

	// Кнопка "Пометить занятым без комментария"
	callbackData := fmt.Sprintf("mark_busy_simple:%d", slotID)
	if len(parts) >= 4 {
		callbackData += fmt.Sprintf(":%s:%s", parts[2], parts[3])
	}
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "📌 Пометить занятым", CallbackData: callbackData},
	})

	// Кнопка "Пометить занятым с комментарием"
	callbackDataWithComment := fmt.Sprintf("mark_busy_comment:%d", slotID)
	if len(parts) >= 4 {
		callbackDataWithComment += fmt.Sprintf(":%s:%s", parts[2], parts[3])
	}
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "📝 Пометить занятым с комментарием", CallbackData: callbackDataWithComment},
	})

	// Кнопка "Закрепить за учеником"
	callbackDataAssign := fmt.Sprintf("assign_slot_student:%d", slotID)
	if len(parts) >= 4 {
		callbackDataAssign += fmt.Sprintf(":%s:%s", parts[2], parts[3])
	}
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "👤 Закрепить за учеником", CallbackData: callbackDataAssign},
	})

	// Кнопка "Назад"
	if len(parts) >= 4 {
		subjectID := parts[2]
		dateStr := parts[3]
		weekdayName := formatting.GetWeekdayName(int(slot.StartTime.Weekday()))
		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("view_schedule_day:%s:%s:%s", subjectID, dateStr, weekdayName)},
		})
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	// Если сообщение содержит фото, отправляем новое сообщение вместо редактирования
	if len(msg.Photo) > 0 {
		// Удаляем старое сообщение с фото
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    msg.Chat.ID,
			MessageID: msg.ID,
		})
		// Отправляем новое текстовое сообщение
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      msg.Chat.ID,
			Text:        text,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: keyboard,
		})
	} else {
		// Обычное текстовое сообщение - редактируем
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      msg.Chat.ID,
			MessageID:   msg.ID,
			Text:        text,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: keyboard,
		})
	}

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleMarkBusySimple помечает слот как занятый без комментария
func HandleMarkBusySimple(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleMarkBusySimple called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: mark_busy_simple:slotID:subjectID:date
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

	// Помечаем слот как занятый без комментария
	err = h.TeacherService.MarkSlotBusy(ctx, slotID, user.ID)
	if err != nil {
		h.Logger.Error("Failed to mark slot busy", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось пометить слот как занятый")
		return
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Слот помечен как занятый")

	// Возвращаемся к экрану дня
	if len(parts) >= 4 {
		dateStr := parts[3]
		slot, err := h.TeacherService.GetSlotByID(ctx, slotID)
		if err == nil && slot != nil {
			callback.Data = fmt.Sprintf("view_schedule_day:%d:%s:%s", slot.SubjectID, dateStr, formatting.GetWeekdayName(int(slot.StartTime.Weekday())))
			HandleViewScheduleDay(ctx, b, callback, h)
		}
	}
}

// HandleMarkBusyComment инициирует ввод комментария для пометки слота занятым
func HandleMarkBusyComment(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleMarkBusyComment called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: mark_busy_comment:slotID:subjectID:date
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

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Сохраняем slotID в state для обработки комментария
	h.StateManager.SetState(telegramID, callbacktypes.UserState(state.StateMarkSlotBusyComment))
	h.StateManager.SetData(telegramID, "slot_id", slotID)
	if len(parts) >= 4 {
		h.StateManager.SetData(telegramID, "subject_id", parts[2])
		h.StateManager.SetData(telegramID, "date", parts[3])
	}

	text := "📝 <b>Введите комментарий для слота</b>\n\n" +
		"Например: Встреча, Выезд, Личные дела\n\n" +
		"Можно оставить пустым, нажав /skip"

	// Формируем callback для отмены с полными параметрами
	cancelCallback := fmt.Sprintf("slot_action:%d", slotID)
	if len(parts) >= 4 {
		cancelCallback = fmt.Sprintf("slot_action:%d:%s:%s", slotID, parts[2], parts[3])
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "❌ Отмена", CallbackData: cancelCallback}},
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

// HandleMarkSlotBusy помечает слот как занятый без привязки к студенту
func HandleMarkSlotBusy(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleMarkSlotBusy called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: mark_slot_busy:slotID:subjectID:date
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

	// Помечаем слот как занятый
	err = h.TeacherService.MarkSlotBusy(ctx, slotID, user.ID)
	if err != nil {
		h.Logger.Error("Failed to mark slot busy", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось пометить слот как занятый")
		return
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Слот помечен как занятый")

	// Возвращаемся к экрану дня (функция находится в calendar.go, но в том же пакете)
	if len(parts) >= 4 {
		dateStr := parts[3]

		// Получаем слот для получения даты и предмета
		slot, err := h.TeacherService.GetSlotByID(ctx, slotID)
		if err == nil && slot != nil {
			// Формируем callback для возврата к экрану дня
			callback.Data = fmt.Sprintf("view_schedule_day:%d:%s:%s", slot.SubjectID, dateStr, formatting.GetWeekdayName(int(slot.StartTime.Weekday())))
			// Вызываем функцию из calendar.go (они в одном пакете schedule)
			HandleViewScheduleDay(ctx, b, callback, h)
		}
	}
}

// HandleAssignSlotStudent показывает список студентов для закрепления слота
func HandleAssignSlotStudent(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleAssignSlotStudent called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: assign_slot_student:slotID:subjectID:date
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
	if err != nil || user == nil || !user.IsTeacher {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Доступ запрещен")
		return
	}

	// Получаем список студентов преподавателя
	students, err := h.AccessService.GetMyStudents(ctx, user.ID)
	if err != nil {
		h.Logger.Error("Failed to get students", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка при загрузке студентов")
		return
	}

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	if len(students) == 0 {
		text := "👥 У вас пока нет студентов.\n\n" +
			"Студенты могут получить доступ к вашим предметам через:\n" +
			"• Публичный профиль (если включен)\n" +
			"• Код приглашения\n" +
			"• Заявку на доступ"

		var buttons [][]models.InlineKeyboardButton
		// Добавляем кнопку "Назад" если есть информация о предмете и дате
		if len(parts) >= 4 {
			subjectID := parts[2]
			dateStr := parts[3]
			// Получаем слот для получения weekday
			slot, err := h.TeacherService.GetSlotByID(ctx, slotID)
			if err == nil && slot != nil {
				weekdayName := formatting.GetWeekdayName(int(slot.StartTime.Weekday()))
				buttons = append(buttons, []models.InlineKeyboardButton{
					{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("view_schedule_day:%s:%s:%s", subjectID, dateStr, weekdayName)},
				})
			} else {
				// Fallback если не удалось получить слот
				buttons = append(buttons, []models.InlineKeyboardButton{
					{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("view_schedule_day:%s:%s:%s", subjectID, dateStr, dateStr)},
				})
			}
		}

		keyboard := &models.InlineKeyboardMarkup{
			InlineKeyboard: buttons,
		}

		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      msg.Chat.ID,
			MessageID:   msg.ID,
			Text:        text,
			ReplyMarkup: keyboard,
		})
		common.AnswerCallback(ctx, b, callback.ID, "")
		return
	}

	text := fmt.Sprintf("👥 <b>Выберите студента для закрепления слота</b>\n\nВсего студентов: %d\n\n", len(students))

	var buttons [][]models.InlineKeyboardButton
	for i, student := range students {
		if i >= 10 {
			break
		}
		studentName := student.FirstName
		if student.LastName != "" {
			studentName += " " + student.LastName
		}
		if len(studentName) > 25 {
			studentName = studentName[:25] + "..."
		}

		// Формат: assign_slot_to:slotID:studentID:subjectID:date
		callbackData := fmt.Sprintf("assign_slot_to:%d:%d", slotID, student.ID)
		if len(parts) >= 4 {
			callbackData += fmt.Sprintf(":%s:%s", parts[2], parts[3])
		}

		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: fmt.Sprintf("👤 %s", studentName), CallbackData: callbackData},
		})
	}

	// Кнопка назад
	if len(parts) >= 4 {
		subjectID := parts[2]
		dateStr := parts[3]
		// Получаем слот для получения weekday
		slot, err := h.TeacherService.GetSlotByID(ctx, slotID)
		if err == nil && slot != nil {
			weekdayName := formatting.GetWeekdayName(int(slot.StartTime.Weekday()))
			buttons = append(buttons, []models.InlineKeyboardButton{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("view_schedule_day:%s:%s:%s", subjectID, dateStr, weekdayName)},
			})
		} else {
			// Fallback если не удалось получить слот
			buttons = append(buttons, []models.InlineKeyboardButton{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("view_schedule_day:%s:%s:%s", subjectID, dateStr, dateStr)},
			})
		}
	}

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

// HandleAssignSlotTo закрепляет слот за конкретным студентом
func HandleAssignSlotTo(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleAssignSlotTo called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: assign_slot_to:slotID:studentID:subjectID:date
	parts := strings.Split(callback.Data, ":")
	if len(parts) < 3 {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	slotID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный ID слота")
		return
	}

	studentID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный ID студента")
		return
	}

	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil || !user.IsTeacher {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Доступ запрещен")
		return
	}

	// Закрепляем слот за студентом
	err = h.TeacherService.AssignSlotToStudent(ctx, slotID, user.ID, studentID)
	if err != nil {
		h.Logger.Error("Failed to assign slot to student", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось закрепить слот за студентом")
		return
	}

	// Получаем студента для уведомления
	student, err := h.UserService.GetByID(ctx, studentID)
	if err == nil && student != nil {
		slot, _ := h.TeacherService.GetSlotByID(ctx, slotID)
		if slot != nil {
			subject, _ := h.TeacherService.GetSubjectByID(ctx, slot.SubjectID)
			if subject != nil {
				notificationText := fmt.Sprintf(
					"📅 <b>Вам назначено занятие</b>\n\n"+
						"📚 Предмет: %s\n"+
						"📆 Дата: %s\n"+
						"🕐 Время: %s - %s\n\n"+
						"Преподаватель закрепил за вами это занятие.",
					subject.Name,
					slot.StartTime.Format("02.01.2006"),
					slot.StartTime.Format("15:04"),
					slot.EndTime.Format("15:04"))

				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:    student.TelegramID,
					Text:      notificationText,
					ParseMode: models.ParseModeHTML,
				})
			}
		}
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Слот закреплен за студентом")

	// Возвращаемся к экрану дня
	if len(parts) >= 5 {
		subjectID := parts[3]
		dateStr := parts[4]

		// Получаем слот для получения правильного weekday
		slot, err := h.TeacherService.GetSlotByID(ctx, slotID)
		if err == nil && slot != nil {
			callback.Data = fmt.Sprintf("view_schedule_day:%s:%s:%s", subjectID, dateStr, formatting.GetWeekdayName(int(slot.StartTime.Weekday())))
			HandleViewScheduleDay(ctx, b, callback, h)
		}
	}
}
