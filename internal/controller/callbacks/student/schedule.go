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
		text := fmt.Sprintf("📅 Расписание: <b>%s</b>\n\n"+
			"К сожалению, сейчас нет доступных слотов на ближайшие 2 недели.\n\n"+
			"Попробуйте позже или выберите другой предмет.",
			subject.Name)

		keyboard := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: "⬅️ К деталям предмета", CallbackData: fmt.Sprintf("view_subject:%d", subjectID)}},
				{{Text: "📚 К списку предметов", CallbackData: "book_another"}},
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
		return
	}

	// Группируем слоты по дням
	slotsByDate := make(map[string][]*model.ScheduleSlot)
	for _, slot := range slots {
		dateKey := slot.StartTime.Format("2006-01-02")
		slotsByDate[dateKey] = append(slotsByDate[dateKey], slot)
	}

	// Формируем текст и кнопки
	text := fmt.Sprintf("📅 <b>Расписание: %s</b>\n\n"+
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

	// Добавляем дополнительные опции
	if len(slots) > 10 {
		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: "📋 Все слоты (2 недели)", CallbackData: fmt.Sprintf("view_all_student_slots:%d:14", subjectID)},
		})
	}

	// Кнопка для просмотра слотов за пределами 2 недель
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "🔮 Слоты на месяц вперёд", CallbackData: fmt.Sprintf("view_extended_slots:%d", subjectID)},
	})

	// Кнопка для постоянной записи
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "🔄 Записаться на постоянной основе", CallbackData: fmt.Sprintf("request_recurring_booking:%d", subjectID)},
	})

	// Добавляем кнопки навигации
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⬅️ К деталям предмета", CallbackData: fmt.Sprintf("view_subject:%d", subjectID)},
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

// HandleViewExtendedSlots показывает слоты на месяц вперёд (за пределами 2 недель)
func HandleViewExtendedSlots(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleViewExtendedSlots called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

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

	// Получаем предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		h.Logger.Error("Subject not found", zap.Int64("subject_id", subjectID), zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	// Получаем слоты с 14 дней до 30 дней
	startDate := time.Now().AddDate(0, 0, 14)
	endDate := time.Now().AddDate(0, 0, 30)

	h.Logger.Info("Fetching extended slots",
		zap.Int64("subject_id", subjectID),
		zap.Time("from", startDate),
		zap.Time("to", endDate))

	slots, err := h.BookingService.GetAvailableSlots(ctx, subjectID, startDate, endDate)
	if err != nil {
		h.Logger.Error("Failed to get extended slots", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось загрузить слоты")
		return
	}

	text := fmt.Sprintf("🔮 <b>Расширенное расписание: %s</b>\n\n"+
		"📅 Период: с %s по %s\n"+
		"💰 Цена: %.2f ₽ | ⏱ %d мин\n\n",
		subject.Name,
		startDate.Format("02.01"),
		endDate.Format("02.01"),
		float64(subject.Price)/100,
		subject.Duration)

	if len(slots) == 0 {
		text += "📭 Нет доступных слотов в этом периоде.\n\n"
		text += "💡 Возможно, преподаватель ещё не добавил слоты на этот период."
	} else {
		text += fmt.Sprintf("Доступно: %d слотов\n\n", len(slots))

		var buttons [][]models.InlineKeyboardButton
		count := 0

		for _, slot := range slots {
			if count >= 15 {
				text += fmt.Sprintf("\n... и ещё %d слотов", len(slots)-15)
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

		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: "⬅️ Вернуться к ближайшим", CallbackData: fmt.Sprintf("view_schedule_subject:%d", subjectID)},
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
		return
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "⬅️ Вернуться к ближайшим", CallbackData: fmt.Sprintf("view_schedule_subject:%d", subjectID)}},
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

// HandleRequestRecurringBooking обрабатывает запрос на постоянную запись
func HandleRequestRecurringBooking(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleRequestRecurringBooking called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

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
	student, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || student == nil {
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

	// Получаем recurring schedules этого предмета
	recurringSchedules, err := h.TeacherService.GetRecurringSchedulesBySubject(ctx, subjectID)
	if err != nil {
		h.Logger.Error("Failed to get recurring schedules", zap.Error(err))
		recurringSchedules = []*model.RecurringSchedule{}
	}

	text := fmt.Sprintf("🔄 <b>Постоянная запись: %s</b>\n\n"+
		"💰 Цена: %.2f ₽ | ⏱ %d мин\n\n",
		subject.Name,
		float64(subject.Price)/100,
		subject.Duration)

	if len(recurringSchedules) == 0 {
		text += "❌ К сожалению, у этого предмета нет постоянного расписания.\n\n"
		text += "Преподаватель не настроил регулярные слоты.\n"
		text += "Вы можете записаться на разовые занятия."

		keyboard := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: "⬅️ К расписанию", CallbackData: fmt.Sprintf("view_schedule_subject:%d", subjectID)}},
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
		return
	}

	text += "📋 <b>Доступные постоянные расписания:</b>\n\n"
	text += "Выберите день и время для регулярных занятий:\n\n"

	weekdayNames := map[int]string{
		0: "Воскресенье", 1: "Понедельник", 2: "Вторник",
		3: "Среда", 4: "Четверг", 5: "Пятница", 6: "Суббота",
	}

	var buttons [][]models.InlineKeyboardButton

	for _, rs := range recurringSchedules {
		if !rs.IsActive {
			continue
		}

		scheduleText := fmt.Sprintf("📅 %s в %02d:%02d",
			weekdayNames[rs.Weekday], rs.StartHour, rs.StartMinute)

		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: scheduleText, CallbackData: fmt.Sprintf("request_recurring_confirm:%d:%d", subjectID, rs.ID)},
		})
	}

	if len(buttons) == 0 {
		text += "❌ Нет активных постоянных расписаний."
		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: "⬅️ К расписанию", CallbackData: fmt.Sprintf("view_schedule_subject:%d", subjectID)},
		})
	} else {
		text += "\n⚠️ <b>Важно:</b>\n"
		text += "• Запись на постоянной основе требует подтверждения преподавателя\n"
		text += "• Преподаватель будет уведомлён о вашем запросе\n"
		text += "• После подтверждения вы будете автоматически записаны на все слоты этого расписания\n"

		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: "⬅️ Отмена", CallbackData: fmt.Sprintf("view_schedule_subject:%d", subjectID)},
		})
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

// HandleRequestRecurringConfirm подтверждает запрос на постоянную запись
func HandleRequestRecurringConfirm(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleRequestRecurringConfirm called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Парсим callback data: request_recurring_confirm:subjectID:scheduleID
	parts := common.ParseMultiIDFromCallback(callback.Data, "request_recurring_confirm:")
	if len(parts) != 2 {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	subjectID := parts[0]
	scheduleID := parts[1]

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	telegramID := callback.From.ID
	student, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || student == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем предмет и расписание
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	schedules, err := h.TeacherService.GetRecurringSchedules(ctx, subject.TeacherID)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка получения расписания")
		return
	}

	var targetSchedule *model.RecurringSchedule
	for _, s := range schedules {
		if s.ID == scheduleID {
			targetSchedule = s
			break
		}
	}

	if targetSchedule == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Расписание не найдено")
		return
	}

	weekdayNames := map[int]string{
		0: "Воскресенье", 1: "Понедельник", 2: "Вторник",
		3: "Среда", 4: "Четверг", 5: "Пятница", 6: "Суббота",
	}

	// Отправляем уведомление преподавателю
	teacher, err := h.UserService.GetByID(ctx, subject.TeacherID)
	if err == nil && teacher != nil {
		notificationText := fmt.Sprintf(
			"📩 <b>Новый запрос на постоянную запись</b>\n\n"+
				"👤 Студент: %s %s (@%s)\n"+
				"📚 Предмет: %s\n"+
				"📅 Расписание: %s в %02d:%02d\n\n"+
				"Что вы хотите сделать?",
			student.FirstName, student.LastName, student.Username,
			subject.Name,
			weekdayNames[targetSchedule.Weekday],
			targetSchedule.StartHour, targetSchedule.StartMinute)

		keyboard := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "✅ Одобрить", CallbackData: fmt.Sprintf("approve_recurring:%d:%d:%d", scheduleID, student.ID, subjectID)},
					{Text: "❌ Отклонить", CallbackData: fmt.Sprintf("reject_recurring:%d:%d:%d", scheduleID, student.ID, subjectID)},
				},
			},
		}

		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      teacher.TelegramID,
			Text:        notificationText,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: keyboard,
		})

		if err != nil {
			h.Logger.Error("Failed to send notification to teacher", zap.Error(err))
		}
	}

	// Уведомляем студента
	text := fmt.Sprintf(
		"✅ <b>Запрос отправлен!</b>\n\n"+
			"📚 Предмет: %s\n"+
			"📅 Расписание: %s в %02d:%02d\n\n"+
			"Преподаватель получил ваш запрос.\n"+
			"Вы получите уведомление, когда он примет решение.",
		subject.Name,
		weekdayNames[targetSchedule.Weekday],
		targetSchedule.StartHour, targetSchedule.StartMinute)

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "⬅️ К расписанию", CallbackData: fmt.Sprintf("view_schedule_subject:%d", subjectID)}},
			{{Text: "📚 К списку предметов", CallbackData: "book_another"}},
		},
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "✅ Запрос отправлен!")
}
