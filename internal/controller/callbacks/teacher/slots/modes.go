package slots

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

// HandleSlotMode обрабатывает выбор режима создания слотов
func HandleSlotMode(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleSlotMode called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: slot_mode:123:weekly
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

	mode := parts[2]
	telegramID := callback.From.ID

	h.Logger.Info("Slot mode selected",
		zap.Int64("subject_id", subjectID),
		zap.String("mode", mode))

	// Сохраняем режим
	h.StateManager.SetData(telegramID, "slot_mode", mode)

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	switch mode {
	case "single":
		// Один раз - показываем выбор конкретной даты
		showSingleDaySelection(ctx, b, callback, h, msg, subjectID)
	case "recurring":
		// Постоянное расписание - показываем выбор дня недели
		showRecurringScheduleSelection(ctx, b, callback, h, msg, subjectID)
	case "period":
		// На период - показываем выбор периода
		showPeriodSelection(ctx, b, callback, h, msg, subjectID)
	case "workday":
		// Рабочий день - показываем выбор дня недели
		showWorkdaySelection(ctx, b, callback, h, msg, subjectID)
	default:
		h.Logger.Error("Unknown slot mode", zap.String("mode", mode))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неизвестный режим")
	}
}

// showRecurringScheduleSelection показывает выбор дня недели для постоянного расписания
func showRecurringScheduleSelection(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler, msg *models.Message, subjectID int64) {
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Понедельник", CallbackData: fmt.Sprintf("set_weekday:%d:1", subjectID)},
				{Text: "Вторник", CallbackData: fmt.Sprintf("set_weekday:%d:2", subjectID)},
			},
			{
				{Text: "Среда", CallbackData: fmt.Sprintf("set_weekday:%d:3", subjectID)},
				{Text: "Четверг", CallbackData: fmt.Sprintf("set_weekday:%d:4", subjectID)},
			},
			{
				{Text: "Пятница", CallbackData: fmt.Sprintf("set_weekday:%d:5", subjectID)},
				{Text: "Суббота", CallbackData: fmt.Sprintf("set_weekday:%d:6", subjectID)},
			},
			{
				{Text: "Воскресенье", CallbackData: fmt.Sprintf("set_weekday:%d:0", subjectID)},
			},
			{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("create_slots:%d", subjectID)},
			},
		},
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        "🔄 Постоянное расписание\n\nВыберите день недели:\n\n✨ Слоты будут создаваться автоматически каждую неделю на месяц вперёд",
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// showSingleDaySelection показывает опции для создания слота на один день (7 дней с днями недели)
func showSingleDaySelection(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler, msg *models.Message, subjectID int64) {
	showSingleDaySelectionWithOffset(ctx, b, callback, h, msg, subjectID, 0)
}

// showSingleDaySelectionWithOffset показывает выбор дня со смещением (для пагинации)
func showSingleDaySelectionWithOffset(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler, msg *models.Message, subjectID int64, offset int) {
	now := time.Now()

	weekdayShortNames := map[time.Weekday]string{
		time.Sunday:    "Вс",
		time.Monday:    "Пн",
		time.Tuesday:   "Вт",
		time.Wednesday: "Ср",
		time.Thursday:  "Чт",
		time.Friday:    "Пт",
		time.Saturday:  "Сб",
	}

	var buttons [][]models.InlineKeyboardButton

	// Генерируем кнопки для 7 дней начиная с offset
	for i := 0; i < 7; i++ {
		dayOffset := offset + i
		date := now.AddDate(0, 0, dayOffset)
		dateStr := date.Format("2006-01-02")
		weekdayShort := weekdayShortNames[date.Weekday()]
		displayText := fmt.Sprintf("%s, %s", weekdayShort, date.Format("02.01"))

		// Добавляем специальные метки для сегодня и завтра (только на первой странице)
		if offset == 0 && i == 0 {
			displayText = "Сегодня • " + displayText
		} else if offset == 0 && i == 1 {
			displayText = "Завтра • " + displayText
		}

		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: displayText, CallbackData: fmt.Sprintf("single_day_date:%d:%s", subjectID, dateStr)},
		})
	}

	// Кнопки навигации (вперед/назад)
	var navButtons []models.InlineKeyboardButton

	// Кнопка "назад" только если не на первой странице
	if offset > 0 {
		prevOffset := offset - 7
		if prevOffset < 0 {
			prevOffset = 0
		}
		navButtons = append(navButtons, models.InlineKeyboardButton{
			Text:         "⬅️ Пред. неделя",
			CallbackData: fmt.Sprintf("single_day_page:%d:%d", subjectID, prevOffset),
		})
	}

	// Кнопка "вперед" (показываем до 12 недель вперед)
	if offset < 84 {
		nextOffset := offset + 7
		navButtons = append(navButtons, models.InlineKeyboardButton{
			Text:         "След. неделя ➡️",
			CallbackData: fmt.Sprintf("single_day_page:%d:%d", subjectID, nextOffset),
		})
	}

	if len(navButtons) > 0 {
		buttons = append(buttons, navButtons)
	}

	// Кнопка "Назад к выбору режима"
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "🔙 К выбору режима", CallbackData: fmt.Sprintf("create_slots:%d", subjectID)},
	})

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	weekNum := (offset / 7) + 1
	text := fmt.Sprintf("📆 Создание слота на один день\n\n📍 Неделя %d\nВыберите день:", weekNum)

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// showPeriodSelection показывает выбор периода и дня недели
func showPeriodSelection(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler, msg *models.Message, subjectID int64) {
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "2 недели", CallbackData: fmt.Sprintf("period_weeks:%d:2", subjectID)},
				{Text: "4 недели", CallbackData: fmt.Sprintf("period_weeks:%d:4", subjectID)},
			},
			{
				{Text: "6 недель", CallbackData: fmt.Sprintf("period_weeks:%d:6", subjectID)},
				{Text: "8 недель", CallbackData: fmt.Sprintf("period_weeks:%d:8", subjectID)},
			},
			{
				{Text: "12 недель", CallbackData: fmt.Sprintf("period_weeks:%d:12", subjectID)},
				{Text: "⌨️ Свой период", CallbackData: fmt.Sprintf("custom_period:%d", subjectID)},
			},
			{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("create_slots:%d", subjectID)},
			},
		},
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        "📅 Создание слотов на период\n\nВыберите период:\n(Слоты будут созданы один раз на указанный период, без автоматического повторения)",
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleSingleDayPage обрабатывает переключение страниц при выборе дня
func HandleSingleDayPage(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleSingleDayPage called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: single_day_page:123:7 (subjectID:offset)
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

	offset, err := strconv.Atoi(parts[2])
	if err != nil {
		h.Logger.Error("Failed to parse offset", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверное смещение")
		return
	}

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	showSingleDaySelectionWithOffset(ctx, b, callback, h, msg, subjectID, offset)
}

// HandleSingleDayDate обрабатывает выбор конкретной даты для слота
func HandleSingleDayDate(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleSingleDayDate called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: single_day_date:123:2024-01-15
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
	targetDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		h.Logger.Error("Failed to parse date", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверная дата")
		return
	}

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Получаем предмет для определения длительности
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		h.Logger.Error("Failed to get subject", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}


	// Генерируем временные слоты на основе длительности занятия
	var buttons [][]models.InlineKeyboardButton
	duration := subject.Duration // в минутах

	// Генерируем слоты с 00:00 до 23:59 с шагом в длительность занятия
	currentTime := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	endOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 23, 59, 0, 0, targetDate.Location())

	var row []models.InlineKeyboardButton
	for currentTime.Before(endOfDay) {
		timeStr := currentTime.Format("15:04")
		row = append(row, models.InlineKeyboardButton{
			Text:         timeStr,
			CallbackData: fmt.Sprintf("single_time_auto:%d:%s:%s", subjectID, dateStr, timeStr),
		})

		// По 3 кнопки в ряд для компактности
		if len(row) == 3 {
			buttons = append(buttons, row)
			row = []models.InlineKeyboardButton{}
		}

		currentTime = currentTime.Add(time.Duration(duration) * time.Minute)
	}

	// Добавляем оставшиеся кнопки
	if len(row) > 0 {
		buttons = append(buttons, row)
	}

	// Кнопка для ввода своего времени
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⌨️ Ввести своё время", CallbackData: fmt.Sprintf("custom_time:%d:%s", subjectID, dateStr)},
	})

	// Кнопка "Назад"
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("slot_mode:%d:single", subjectID)},
	})

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	text := fmt.Sprintf("📆 Создание слота на %s, %s\n\n"+
		"⏱ Длительность: %d мин\n\n"+
		"Временные слоты рассчитаны автоматически на основе длительности.\n"+
		"Выберите время начала занятия:",
		targetDate.Format("02.01.2006"),
		formatting.GetWeekdayName(int(targetDate.Weekday())),
		duration)

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandlePeriodWeeks обрабатывает выбор периода для создания слотов
func HandlePeriodWeeks(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandlePeriodWeeks called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: period_weeks:123:4
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

	weeks, err := strconv.Atoi(parts[2])
	if err != nil {
		h.Logger.Error("Failed to parse weeks", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный период")
		return
	}

	// Сохраняем количество недель в state
	telegramID := callback.From.ID
	h.StateManager.SetData(telegramID, "period_weeks", weeks)

	// Показываем выбор дня недели
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Понедельник", CallbackData: fmt.Sprintf("period_weekday:%d:1", subjectID)},
				{Text: "Вторник", CallbackData: fmt.Sprintf("period_weekday:%d:2", subjectID)},
			},
			{
				{Text: "Среда", CallbackData: fmt.Sprintf("period_weekday:%d:3", subjectID)},
				{Text: "Четверг", CallbackData: fmt.Sprintf("period_weekday:%d:4", subjectID)},
			},
			{
				{Text: "Пятница", CallbackData: fmt.Sprintf("period_weekday:%d:5", subjectID)},
				{Text: "Суббота", CallbackData: fmt.Sprintf("period_weekday:%d:6", subjectID)},
			},
			{
				{Text: "Воскресенье", CallbackData: fmt.Sprintf("period_weekday:%d:0", subjectID)},
			},
			{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("create_slots:%d", subjectID)},
			},
		},
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        fmt.Sprintf("📅 Создание слотов на %d %s\n\nВыберите день недели:", weeks, getWeeksWord(weeks)),
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

func getWeeksWord(weeks int) string {
	if weeks == 1 {
		return "неделю"
	}
	if weeks >= 2 && weeks <= 4 {
		return "недели"
	}
	return "недель"
}

// showWorkdaySelection показывает выбор дня недели для автозаполнения рабочего дня
func showWorkdaySelection(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler, msg *models.Message, subjectID int64) {
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Понедельник", CallbackData: fmt.Sprintf("workday_day:%d:1", subjectID)},
				{Text: "Вторник", CallbackData: fmt.Sprintf("workday_day:%d:2", subjectID)},
			},
			{
				{Text: "Среда", CallbackData: fmt.Sprintf("workday_day:%d:3", subjectID)},
				{Text: "Четверг", CallbackData: fmt.Sprintf("workday_day:%d:4", subjectID)},
			},
			{
				{Text: "Пятница", CallbackData: fmt.Sprintf("workday_day:%d:5", subjectID)},
				{Text: "Суббота", CallbackData: fmt.Sprintf("workday_day:%d:6", subjectID)},
			},
			{
				{Text: "Воскресенье", CallbackData: fmt.Sprintf("workday_day:%d:0", subjectID)},
			},
			{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("create_slots:%d", subjectID)},
			},
		},
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        "⚡️ Автозаполнение рабочего дня\n\nВыберите день недели:\n\nБудут созданы слоты с 9:00 до 18:00 с учётом длительности вашего занятия",
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandlePeriodWeekday обрабатывает выбор дня недели для периода
func HandlePeriodWeekday(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandlePeriodWeekday called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: period_weekday:123:1
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

	weekdayNum, err := strconv.Atoi(parts[2])
	if err != nil {
		h.Logger.Error("Failed to parse weekday", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный день")
		return
	}

	// Сохраняем день недели
	telegramID := callback.From.ID
	h.StateManager.SetData(telegramID, "period_weekday", weekdayNum)

	// Показываем выбор времени (перенаправляем на стандартный выбор времени, но с другой логикой)
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}


	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "08:00", CallbackData: fmt.Sprintf("period_time:%d:%d:8", subjectID, weekdayNum)},
				{Text: "09:00", CallbackData: fmt.Sprintf("period_time:%d:%d:9", subjectID, weekdayNum)},
				{Text: "10:00", CallbackData: fmt.Sprintf("period_time:%d:%d:10", subjectID, weekdayNum)},
			},
			{
				{Text: "11:00", CallbackData: fmt.Sprintf("period_time:%d:%d:11", subjectID, weekdayNum)},
				{Text: "12:00", CallbackData: fmt.Sprintf("period_time:%d:%d:12", subjectID, weekdayNum)},
				{Text: "13:00", CallbackData: fmt.Sprintf("period_time:%d:%d:13", subjectID, weekdayNum)},
			},
			{
				{Text: "14:00", CallbackData: fmt.Sprintf("period_time:%d:%d:14", subjectID, weekdayNum)},
				{Text: "15:00", CallbackData: fmt.Sprintf("period_time:%d:%d:15", subjectID, weekdayNum)},
				{Text: "16:00", CallbackData: fmt.Sprintf("period_time:%d:%d:16", subjectID, weekdayNum)},
			},
			{
				{Text: "17:00", CallbackData: fmt.Sprintf("period_time:%d:%d:17", subjectID, weekdayNum)},
				{Text: "18:00", CallbackData: fmt.Sprintf("period_time:%d:%d:18", subjectID, weekdayNum)},
				{Text: "19:00", CallbackData: fmt.Sprintf("period_time:%d:%d:19", subjectID, weekdayNum)},
			},
			{
				{Text: "20:00", CallbackData: fmt.Sprintf("period_time:%d:%d:20", subjectID, weekdayNum)},
			},
			{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("create_slots:%d", subjectID)},
			},
		},
	}

	weeksData, ok := h.StateManager.GetData(telegramID, "period_weeks")
	weeks := 4
	if ok {
		weeks, _ = weeksData.(int)
	}

	text := fmt.Sprintf("📅 Создание слотов на %d %s\n\n"+
		"День недели: %s\n\n"+
		"Выберите время начала занятия:",
		weeks, getWeeksWord(weeks), formatting.GetWeekdayName(int(weekdayNum)))

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}
