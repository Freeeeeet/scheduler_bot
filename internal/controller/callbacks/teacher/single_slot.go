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

// HandleSingleDayTime обрабатывает выбор времени для разового слота
func HandleSingleDayTime(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleSingleDayTime called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	telegramID := callback.From.ID
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Формат: single_time:123:2025-01-15:10  (subject_id:date:hour)
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

	dateStr := parts[2]
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		h.Logger.Error("Failed to parse date", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверная дата")
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

	// Создаём один слот
	location := time.Now().Location()
	startTime := time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, location)
	endTime := startTime.Add(time.Duration(subject.Duration) * time.Minute)

	// Проверяем, что слот не в прошлом
	if startTime.Before(time.Now()) {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Нельзя создать слот в прошлом")
		return
	}

	_, err = h.TeacherService.CreateSlot(ctx, user.ID, subjectID, startTime, endTime)
	if err != nil {
		h.Logger.Error("Failed to create slot",
			zap.Error(err),
			zap.Time("start_time", startTime))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось создать слот")
		return
	}

	h.Logger.Info("Single slot created successfully",
		zap.Int64("telegram_id", telegramID),
		zap.Int64("subject_id", subjectID),
		zap.Time("start_time", startTime))

	// Очищаем state
	h.StateManager.ClearState(telegramID)

	weekdayNames := map[time.Weekday]string{
		time.Sunday:    "Воскресенье",
		time.Monday:    "Понедельник",
		time.Tuesday:   "Вторник",
		time.Wednesday: "Среда",
		time.Thursday:  "Четверг",
		time.Friday:    "Пятница",
		time.Saturday:  "Суббота",
	}

	text := fmt.Sprintf("✅ Слот успешно создан!\n\n"+
		"📚 Предмет: %s\n"+
		"📅 Дата: %s, %s\n"+
		"🕐 Время: %s - %s\n"+
		"⏱ Длительность: %d мин\n\n"+
		"Посмотреть расписание: /myschedule",
		subject.Name,
		date.Format("02.01.2006"),
		weekdayNames[date.Weekday()],
		startTime.Format("15:04"),
		endTime.Format("15:04"),
		subject.Duration)

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
		Text:      text,
	})

	common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Слот создан!")
}

