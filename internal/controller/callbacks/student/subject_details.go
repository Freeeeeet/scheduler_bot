package student

import (
	"context"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// HandleViewSubjectDetails показывает детали предмета для студента
func HandleViewSubjectDetails(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleViewSubjectDetails called",
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

	// Получаем предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		h.Logger.Error("Subject not found",
			zap.Int64("subject_id", subjectID),
			zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	// Получаем учителя
	teacher, err := h.UserService.GetByID(ctx, subject.TeacherID)
	if err != nil {
		h.Logger.Error("Teacher not found",
			zap.Int64("teacher_id", subject.TeacherID),
			zap.Error(err))
	}

	teacherName := "Неизвестный преподаватель"
	if teacher != nil {
		teacherName = teacher.FirstName
		if teacher.LastName != "" {
			teacherName += " " + teacher.LastName
		}
	}

	// Используем билдер экрана
	text, keyboard := common.BuildStudentSubjectDetailsScreen(subject, teacherName)

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}
