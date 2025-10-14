package recurring

import (
	"context"
	"fmt"
	"sort"
	"strconv"
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
// Recurring Schedule Management Handlers
// ========================

// HandleManageRecurring показывает управление постоянными расписаниями
func HandleManageRecurring(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleManageRecurring called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: manage_recurring:{subject_id} или manage_recurring:{subject_id}:{source}
	parts := strings.Split(callback.Data, ":")
	if len(parts) < 2 {
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

	// Определяем источник (откуда пришли)
	source := "mysubjects" // по умолчанию
	if len(parts) >= 3 {
		source = parts[2]
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

	// Получаем recurring schedules
	recurringSchedules, err := h.TeacherService.GetRecurringSchedulesBySubject(ctx, subjectID)
	if err != nil {
		h.Logger.Error("Failed to get recurring schedules", zap.Error(err))
		recurringSchedules = []*model.RecurringSchedule{}
	}

	text := fmt.Sprintf("🔄 <b>Постоянные расписания</b>\n\n<b>Предмет:</b> %s\n\n", subject.Name)

	var buttons [][]models.InlineKeyboardButton

	if len(recurringSchedules) == 0 {
		text += "У вас пока нет постоянных расписаний для этого предмета.\n\n"
		text += "Постоянное расписание автоматически создаёт слоты каждую неделю."
	} else {
		// Группируем расписания по group_id
		groupMap := make(map[int64][]*model.RecurringSchedule)
		for _, rs := range recurringSchedules {
			if !rs.IsActive {
				continue
			}
			groupID := rs.GroupID
			groupMap[groupID] = append(groupMap[groupID], rs)
		}

		text += fmt.Sprintf("У вас <b>%d</b> %s:\n\n", len(groupMap), formatting.PluralizeSchedules(len(groupMap)))

		// Сортируем группы для стабильного отображения
		var groupIDs []int64
		for groupID := range groupMap {
			groupIDs = append(groupIDs, groupID)
		}
		sort.Slice(groupIDs, func(i, j int) bool { return groupIDs[i] < groupIDs[j] })

		// Отображаем группы
		for i, groupID := range groupIDs {
			group := groupMap[groupID]
			if len(group) == 0 {
				continue
			}

			// Формируем отображаемый текст напрямую
			// Группировка уже сделана по group_id, не нужно использовать GroupRecurringSchedules
			displayText := formatGroupDisplay(group)

			buttons = append(buttons, []models.InlineKeyboardButton{
				{Text: displayText, CallbackData: fmt.Sprintf("view_recurring_group:%d", groupID)},
				{Text: "🗑", CallbackData: fmt.Sprintf("delete_recurring_group:%d", groupID)},
			})

			// Ограничиваем количество отображаемых групп
			if i >= 9 {
				text += fmt.Sprintf("\n... и ещё %d %s", len(groupMap)-10, formatting.PluralizeSchedules(len(groupMap)-10))
				break
			}
		}
	}

	// Кнопка создать новое
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "➕ Создать постоянное расписание", CallbackData: fmt.Sprintf("create_recurring_start:%d", subjectID)},
	})

	// Кнопка назад (зависит от источника)
	var backCallback string
	if source == "myschedule" {
		backCallback = "view_schedule"
	} else {
		backCallback = fmt.Sprintf("subject_schedule:%d", subjectID)
	}
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⬅️ Назад", CallbackData: backCallback},
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

// HandleViewRecurringGroup показывает детали группы постоянных расписаний
func HandleViewRecurringGroup(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleViewRecurringGroup called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: view_recurring_group:group_id или view_recurring_group:group_id:source
	parts := strings.Split(callback.Data, ":")
	if len(parts) < 2 {
		h.Logger.Error("Invalid callback format", zap.String("data", callback.Data))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	groupID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		h.Logger.Error("Invalid group_id", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный ID группы")
		return
	}

	// Определяем source из callback
	source := "mysubjects" // по умолчанию
	if len(parts) >= 3 {
		source = parts[2]
	}

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	telegramID := callback.From.ID
	_, err = h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем все расписания в группе
	groupSchedules, err := h.TeacherService.GetRecurringSchedulesByGroupID(ctx, groupID)
	if err != nil || len(groupSchedules) == 0 {
		h.Logger.Error("Failed to get schedules by group_id", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Расписания не найдены")
		return
	}

	// Получаем предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, groupSchedules[0].SubjectID)
	if err != nil || subject == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	// Собираем информацию о группе
	var weekdays []int
	weekdaySet := make(map[int]bool)
	minTime := "23:59"
	maxTime := "00:00"

	for _, rs := range groupSchedules {
		if !rs.IsActive {
			continue
		}

		if !weekdaySet[rs.Weekday] {
			weekdays = append(weekdays, rs.Weekday)
			weekdaySet[rs.Weekday] = true
		}

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

	sort.Ints(weekdays)

	timeRange := fmt.Sprintf("%s-%s", minTime, maxTime)
	if minTime == maxTime {
		timeRange = minTime
	}

	weekdayFullNames := map[int]string{
		0: "Воскресенье", 1: "Понедельник", 2: "Вторник",
		3: "Среда", 4: "Четверг", 5: "Пятница", 6: "Суббота",
	}

	// Формируем список дней
	var weekdaysList []string
	for _, wd := range weekdays {
		weekdaysList = append(weekdaysList, weekdayFullNames[wd])
	}

	text := fmt.Sprintf("🔄 <b>Детали постоянного расписания</b>\n\n"+
		"📚 Предмет: <b>%s</b>\n"+
		"📅 Дни недели: %s\n"+
		"🕐 Время: %s\n"+
		"⏱ Длительность: %d мин\n"+
		"📆 Создано: %s\n"+
		"📋 Слотов в группе: %d\n\n"+
		"Автоматически создаются слоты каждую неделю на месяц вперёд.",
		subject.Name,
		strings.Join(weekdaysList, ", "),
		timeRange,
		groupSchedules[0].DurationMinutes,
		groupSchedules[0].CreatedAt.Format("02.01.2006"),
		len(groupSchedules))

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "🗑 Удалить расписание", CallbackData: fmt.Sprintf("delete_recurring_group:%d:%s", groupID, source)},
			},
			{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("manage_recurring:%d:%s", subject.ID, source)},
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

// HandleDeleteRecurringGroup удаляет группу постоянных расписаний
func HandleDeleteRecurringGroup(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleDeleteRecurringGroup called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: delete_recurring_group:group_id или delete_recurring_group:group_id:source
	parts := strings.Split(callback.Data, ":")
	if len(parts) < 2 {
		h.Logger.Error("Invalid callback format", zap.String("data", callback.Data))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	groupID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		h.Logger.Error("Invalid group_id", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный ID группы")
		return
	}

	// Определяем source
	source := "mysubjects" // по умолчанию
	if len(parts) >= 3 {
		source = parts[2]
	}

	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем расписания группы для определения subject_id
	groupSchedules, err := h.TeacherService.GetRecurringSchedulesByGroupID(ctx, groupID)
	if err != nil || len(groupSchedules) == 0 {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Расписание не найдено")
		return
	}

	subjectID := groupSchedules[0].SubjectID
	deactivatedCount := len(groupSchedules)

	// Деактивируем всю группу
	err = h.TeacherService.DeactivateRecurringScheduleGroup(ctx, user.ID, groupID)
	if err != nil {
		h.Logger.Error("Failed to deactivate group", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка удаления расписания")
		return
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, fmt.Sprintf("✅ Удалено %d расписаний", deactivatedCount))

	// Возвращаемся к списку (обновляем экран)
	// Создаем новый callback_data для HandleManageRecurring с сохранением source
	newCallbackData := fmt.Sprintf("manage_recurring:%d:%s", subjectID, source)
	newCallback := &models.CallbackQuery{
		ID:      callback.ID,
		From:    callback.From,
		Data:    newCallbackData,
		Message: callback.Message,
	}
	HandleManageRecurring(ctx, b, newCallback, h)
}

// HandleToggleRecurring переключает активность recurring schedule
func HandleToggleRecurring(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	scheduleID, err := common.ParseIDFromCallback(callback.Data)
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

	// Получаем расписание
	schedule, err := h.TeacherService.GetRecurringSchedules(ctx, user.ID)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка получения расписания")
		return
	}

	// Находим нужное расписание
	var targetSchedule *model.RecurringSchedule
	for _, s := range schedule {
		if s.ID == scheduleID {
			targetSchedule = s
			break
		}
	}

	if targetSchedule == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Расписание не найдено")
		return
	}

	// Переключаем активность
	if targetSchedule.IsActive {
		err = h.TeacherService.DeactivateRecurringSchedule(ctx, user.ID, scheduleID)
	} else {
		// Для активации нужен отдельный метод, пока используем деактивацию
		common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Используйте создание нового расписания")
		return
	}

	if err != nil {
		h.Logger.Error("Failed to toggle recurring schedule", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось изменить статус")
		return
	}

	// Обновляем сообщение
	HandleManageRecurring(ctx, b, callback, h)
}

// formatGroupDisplay форматирует отображение группы расписаний
func formatGroupDisplay(schedules []*model.RecurringSchedule) string {
	if len(schedules) == 0 {
		return ""
	}

	// Собираем дни недели
	weekdaySet := make(map[int]bool)
	for _, rs := range schedules {
		if rs.IsActive {
			weekdaySet[rs.Weekday] = true
		}
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
		if !rs.IsActive {
			continue
		}
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
