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

// HandleWorkdayDay обрабатывает автозаполнение рабочего дня
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

	// Автозаполнение: создаём слоты с 9:00 до 18:00
	now := time.Now()
	location := now.Location()
	weekday := time.Weekday(weekdayNum)

	// Ищем следующий день с нужным днём недели
	daysUntilTarget := (int(weekday) - int(now.Weekday()) + 7) % 7
	if daysUntilTarget == 0 && now.Hour() >= 18 {
		daysUntilTarget = 7 // Если сегодня этот день, но уже поздно, берём следующую неделю
	}
	targetDate := now.AddDate(0, 0, daysUntilTarget)

	// Рассчитываем сколько слотов помещается в рабочий день (9:00 - 18:00)
	workdayMinutes := 9 * 60 // 9 часов * 60 минут
	slotsCount := workdayMinutes / subject.Duration
	
	count := 0
	startHour := 9

	for i := 0; i < slotsCount; i++ {
		// Вычисляем время начала слота
		minutesFromStart := i * subject.Duration
		slotStartHour := startHour + (minutesFromStart / 60)
		slotStartMinute := minutesFromStart % 60

		// Проверяем, что слот заканчивается до 18:00
		slotEndMinutes := minutesFromStart + subject.Duration
		slotEndHour := startHour + (slotEndMinutes / 60)
		
		if slotEndHour > 18 {
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

	weekdayNames := map[int]string{
		0: "Воскресенье",
		1: "Понедельник",
		2: "Вторник",
		3: "Среда",
		4: "Четверг",
		5: "Пятница",
		6: "Суббота",
	}

	slotsWord := "слотов"
	if count%10 == 1 && count%100 != 11 {
		slotsWord = "слот"
	} else if count%10 >= 2 && count%10 <= 4 && (count%100 < 10 || count%100 >= 20) {
		slotsWord = "слота"
	}

	text := fmt.Sprintf("✅ Рабочий день заполнен!\n\n"+
		"📚 Предмет: %s\n"+
		"📅 День: %s\n"+
		"🕐 Рабочее время: 9:00 - 18:00\n"+
		"⏱ Длительность занятия: %d мин\n\n"+
		"Создано %d %s\n\n"+
		"Посмотреть расписание: /myschedule",
		subject.Name,
		weekdayNames[weekdayNum],
		subject.Duration,
		count, slotsWord)

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
		Text:      text,
	})

	common.AnswerCallbackAlert(ctx, b, callback.ID, fmt.Sprintf("✅ Создано %d %s!", count, slotsWord))
}

