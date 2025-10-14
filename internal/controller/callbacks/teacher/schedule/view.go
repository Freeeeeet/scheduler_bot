package schedule

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common/formatting"
	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// ========================
// Schedule View Handlers
// ========================

// HandleViewSchedule показывает общее расписание учителя с управлением
func HandleViewSchedule(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleViewSchedule called",
		zap.Int64("user_id", callback.From.ID))

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

	// Получаем предметы учителя
	subjects, err := h.TeacherService.GetTeacherSubjects(ctx, user.ID)
	if err != nil || len(subjects) == 0 {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    msg.Chat.ID,
			MessageID: msg.ID,
			Text:      "📅 У вас пока нет предметов.\n\nСоздайте предмет через /mysubjects",
		})
		common.AnswerCallback(ctx, b, callback.ID, "")
		return
	}

	// Получаем расписание на следующие 7 дней
	now := time.Now()
	endDate := now.AddDate(0, 0, 7)
	slots, err := h.TeacherService.GetTeacherSchedule(ctx, user.ID, now, endDate)
	if err != nil {
		h.Logger.Error("Failed to get schedule", zap.Error(err))
		slots = []*model.ScheduleSlot{}
	}

	text := "📅 <b>Управление расписанием</b>\n\n"

	if len(slots) == 0 {
		text += "У вас пока нет слотов на ближайшие 7 дней.\n\n"
		text += "Выберите предмет для создания слотов:"
	} else {
		text += fmt.Sprintf("📊 Всего слотов на 7 дней: %d\n\n", len(slots))

		// Группируем по предметам
		slotsBySubject := make(map[int64]int)
		for _, slot := range slots {
			slotsBySubject[slot.SubjectID]++
		}

		text += "Слотов по предметам:\n"
		for _, subj := range subjects {
			count := slotsBySubject[subj.ID]
			if count > 0 {
				text += fmt.Sprintf("  • %s: %d слотов\n", subj.Name, count)
			}
		}
		text += "\nВыберите предмет для управления:"
	}

	// Кнопки для каждого предмета
	var buttons [][]models.InlineKeyboardButton
	for _, subj := range subjects {
		emoji := "✅"
		if !subj.IsActive {
			emoji = "⏸"
		}
		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: fmt.Sprintf("%s %s", emoji, subj.Name), CallbackData: fmt.Sprintf("subject_schedule:%d", subj.ID)},
		})
	}

	// Кнопка назад
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⬅️ К списку предметов", CallbackData: "back_to_subjects"},
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

// HandleViewSubjectSchedule показывает расписание конкретного предмета с управлением
func HandleViewSubjectSchedule(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleViewSubjectSchedule called",
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

	// Получаем recurring schedules для этого предмета
	recurringSchedules, err := h.TeacherService.GetRecurringSchedulesBySubject(ctx, subjectID)
	if err != nil {
		h.Logger.Error("Failed to get recurring schedules", zap.Error(err))
		recurringSchedules = []*model.RecurringSchedule{}
	}

	// Получаем слоты на ближайшие 7 дней
	now := time.Now()
	endDate := now.AddDate(0, 0, 7)
	slots, err := h.TeacherService.GetTeacherSchedule(ctx, user.ID, now, endDate)
	if err != nil {
		h.Logger.Error("Failed to get schedule", zap.Error(err))
		slots = []*model.ScheduleSlot{}
	}

	// Фильтруем слоты только для этого предмета
	var subjectSlots []*model.ScheduleSlot
	for _, slot := range slots {
		if slot.SubjectID == subjectID {
			subjectSlots = append(subjectSlots, slot)
		}
	}

	// Подсчитываем статистику
	totalSlots := len(subjectSlots)
	bookedCount := 0
	freeCount := 0
	for _, slot := range subjectSlots {
		if slot.Status == model.SlotStatusBooked {
			bookedCount++
		} else if slot.Status == model.SlotStatusFree {
			freeCount++
		}
	}

	text := fmt.Sprintf("📅 <b>Расписание: %s</b>\n\n", subject.Name)

	// Добавляем статистику на 7 дней
	text += fmt.Sprintf("📊 <b>На ближайшие 7 дней:</b>\n"+
		"📋 Всего занятий: %d\n"+
		"👥 Записались учеников: %d\n"+
		"🟢 Свободных слотов: %d\n\n",
		totalSlots, bookedCount, freeCount)

	// Показываем recurring schedules с группировкой по group_id
	if len(recurringSchedules) > 0 {
		text += "🔄 <b>Постоянные расписания:</b>\n"

		// Группируем расписания по group_id
		groupMap := make(map[int64][]*model.RecurringSchedule)
		for _, rs := range recurringSchedules {
			if !rs.IsActive {
				continue
			}
			groupID := rs.GroupID
			groupMap[groupID] = append(groupMap[groupID], rs)
		}

		// Форматируем и выводим каждую группу
		for _, group := range groupMap {
			if len(group) == 0 {
				continue
			}
			displayText := formatRecurringGroupSummary(group)
			text += fmt.Sprintf("  • %s\n", displayText)
		}
		text += "\n"
	}

	// Показываем ближайшие слоты
	text += "📊 <b>Ближайшие слоты:</b>\n"
	if len(subjectSlots) > 0 {
		for i, slot := range subjectSlots {
			if i >= 5 {
				text += fmt.Sprintf("... и еще %d слотов\n", len(subjectSlots)-5)
				break
			}
			statusEmoji := "🟢"
			switch slot.Status {
			case "booked":
				statusEmoji = "🔴"
			case "canceled":
				statusEmoji = "⚫️"
			}
			text += fmt.Sprintf("%s %s в %s\n",
				statusEmoji,
				slot.StartTime.Format("02.01 (Mon)"),
				slot.StartTime.Format("15:04"))
		}
	} else {
		text += "📭 Нет слотов на ближайшие 7 дней\n"
	}

	// Кнопки управления (по ТЗ 2.3)
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "🔄 Постоянные расписания", CallbackData: fmt.Sprintf("manage_recurring:%d:mysubjects", subjectID)},
			},
			{
				{Text: "📅 Временные расписания", CallbackData: fmt.Sprintf("manage_temporary:%d", subjectID)},
			},
			{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("subject_schedule:%d", subjectID)},
			},
		},
	}

	h.Logger.Info("Updating message with subject schedule",
		zap.Int64("subject_id", subjectID),
		zap.Int64("chat_id", msg.Chat.ID),
		zap.Int("message_id", msg.ID),
		zap.Int("recurring_count", len(recurringSchedules)),
		zap.Int("slots_count", len(subjectSlots)))

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	if err != nil {
		// Игнорируем ошибку "message is not modified" - это не настоящая ошибка
		if !common.IsMessageNotModifiedError(err) {
			h.Logger.Error("Failed to edit message",
				zap.Error(err),
				zap.Int64("chat_id", msg.Chat.ID),
				zap.Int("message_id", msg.ID))
			common.AnswerCallbackAlert(ctx, b, callback.ID, fmt.Sprintf("❌ Ошибка: %v", err))
			return
		}
		h.Logger.Debug("Message content unchanged, skipping update")
	}

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleViewAllSlots показывает все слоты предмета
func HandleViewAllSlots(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleViewAllSlots called",
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

	// Получаем все слоты на следующие 30 дней
	now := time.Now()
	endDate := now.AddDate(0, 0, 30)
	allSlots, err := h.TeacherService.GetTeacherSchedule(ctx, user.ID, now, endDate)
	if err != nil {
		h.Logger.Error("Failed to get schedule", zap.Error(err))
		allSlots = []*model.ScheduleSlot{}
	}

	// Фильтруем только этот предмет
	var slots []*model.ScheduleSlot
	for _, slot := range allSlots {
		if slot.SubjectID == subjectID {
			slots = append(slots, slot)
		}
	}

	text := fmt.Sprintf("📋 <b>Все слоты: %s</b>\n\n", subject.Name)

	if len(slots) == 0 {
		text += "📭 Нет слотов на ближайшие 30 дней\n\n"
		text += "Создайте слоты через кнопку \"➕ Добавить слоты\""
	} else {
		// Группируем по статусу
		var freeSlots, bookedSlots, canceledSlots int
		for _, slot := range slots {
			switch slot.Status {
			case model.SlotStatusFree:
				freeSlots++
			case model.SlotStatusBooked:
				bookedSlots++
			case model.SlotStatusCanceled:
				canceledSlots++
			}
		}

		text += fmt.Sprintf("📊 <b>Статистика (30 дней):</b>\n"+
			"🟢 Свободно: %d\n"+
			"🔴 Забронировано: %d\n"+
			"⚫️ Отменено: %d\n"+
			"<b>Всего:</b> %d слотов\n\n",
			freeSlots, bookedSlots, canceledSlots, len(slots))

		// Показываем первые 10 слотов
		text += "<b>Ближайшие слоты:</b>\n"
		displayCount := 10
		if len(slots) < displayCount {
			displayCount = len(slots)
		}

		for i := 0; i < displayCount; i++ {
			slot := slots[i]
			statusEmoji := "🟢"
			statusText := "Свободен"
			switch slot.Status {
			case model.SlotStatusBooked:
				statusEmoji = "🔴"
				statusText = "Занят"
			case model.SlotStatusCanceled:
				statusEmoji = "⚫️"
				statusText = "Отменён"
			}

			text += fmt.Sprintf("%s %s %s - %s\n",
				statusEmoji,
				slot.StartTime.Format("02.01 (Mon)"),
				slot.StartTime.Format("15:04"),
				statusText)
		}

		if len(slots) > displayCount {
			text += fmt.Sprintf("\n... и ещё %d слотов", len(slots)-displayCount)
		}
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "➕ Добавить слоты", CallbackData: fmt.Sprintf("create_slots:%d", subjectID)},
			},
			{
				{Text: "🗑 Удалить свободные слоты", CallbackData: fmt.Sprintf("delete_free_slots:%d", subjectID)},
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

// formatRecurringGroupSummary форматирует краткую информацию о группе расписаний
func formatRecurringGroupSummary(schedules []*model.RecurringSchedule) string {
	if len(schedules) == 0 {
		return ""
	}

	// Собираем дни недели
	weekdaySet := make(map[int]bool)
	for _, rs := range schedules {
		weekdaySet[rs.Weekday] = true
	}

	var weekdays []int
	for wd := range weekdaySet {
		weekdays = append(weekdays, wd)
	}
	sort.Ints(weekdays)

	// Собираем время
	minTime := "23:59"
	maxTime := "00:00"
	for _, rs := range schedules {
		timeStr := fmt.Sprintf("%02d:%02d", rs.StartHour, rs.StartMinute)
		if timeStr < minTime {
			minTime = timeStr
		}
		endTime := time.Date(2000, 1, 1, rs.StartHour, rs.StartMinute, 0, 0, time.UTC).
			Add(time.Duration(rs.DurationMinutes) * time.Minute)
		endTimeStr := endTime.Format("15:04")
		if endTimeStr > maxTime {
			maxTime = endTimeStr
		}
	}

	// Форматируем дни недели
	var weekdayNames []string
	for _, wd := range weekdays {
		weekdayNames = append(weekdayNames, formatting.GetWeekdayShortName(wd))
	}

	timeRange := fmt.Sprintf("%s-%s", minTime, maxTime)
	if minTime == maxTime {
		timeRange = minTime
	}

	return fmt.Sprintf("%s: %s", strings.Join(weekdayNames, ","), timeRange)
}

// HandleViewScheduleCalendar показывает календарь для выбора дня просмотра расписания
