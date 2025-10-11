package teacher

import (
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"context"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// ========================
// Teacher Onboarding Handlers
// ========================
// These handlers manage the process of becoming a teacher

// HandleBecomeTeacherConfirm обрабатывает подтверждение стать учителем
func HandleBecomeTeacherConfirm(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	telegramID := callback.From.ID

	// Делаем пользователя учителем
	err := h.UserService.MakeTeacher(ctx, telegramID)
	if err != nil {
		h.Logger.Error("Failed to make teacher", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Произошла ошибка. Попробуйте позже.")
		return
	}

	// Удаляем старое сообщение
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
	})

	// Отправляем новое сообщение с предложением создать предмет
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "➕ Создать первый предмет", CallbackData: "create_first_subject"},
			},
			{
				{Text: "⏭ Пропустить", CallbackData: "skip_first_subject"},
			},
		},
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text: "🎓 Поздравляем! Теперь вы учитель!\n\n" +
			"Вы можете:\n" +
			"• Создавать предметы\n" +
			"• Управлять расписанием\n" +
			"• Принимать записи от студентов\n\n" +
			"Хотите создать свой первый предмет прямо сейчас?",
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "✅ Вы стали учителем!")
}

// HandleBecomeTeacherCancel обрабатывает отмену становления учителем
func HandleBecomeTeacherCancel(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Удаляем сообщение
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
	})

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   "✅ Операция отменена.\n\nВы всегда можете стать учителем позже через /becometeacher",
	})

	common.AnswerCallback(ctx, b, callback.ID, "Отменено")
}
