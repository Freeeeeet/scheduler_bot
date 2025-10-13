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

// HandleWorkdayDay показывает выбор времени для рабочего дня
func HandleWorkdayDay(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleWorkdayDay called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	telegramID := callback.From.ID
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Формат: workday_day:123:1  (subject_id:weekday)
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

	// Сохраняем день недели в state
	h.StateManager.SetData(telegramID, "workday_weekday", weekdayNum)


	// Показываем выбор времени начала рабочего дня
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "08:00", CallbackData: fmt.Sprintf("workday_start:%d:8", subjectID)},
				{Text: "09:00", CallbackData: fmt.Sprintf("workday_start:%d:9", subjectID)},
				{Text: "10:00", CallbackData: fmt.Sprintf("workday_start:%d:10", subjectID)},
			},
			{
				{Text: "11:00", CallbackData: fmt.Sprintf("workday_start:%d:11", subjectID)},
				{Text: "12:00", CallbackData: fmt.Sprintf("workday_start:%d:12", subjectID)},
				{Text: "13:00", CallbackData: fmt.Sprintf("workday_start:%d:13", subjectID)},
			},
			{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("slot_mode:%d:workday", subjectID)},
			},
		},
	}

	text := fmt.Sprintf("⚡️ Автозаполнение рабочего дня\n\n"+
		"📅 День: %s\n\n"+
		"Выберите время <b>начала</b> рабочего дня:",
		formatting.GetWeekdayName(int(weekdayNum)))

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleWorkdayStart обрабатывает выбор времени начала рабочего дня
func HandleWorkdayStart(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleWorkdayStart called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	telegramID := callback.From.ID
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Формат: workday_start:123:9  (subject_id:start_hour)
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

	startHour, err := strconv.Atoi(parts[2])
	if err != nil {
		h.Logger.Error("Failed to parse start hour", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверное время")
		return
	}

	// Сохраняем время начала в state
	h.StateManager.SetData(telegramID, "workday_start", startHour)

	// Получаем день недели из state
	weekdayData, ok := h.StateManager.GetData(telegramID, "workday_weekday")
	if !ok {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка: потеряны данные")
		return
	}
	weekdayNum, _ := weekdayData.(int)


	// Показываем выбор времени окончания (от startHour+1 до 22:00)
	var buttons [][]models.InlineKeyboardButton
	var row []models.InlineKeyboardButton

	for hour := startHour + 1; hour <= 22; hour++ {
		row = append(row, models.InlineKeyboardButton{
			Text:         fmt.Sprintf("%02d:00", hour),
			CallbackData: fmt.Sprintf("workday_end:%d:%d", subjectID, hour),
		})

		if len(row) == 3 {
			buttons = append(buttons, row)
			row = []models.InlineKeyboardButton{}
		}
	}

	if len(row) > 0 {
		buttons = append(buttons, row)
	}

	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("workday_day:%d:%d", subjectID, weekdayNum)},
	})

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	text := fmt.Sprintf("⚡️ Автозаполнение рабочего дня\n\n"+
		"📅 День: %s\n"+
		"🕐 Начало: %02d:00\n\n"+
		"Выберите время <b>окончания</b> рабочего дня:",
		formatting.GetWeekdayName(int(weekdayNum)),
		startHour)

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleWorkdayEnd создает слоты для рабочего дня
func HandleWorkdayEnd(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleWorkdayEnd called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	telegramID := callback.From.ID
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Формат: workday_end:123:18  (subject_id:end_hour)
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

	endHour, err := strconv.Atoi(parts[2])
	if err != nil {
		h.Logger.Error("Failed to parse end hour", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверное время")
		return
	}

	// Получаем данные из state
	weekdayData, ok := h.StateManager.GetData(telegramID, "workday_weekday")
	if !ok {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка: потеряны данные")
		return
	}
	weekdayNum, _ := weekdayData.(int)

	startHourData, ok := h.StateManager.GetData(telegramID, "workday_start")
	if !ok {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка: потеряны данные")
		return
	}
	startHour, _ := startHourData.(int)

	// Получаем пользователя
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		h.Logger.Error("Failed to get user",
			zap.Error(err),
			zap.Int64("telegram_id", telegramID))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil {
		h.Logger.Error("Failed to get subject",
			zap.Error(err),
			zap.Int64("subject_id", subjectID))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	if subject.TeacherID != user.ID {
		h.Logger.Error("Subject does not belong to teacher",
			zap.Int64("subject_id", subjectID),
			zap.Int64("teacher_id", user.ID))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Это не ваш предмет")
		return
	}

	// Автозаполнение: создаём слоты между startHour и endHour
	now := time.Now()
	location := now.Location()
	weekday := time.Weekday(weekdayNum)

	// Ищем следующий день с нужным днём недели
	daysUntilTarget := (int(weekday) - int(now.Weekday()) + 7) % 7
	if daysUntilTarget == 0 && now.Hour() >= endHour {
		daysUntilTarget = 7 // Если сегодня этот день, но уже поздно, берём следующую неделю
	}
	targetDate := now.AddDate(0, 0, daysUntilTarget)

	// Рассчитываем сколько слотов помещается в рабочий день
	workdayMinutes := (endHour - startHour) * 60
	slotsCount := workdayMinutes / subject.Duration

	count := 0

	for i := 0; i < slotsCount; i++ {
		// Вычисляем время начала слота
		minutesFromStart := i * subject.Duration
		slotStartHour := startHour + (minutesFromStart / 60)
		slotStartMinute := minutesFromStart % 60

		// Проверяем, что слот заканчивается до endHour
		slotEndMinutes := minutesFromStart + subject.Duration
		slotEndHour := startHour + (slotEndMinutes / 60)

		if slotEndHour > endHour {
			break // Слот выходит за пределы рабочего дня
		}

		startTime := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(),
			slotStartHour, slotStartMinute, 0, 0, location)
		endTime := startTime.Add(time.Duration(subject.Duration) * time.Minute)

		// Пропускаем прошедшие слоты
		if startTime.Before(now) {
			continue
		}

		_, err = h.TeacherService.CreateSlot(ctx, user.ID, subjectID, startTime, endTime)
		if err != nil {
			h.Logger.Warn("Failed to create slot",
				zap.Error(err),
				zap.Time("start_time", startTime),
			)
			continue
		}

		count++
	}

	h.Logger.Info("Workday slots created successfully",
		zap.Int64("telegram_id", telegramID),
		zap.Int64("subject_id", subjectID),
		zap.Int("count", count))

	// Очищаем state
	h.StateManager.ClearState(telegramID)


	slotsWord := "слотов"
	if count%10 == 1 && count%100 != 11 {
		slotsWord = "слот"
	} else if count%10 >= 2 && count%10 <= 4 && (count%100 < 10 || count%100 >= 20) {
		slotsWord = "слота"
	}

	text := fmt.Sprintf("✅ Рабочий день заполнен!\n\n"+
		"📚 Предмет: %s\n"+
		"📅 День: %s\n"+
		"🕐 Рабочее время: %02d:00 - %02d:00\n"+
		"⏱ Длительность занятия: %d мин\n\n"+
		"Создано %d %s\n\n"+
		"Посмотреть расписание: /myschedule",
		subject.Name,
		formatting.GetWeekdayName(int(weekdayNum)),
		startHour,
		endHour,
		subject.Duration,
		count, slotsWord)

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
		Text:      text,
	})

	common.AnswerCallbackAlert(ctx, b, callback.ID, fmt.Sprintf("✅ Создано %d %s!", count, slotsWord))
}
