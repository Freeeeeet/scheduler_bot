package recurring

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common/formatting"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// ========================
// Create Recurring Schedule Handlers (New Flow with Multiple Weekdays)
// ========================

// HandleCreateRecurringStart начинает процесс создания постоянного расписания с выбором нескольких дней
func HandleCreateRecurringStart(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleCreateRecurringStart called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: create_recurring_start:123
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

	// Инициализируем пустой выбор дней
	selectedWeekdays := make(map[int]bool)
	h.StateManager.SetState(telegramID, "create_recurring_select_days")
	h.StateManager.SetData(telegramID, "subject_id", subjectID)
	h.StateManager.SetData(telegramID, "selected_weekdays", selectedWeekdays)

	showCreateRecurringDaysSelection(ctx, b, callback, h, msg, subjectID, selectedWeekdays)
}

// showCreateRecurringDaysSelection показывает интерфейс выбора дней недели
func showCreateRecurringDaysSelection(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler, msg *models.Message, subjectID int64, selectedWeekdays map[int]bool) {
	// Получаем предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		h.Logger.Error("Failed to get subject", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	text := fmt.Sprintf("🔄 <b>Создание постоянного расписания</b>\n\n"+
		"📚 Предмет: <b>%s</b>\n"+
		"⏱ Длительность: %d мин\n\n"+
		"<b>Шаг 1/3: Выберите дни недели</b>\n\n"+
		"Выберите один или несколько дней:\n"+
		"✅ - день выбран\n"+
		"⬜️ - день не выбран",
		subject.Name,
		subject.Duration)

	var buttons [][]models.InlineKeyboardButton

	// Кнопки для каждого дня недели
	weekdayOrder := []int{1, 2, 3, 4, 5, 6, 0} // Пн-Вс
	for _, wd := range weekdayOrder {
		emoji := "⬜️"
		if selectedWeekdays[wd] {
			emoji = "✅"
		}
		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: fmt.Sprintf("%s %s", emoji, formatting.GetWeekdayName(wd)), CallbackData: fmt.Sprintf("toggle_create_weekday:%d:%d", subjectID, wd)},
		})
	}

	// Проверяем, выбран ли хотя бы один день
	hasSelected := false
	for _, selected := range selectedWeekdays {
		if selected {
			hasSelected = true
			break
		}
	}

	// Кнопка "Продолжить" (активна только если выбран хотя бы один день)
	if hasSelected {
		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: "➡️ Продолжить", CallbackData: fmt.Sprintf("create_recurring_continue:%d", subjectID)},
		})
	}

	// Кнопка "Назад"
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("manage_recurring:%d", subjectID)},
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

// HandleToggleCreateWeekday переключает день недели при создании
func HandleToggleCreateWeekday(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleToggleCreateWeekday called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: toggle_create_weekday:123:1
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 3 {
		h.Logger.Error("Invalid callback format", zap.String("data", callback.Data))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	subjectID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный ID")
		return
	}

	weekday, err := strconv.Atoi(parts[2])
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный день")
		return
	}

	telegramID := callback.From.ID

	// Получаем текущий выбор из state
	selectedData, ok := h.StateManager.GetData(telegramID, "selected_weekdays")
	if !ok {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Сессия истекла, начните заново")
		return
	}

	selectedWeekdays, ok := selectedData.(map[int]bool)
	if !ok {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка данных")
		return
	}

	// Переключаем день
	selectedWeekdays[weekday] = !selectedWeekdays[weekday]
	h.StateManager.SetData(telegramID, "selected_weekdays", selectedWeekdays)

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Перерисовываем интерфейс
	showCreateRecurringDaysSelection(ctx, b, callback, h, msg, subjectID, selectedWeekdays)
}

// HandleCreateRecurringContinue переходит к выбору режима времени
func HandleCreateRecurringContinue(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleCreateRecurringContinue called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: create_recurring_continue:123
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

	telegramID := callback.From.ID

	// Проверяем что дни выбраны
	selectedData, ok := h.StateManager.GetData(telegramID, "selected_weekdays")
	if !ok {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Сессия истекла")
		return
	}

	selectedWeekdays, ok := selectedData.(map[int]bool)
	if !ok {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка данных")
		return
	}

	hasSelected := false
	for _, selected := range selectedWeekdays {
		if selected {
			hasSelected = true
			break
		}
	}

	if !hasSelected {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Выберите хотя бы один день")
		return
	}

	// Получаем предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	// Переходим к выбору режима времени
	h.StateManager.SetState(telegramID, "create_recurring_select_time_mode")

	text := fmt.Sprintf("🔄 <b>Создание постоянного расписания</b>\n\n"+
		"📚 Предмет: <b>%s</b>\n"+
		"⏱ Длительность: %d мин\n\n"+
		"<b>Шаг 2/3: Выберите режим времени</b>\n\n"+
		"Выберите как задать время для слотов:",
		subject.Name,
		subject.Duration)

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "⏰ Временной интервал", CallbackData: fmt.Sprintf("recurring_time_mode:%d:interval", subjectID)},
			},
			{
				{Text: "🕐 Конкретные слоты", CallbackData: fmt.Sprintf("recurring_time_mode:%d:specific", subjectID)},
			},
			{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("create_recurring_start:%d", subjectID)},
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

// HandleRecurringTimeMode обрабатывает выбор режима времени
func HandleRecurringTimeMode(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleRecurringTimeMode called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: recurring_time_mode:123:interval
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

	mode := parts[2]
	telegramID := callback.From.ID

	// Сохраняем режим
	h.StateManager.SetData(telegramID, "time_mode", mode)

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	switch mode {
	case "interval":
		showRecurringIntervalSelection(ctx, b, callback, h, msg, subjectID)
	case "specific":
		showRecurringSpecificSlotsSelection(ctx, b, callback, h, msg, subjectID)
	default:
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неизвестный режим")
	}
}

// showRecurringIntervalSelection показывает выбор временного интервала
func showRecurringIntervalSelection(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler, msg *models.Message, subjectID int64) {
	// Получаем предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	telegramID := callback.From.ID
	h.StateManager.SetState(telegramID, "create_recurring_interval_start")

	text := fmt.Sprintf("🔄 <b>Создание постоянного расписания</b>\n\n"+
		"📚 Предмет: <b>%s</b>\n"+
		"⏱ Длительность: %d мин\n\n"+
		"<b>Шаг 3/3: Временной интервал</b>\n\n"+
		"Выберите <b>начало интервала</b>:\n"+
		"(Слоты будут автоматически созданы от начала до конца с учётом длительности)",
		subject.Name,
		subject.Duration)

	// Генерируем кнопки времени (с 00:00 до 23:00)
	var buttons [][]models.InlineKeyboardButton
	var row []models.InlineKeyboardButton

	for hour := 0; hour < 24; hour++ {
		timeStr := fmt.Sprintf("%02d:00", hour)
		row = append(row, models.InlineKeyboardButton{
			Text:         timeStr,
			CallbackData: fmt.Sprintf("recurring_interval_start:%d:%d:0", subjectID, hour),
		})

		if len(row) == 3 {
			buttons = append(buttons, row)
			row = []models.InlineKeyboardButton{}
		}
	}

	if len(row) > 0 {
		buttons = append(buttons, row)
	}

	// Кнопка для ввода своего времени
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⌨️ Ввести своё время", CallbackData: fmt.Sprintf("recurring_custom_start:%d", subjectID)},
	})

	// Кнопка назад
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("create_recurring_continue:%d", subjectID)},
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

// HandleRecurringIntervalStart обрабатывает выбор начала интервала
func HandleRecurringIntervalStart(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleRecurringIntervalStart called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: recurring_interval_start:123:9:0
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 4 {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	subjectID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный ID")
		return
	}

	startHour, err := strconv.Atoi(parts[2])
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный час")
		return
	}

	startMinute, err := strconv.Atoi(parts[3])
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверные минуты")
		return
	}

	telegramID := callback.From.ID

	// Сохраняем начало интервала
	h.StateManager.SetData(telegramID, "interval_start_hour", startHour)
	h.StateManager.SetData(telegramID, "interval_start_minute", startMinute)
	h.StateManager.SetState(telegramID, "create_recurring_interval_end")

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Получаем предмет для расчета допустимого конца
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	// Показываем выбор конца интервала
	text := fmt.Sprintf("🔄 <b>Создание постоянного расписания</b>\n\n"+
		"📚 Предмет: <b>%s</b>\n"+
		"⏱ Длительность: %d мин\n\n"+
		"<b>Шаг 3/3: Временной интервал</b>\n\n"+
		"Начало: <b>%02d:%02d</b>\n\n"+
		"Выберите <b>конец интервала</b>:\n"+
		"(Минимум: начало + длительность занятия)",
		subject.Name,
		subject.Duration,
		startHour,
		startMinute)

	// Генерируем кнопки для конца интервала
	// Минимальный конец = начало + длительность
	minEndTime := time.Date(2000, 1, 1, startHour, startMinute, 0, 0, time.UTC).Add(time.Duration(subject.Duration) * time.Minute)
	minEndHour := minEndTime.Hour()
	minEndMinute := minEndTime.Minute()

	var buttons [][]models.InlineKeyboardButton
	var row []models.InlineKeyboardButton

	for hour := minEndHour; hour <= 23; hour++ {
		// Для первого часа учитываем минуты
		if hour == minEndHour && minEndMinute > 0 {
			continue // Пропускаем, если минуты не :00
		}

		timeStr := fmt.Sprintf("%02d:00", hour)
		row = append(row, models.InlineKeyboardButton{
			Text:         timeStr,
			CallbackData: fmt.Sprintf("recurring_interval_end:%d:%d:0", subjectID, hour),
		})

		if len(row) == 3 {
			buttons = append(buttons, row)
			row = []models.InlineKeyboardButton{}
		}
	}

	if len(row) > 0 {
		buttons = append(buttons, row)
	}

	// Кнопка для ввода своего времени
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⌨️ Ввести своё время", CallbackData: fmt.Sprintf("recurring_custom_end:%d", subjectID)},
	})

	// Кнопка назад
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("recurring_time_mode:%d:interval", subjectID)},
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

// HandleRecurringIntervalEnd обрабатывает выбор конца интервала и создаёт расписание
func HandleRecurringIntervalEnd(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleRecurringIntervalEnd called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: recurring_interval_end:123:18:0
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 4 {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	subjectID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный ID")
		return
	}

	endHour, err := strconv.Atoi(parts[2])
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный час")
		return
	}

	endMinute, err := strconv.Atoi(parts[3])
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверные минуты")
		return
	}

	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем данные из state
	selectedData, ok := h.StateManager.GetData(telegramID, "selected_weekdays")
	if !ok {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Сессия истекла")
		return
	}

	selectedWeekdays, ok := selectedData.(map[int]bool)
	if !ok {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка данных")
		return
	}

	startHourData, ok := h.StateManager.GetData(telegramID, "interval_start_hour")
	if !ok {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не найдено начало интервала")
		return
	}
	startHour, _ := startHourData.(int)

	startMinuteData, ok := h.StateManager.GetData(telegramID, "interval_start_minute")
	if !ok {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не найдено начало интервала")
		return
	}
	startMinute, _ := startMinuteData.(int)

	// Получаем предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	// Создаём recurring schedules для каждого выбранного дня и каждого времени в интервале
	startTime := time.Date(2000, 1, 1, startHour, startMinute, 0, 0, time.UTC)
	endTime := time.Date(2000, 1, 1, endHour, endMinute, 0, 0, time.UTC)

	// Генерируем слоты по времени с шагом = длительность занятия
	currentTime := startTime
	var timeSlots []struct{ Hour, Minute int }

	for currentTime.Before(endTime) || currentTime.Equal(endTime.Add(-time.Duration(subject.Duration)*time.Minute)) {
		timeSlots = append(timeSlots, struct{ Hour, Minute int }{
			Hour:   currentTime.Hour(),
			Minute: currentTime.Minute(),
		})
		currentTime = currentTime.Add(time.Duration(subject.Duration) * time.Minute)
	}

	// Собираем выбранные дни недели
	var weekdays []int
	for weekday, selected := range selectedWeekdays {
		if selected {
			weekdays = append(weekdays, weekday)
		}
	}

	// Создаём группу расписаний одним вызовом
	groupID, err := h.TeacherService.CreateWeeklySlotsGroup(ctx, user.ID, subjectID, weekdays, timeSlots, subject.Duration)
	if err != nil {
		h.Logger.Error("Failed to create recurring schedule group", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка создания расписания")
		return
	}

	totalCreated := len(weekdays) * len(timeSlots)

	h.Logger.Info("Created recurring schedule group",
		zap.String("group_id", groupID.String()),
		zap.Int("total_schedules", totalCreated))

	// Очищаем state
	h.StateManager.ClearState(telegramID)

	msg := common.GetMessageFromCallback(callback)
	if msg != nil {
		// Формируем список выбранных дней
		var selectedDaysList []string
		for wd := 0; wd <= 6; wd++ {
			if selectedWeekdays[wd] {
				selectedDaysList = append(selectedDaysList, formatting.GetWeekdayName(wd))
			}
		}

		text := fmt.Sprintf("✅ Постоянное расписание создано!\n\n"+
			"📚 Предмет: %s\n"+
			"📅 Дни: %s\n"+
			"📊 Создано %d временных слотов\n\n"+
			"Посмотреть расписание: /myschedule",
			subject.Name,
			strings.Join(selectedDaysList, ", "),
			totalCreated)

		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    msg.Chat.ID,
			MessageID: msg.ID,
			Text:      text,
		})
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Расписание создано!")
}

// showRecurringSpecificSlotsSelection показывает выбор конкретных временных слотов
func showRecurringSpecificSlotsSelection(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler, msg *models.Message, subjectID int64) {
	// Получаем предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	telegramID := callback.From.ID

	// Инициализируем выбранные слоты если их еще нет
	selectedSlotsData, ok := h.StateManager.GetData(telegramID, "selected_time_slots")
	var selectedSlots map[string]bool
	if !ok {
		selectedSlots = make(map[string]bool)
		h.StateManager.SetData(telegramID, "selected_time_slots", selectedSlots)
	} else {
		selectedSlots, _ = selectedSlotsData.(map[string]bool)
	}

	h.StateManager.SetState(telegramID, "create_recurring_specific_slots")

	text := fmt.Sprintf("🔄 <b>Создание постоянного расписания</b>\n\n"+
		"📚 Предмет: <b>%s</b>\n"+
		"⏱ Длительность: %d мин\n\n"+
		"<b>Шаг 3/3: Конкретные слоты</b>\n\n"+
		"Выберите один или несколько слотов времени:\n"+
		"✅ - слот выбран\n"+
		"⬜️ - слот не выбран",
		subject.Name,
		subject.Duration)

	// Генерируем кнопки времени (с учетом длительности)
	var buttons [][]models.InlineKeyboardButton
	var row []models.InlineKeyboardButton

	now := time.Now()
	currentTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 0, 0, now.Location())

	slotCount := 0
	for currentTime.Before(endOfDay) && slotCount < 24 { // Ограничиваем 24 слотами для компактности
		timeStr := currentTime.Format("15:04")
		emoji := "⬜️"
		if selectedSlots[timeStr] {
			emoji = "✅"
		}

		row = append(row, models.InlineKeyboardButton{
			Text:         fmt.Sprintf("%s %s", emoji, timeStr),
			CallbackData: fmt.Sprintf("toggle_time_slot:%d:%s", subjectID, timeStr),
		})

		if len(row) == 2 {
			buttons = append(buttons, row)
			row = []models.InlineKeyboardButton{}
		}

		currentTime = currentTime.Add(time.Duration(subject.Duration) * time.Minute)
		slotCount++
	}

	if len(row) > 0 {
		buttons = append(buttons, row)
	}

	// Проверяем, выбран ли хотя бы один слот
	hasSelected := false
	for _, selected := range selectedSlots {
		if selected {
			hasSelected = true
			break
		}
	}

	// Кнопка "Создать" (активна только если выбран хотя бы один слот)
	if hasSelected {
		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: "✅ Создать расписание", CallbackData: fmt.Sprintf("create_recurring_specific_confirm:%d", subjectID)},
		})
	}

	// Кнопка назад
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("create_recurring_continue:%d", subjectID)},
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

// HandleToggleTimeSlot переключает выбор временного слота
func HandleToggleTimeSlot(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleToggleTimeSlot called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: toggle_time_slot:123:09:00
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 4 {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	subjectID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный ID")
		return
	}

	timeStr := parts[2] + ":" + parts[3]
	telegramID := callback.From.ID

	// Получаем текущий выбор из state
	selectedData, ok := h.StateManager.GetData(telegramID, "selected_time_slots")
	if !ok {
		selectedData = make(map[string]bool)
		h.StateManager.SetData(telegramID, "selected_time_slots", selectedData)
	}

	selectedSlots, ok := selectedData.(map[string]bool)
	if !ok {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка данных")
		return
	}

	// Переключаем слот
	selectedSlots[timeStr] = !selectedSlots[timeStr]
	h.StateManager.SetData(telegramID, "selected_time_slots", selectedSlots)

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Перерисовываем интерфейс
	showRecurringSpecificSlotsSelection(ctx, b, callback, h, msg, subjectID)
}

// HandleCreateRecurringSpecificConfirm создаёт расписание с конкретными слотами
func HandleCreateRecurringSpecificConfirm(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleCreateRecurringSpecificConfirm called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: create_recurring_specific_confirm:123
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

	// Получаем данные из state
	selectedWeekdaysData, ok := h.StateManager.GetData(telegramID, "selected_weekdays")
	if !ok {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Сессия истекла")
		return
	}

	selectedWeekdays, ok := selectedWeekdaysData.(map[int]bool)
	if !ok {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка данных дней")
		return
	}

	selectedSlotsData, ok := h.StateManager.GetData(telegramID, "selected_time_slots")
	if !ok {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не выбраны слоты времени")
		return
	}

	selectedSlots, ok := selectedSlotsData.(map[string]bool)
	if !ok {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка данных времени")
		return
	}

	// Получаем предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	// Собираем выбранные дни недели
	var weekdays []int
	for weekday, selectedDay := range selectedWeekdays {
		if selectedDay {
			weekdays = append(weekdays, weekday)
		}
	}

	// Собираем выбранные временные слоты
	var timeSlots []struct{ Hour, Minute int }
	for timeStr, selectedTime := range selectedSlots {
		if !selectedTime {
			continue
		}

		// Парсим время
		timeParts := strings.Split(timeStr, ":")
		if len(timeParts) != 2 {
			continue
		}
		hour, _ := strconv.Atoi(timeParts[0])
		minute, _ := strconv.Atoi(timeParts[1])

		timeSlots = append(timeSlots, struct{ Hour, Minute int }{
			Hour:   hour,
			Minute: minute,
		})
	}

	// Создаём группу расписаний одним вызовом
	groupID, err := h.TeacherService.CreateWeeklySlotsGroup(ctx, user.ID, subjectID, weekdays, timeSlots, subject.Duration)
	if err != nil {
		h.Logger.Error("Failed to create recurring schedule group", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка создания расписания")
		return
	}

	totalCreated := len(weekdays) * len(timeSlots)

	h.Logger.Info("Created recurring schedule group",
		zap.String("group_id", groupID.String()),
		zap.Int("total_schedules", totalCreated))

	// Очищаем state
	h.StateManager.ClearState(telegramID)

	msg := common.GetMessageFromCallback(callback)
	if msg != nil {
		// Формируем список выбранных дней
		var selectedDaysList []string
		for wd := 0; wd <= 6; wd++ {
			if selectedWeekdays[wd] {
				selectedDaysList = append(selectedDaysList, formatting.GetWeekdayName(wd))
			}
		}

		text := fmt.Sprintf("✅ Постоянное расписание создано!\n\n"+
			"📚 Предмет: %s\n"+
			"📅 Дни: %s\n"+
			"📊 Создано %d временных слотов\n\n"+
			"Посмотреть расписание: /myschedule",
			subject.Name,
			strings.Join(selectedDaysList, ", "),
			totalCreated)

		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    msg.Chat.ID,
			MessageID: msg.ID,
			Text:      text,
		})
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Расписание создано!")
}
