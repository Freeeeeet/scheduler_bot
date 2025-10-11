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

// HandlePeriodTime создает слоты на период БЕЗ recurring schedule
func HandlePeriodTime(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandlePeriodTime called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	telegramID := callback.From.ID
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Формат: period_time:123:1:10  (subject_id:weekday:hour)
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 4 {
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

	hour, err := strconv.Atoi(parts[3])
	if err != nil {
		h.Logger.Error("Failed to parse hour", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверное время")
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

	// Получаем количество недель из state
	weeksData, ok := h.StateManager.GetData(telegramID, "period_weeks")
	weeks := 4
	if ok {
		weeks, _ = weeksData.(int)
	}

	// Создаём слоты на указанный период БЕЗ recurring schedule
	now := time.Now()
	location := now.Location()
	weekday := time.Weekday(weekdayNum)

	count := 0
	daysToCheck := weeks * 7

	for i := 0; i < daysToCheck; i++ {
		date := now.AddDate(0, 0, i)

		if date.Weekday() == weekday {
			startTime := time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, location)
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
	}

	h.Logger.Info("Period slots created successfully",
		zap.Int64("telegram_id", telegramID),
		zap.Int64("subject_id", subjectID),
		zap.Int("count", count),
		zap.Int("weeks", weeks))

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

	weeksWord := "недель"
	if weeks == 1 {
		weeksWord = "неделю"
	} else if weeks >= 2 && weeks <= 4 {
		weeksWord = "недели"
	}

	text := fmt.Sprintf("✅ Слоты успешно созданы!\n\n"+
		"📚 Предмет: %s\n"+
		"📅 День: %s\n"+
		"🕐 Время: %02d:00\n"+
		"⏱ Длительность: %d мин\n"+
		"📆 Период: %d %s\n\n"+
		"Создано %d %s\n\n"+
		"Посмотреть расписание: /myschedule",
		subject.Name,
		weekdayNames[weekdayNum],
		hour,
		subject.Duration,
		weeks, weeksWord,
		count, getSlotsWord(count))

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
		Text:      text,
	})

	common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Слоты созданы!")
}

func getSlotsWord(count int) string {
	if count%10 == 1 && count%100 != 11 {
		return "слот"
	}
	if count%10 >= 2 && count%10 <= 4 && (count%100 < 10 || count%100 >= 20) {
		return "слота"
	}
	return "слотов"
}
