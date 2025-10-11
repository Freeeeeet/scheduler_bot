package teacher

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
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

// showSingleDaySelection показывает опции для создания слота на один день
func showSingleDaySelection(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler, msg *models.Message, subjectID int64) {
	// Пока упрощенный вариант - тоже через день недели, но только 1 слот
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Сегодня", CallbackData: fmt.Sprintf("single_day:%d:today", subjectID)},
				{Text: "Завтра", CallbackData: fmt.Sprintf("single_day:%d:tomorrow", subjectID)},
			},
			{
				{Text: "Послезавтра", CallbackData: fmt.Sprintf("single_day:%d:dayafter", subjectID)},
			},
			{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("create_slots:%d", subjectID)},
			},
		},
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        "📆 Создание слота на один день\n\nВыберите день:",
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

// HandleSingleDay обрабатывает создание одного слота на конкретный день
func HandleSingleDay(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleSingleDay called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: single_day:123:today
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

	dayOption := parts[2]

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Вычисляем дату
	now := time.Now()
	var targetDate time.Time

	switch dayOption {
	case "today":
		targetDate = now
	case "tomorrow":
		targetDate = now.AddDate(0, 0, 1)
	case "dayafter":
		targetDate = now.AddDate(0, 0, 2)
	default:
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверная опция")
		return
	}

	dateStr := targetDate.Format("2006-01-02")
	weekdayNames := map[time.Weekday]string{
		time.Sunday:    "воскресенье",
		time.Monday:    "понедельник",
		time.Tuesday:   "вторник",
		time.Wednesday: "среду",
		time.Thursday:  "четверг",
		time.Friday:    "пятницу",
		time.Saturday:  "субботу",
	}

	// Показываем выбор времени
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "08:00", CallbackData: fmt.Sprintf("single_time:%d:%s:8", subjectID, dateStr)},
				{Text: "09:00", CallbackData: fmt.Sprintf("single_time:%d:%s:9", subjectID, dateStr)},
				{Text: "10:00", CallbackData: fmt.Sprintf("single_time:%d:%s:10", subjectID, dateStr)},
			},
			{
				{Text: "11:00", CallbackData: fmt.Sprintf("single_time:%d:%s:11", subjectID, dateStr)},
				{Text: "12:00", CallbackData: fmt.Sprintf("single_time:%d:%s:12", subjectID, dateStr)},
				{Text: "13:00", CallbackData: fmt.Sprintf("single_time:%d:%s:13", subjectID, dateStr)},
			},
			{
				{Text: "14:00", CallbackData: fmt.Sprintf("single_time:%d:%s:14", subjectID, dateStr)},
				{Text: "15:00", CallbackData: fmt.Sprintf("single_time:%d:%s:15", subjectID, dateStr)},
				{Text: "16:00", CallbackData: fmt.Sprintf("single_time:%d:%s:16", subjectID, dateStr)},
			},
			{
				{Text: "17:00", CallbackData: fmt.Sprintf("single_time:%d:%s:17", subjectID, dateStr)},
				{Text: "18:00", CallbackData: fmt.Sprintf("single_time:%d:%s:18", subjectID, dateStr)},
				{Text: "19:00", CallbackData: fmt.Sprintf("single_time:%d:%s:19", subjectID, dateStr)},
			},
			{
				{Text: "20:00", CallbackData: fmt.Sprintf("single_time:%d:%s:20", subjectID, dateStr)},
			},
			{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("create_slots:%d", subjectID)},
			},
		},
	}

	text := fmt.Sprintf("📆 Создание слота на один день\n\n"+
		"Выбран день: %s, %s\n\n"+
		"Выберите время начала занятия:",
		targetDate.Format("02.01.2006"),
		weekdayNames[targetDate.Weekday()])

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

	weekdayNames := map[int]string{
		0: "Воскресенье",
		1: "Понедельник",
		2: "Вторник",
		3: "Среда",
		4: "Четверг",
		5: "Пятница",
		6: "Суббота",
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
				{Text: "❌ Отмена", CallbackData: "back_to_main"},
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
		weeks, getWeeksWord(weeks), weekdayNames[weekdayNum])

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}
