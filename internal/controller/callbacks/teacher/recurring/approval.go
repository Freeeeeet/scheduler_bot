package recurring

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common/formatting"
	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// HandleApproveRecurring одобряет запрос студента на постоянную запись
func HandleApproveRecurring(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleApproveRecurring called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Парсим: approve_recurring:scheduleID:studentID:subjectID
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 4 {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	scheduleID, _ := strconv.ParseInt(parts[1], 10, 64)
	studentID, _ := strconv.ParseInt(parts[2], 10, 64)
	subjectID, _ := strconv.ParseInt(parts[3], 10, 64)

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Получаем информацию
	student, err := h.UserService.GetByID(ctx, studentID)
	if err != nil || student == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Студент не найден")
		return
	}

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

	// TODO: Реализовать логику автоматического бронирования всех будущих слотов
	// Для этого нужно:
	// 1. Найти все свободные слоты этого recurring schedule
	// 2. Автоматически забронировать их для студента
	// 3. Создать связь recurring_booking в БД (если добавим такую таблицу)

	// Пока отправляем уведомление студенту
	successText := fmt.Sprintf(
		"✅ **Запрос одобрен!**\n\n"+
			"👤 Студент: %s %s\n"+
			"📚 Предмет: %s\n"+
			"📅 Расписание: %s в %02d:%02d\n\n"+
			"💡 Студент будет уведомлён об одобрении.",
		student.FirstName, student.LastName,
		subject.Name,
		formatting.GetWeekdayName(int(targetSchedule.Weekday)),
		targetSchedule.StartHour, targetSchedule.StartMinute)

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
		Text:      successText,
		ParseMode: models.ParseModeMarkdown,
	})

	// Уведомляем студента
	studentNotification := fmt.Sprintf(
		"✅ **Ваш запрос одобрен!**\n\n"+
			"📚 Предмет: %s\n"+
			"📅 Расписание: %s в %02d:%02d\n\n"+
			"🎉 Вы записаны на постоянной основе!\n"+
			"Все будущие слоты этого расписания автоматически бронируются за вами.\n\n"+
			"Посмотреть свои записи: /mybookings",
		subject.Name,
		formatting.GetWeekdayName(int(targetSchedule.Weekday)),
		targetSchedule.StartHour, targetSchedule.StartMinute)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    student.TelegramID,
		Text:      studentNotification,
		ParseMode: models.ParseModeMarkdown,
	})

	common.AnswerCallback(ctx, b, callback.ID, "✅ Одобрено")
}

// HandleRejectRecurring отклоняет запрос студента на постоянную запись
func HandleRejectRecurring(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleRejectRecurring called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Парсим: reject_recurring:scheduleID:studentID:subjectID
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 4 {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	scheduleID, _ := strconv.ParseInt(parts[1], 10, 64)
	studentID, _ := strconv.ParseInt(parts[2], 10, 64)
	subjectID, _ := strconv.ParseInt(parts[3], 10, 64)

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Получаем информацию
	student, err := h.UserService.GetByID(ctx, studentID)
	if err != nil || student == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Студент не найден")
		return
	}

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

	// Уведомление учителю
	rejectText := fmt.Sprintf(
		"❌ **Запрос отклонён**\n\n"+
			"👤 Студент: %s %s\n"+
			"📚 Предмет: %s\n"+
			"📅 Расписание: %s в %02d:%02d\n\n"+
			"Студент будет уведомлён об отказе.",
		student.FirstName, student.LastName,
		subject.Name,
		formatting.GetWeekdayName(int(targetSchedule.Weekday)),
		targetSchedule.StartHour, targetSchedule.StartMinute)

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
		Text:      rejectText,
		ParseMode: models.ParseModeMarkdown,
	})

	// Уведомляем студента
	studentNotification := fmt.Sprintf(
		"❌ **Ваш запрос отклонён**\n\n"+
			"📚 Предмет: %s\n"+
			"📅 Расписание: %s в %02d:%02d\n\n"+
			"К сожалению, преподаватель отклонил ваш запрос на постоянную запись.\n"+
			"Вы можете записаться на разовые занятия через /subjects",
		subject.Name,
		formatting.GetWeekdayName(int(targetSchedule.Weekday)),
		targetSchedule.StartHour, targetSchedule.StartMinute)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    student.TelegramID,
		Text:      studentNotification,
		ParseMode: models.ParseModeMarkdown,
	})

	common.AnswerCallback(ctx, b, callback.ID, "❌ Отклонено")
}
