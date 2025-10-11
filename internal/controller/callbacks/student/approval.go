package student

import (
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"context"
	"fmt"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// ========================
// Booking Approval System Handlers (for teachers)
// ========================

// HandleApproveBooking одобряет запрос на бронирование
func HandleApproveBooking(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	bookingID, err := common.ParseIDFromCallback(callback.Data)
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

	// Одобряем бронирование
	err = h.BookingService.ApproveBooking(ctx, bookingID, user.ID)
	if err != nil {
		h.Logger.Error("Failed to approve booking", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось одобрить запись")
		return
	}

	// Получаем детали бронирования для уведомления
	booking, _ := h.BookingService.GetByID(ctx, bookingID)
	if booking != nil {
		student, _ := h.UserService.GetByID(ctx, booking.StudentID)
		if student != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: student.TelegramID,
				Text: fmt.Sprintf(
					"✅ **Запись одобрена!**\n\n"+
						"Ваша запись #%d была одобрена учителем.\n"+
						"Занятие подтверждено!",
					bookingID,
				),
				ParseMode: models.ParseModeMarkdown,
			})
		}
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Запись одобрена")

	// Обновляем сообщение
	msg := common.GetMessageFromCallback(callback)
	if msg != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    msg.Chat.ID,
			MessageID: msg.ID,
			Text:      fmt.Sprintf("✅ Запись #%d одобрена", bookingID),
		})
	}
}

// HandleRejectBooking отклоняет запрос на бронирование
func HandleRejectBooking(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	bookingID, err := common.ParseIDFromCallback(callback.Data)
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

	// Отклоняем бронирование
	err = h.BookingService.RejectBooking(ctx, bookingID, user.ID)
	if err != nil {
		h.Logger.Error("Failed to reject booking", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось отклонить запись")
		return
	}

	// Получаем детали бронирования для уведомления
	booking, _ := h.BookingService.GetByID(ctx, bookingID)
	if booking != nil {
		student, _ := h.UserService.GetByID(ctx, booking.StudentID)
		if student != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: student.TelegramID,
				Text: fmt.Sprintf(
					"❌ **Запись отклонена**\n\n"+
						"К сожалению, ваша запись #%d была отклонена учителем.\n"+
						"Попробуйте выбрать другое время.",
					bookingID,
				),
				ParseMode: models.ParseModeMarkdown,
			})
		}
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Запись отклонена")

	// Обновляем сообщение
	msg := common.GetMessageFromCallback(callback)
	if msg != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    msg.Chat.ID,
			MessageID: msg.ID,
			Text:      fmt.Sprintf("❌ Запись #%d отклонена", bookingID),
		})
	}
}

// HandleApproveCancel одобряет запрос студента на отмену
func HandleApproveCancel(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	bookingID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка получения данных пользователя")
		return
	}

	if !user.IsTeacher {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Только учителя могут одобрять отмены")
		return
	}

	// Отменяем бронирование от имени учителя
	err = h.BookingService.CancelBooking(ctx, bookingID, user.ID)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось одобрить отмену")
		return
	}

	text := fmt.Sprintf(
		"✅ Отмена одобрена\n\n"+
			"Запись #%d успешно отменена.\n"+
			"Слот снова доступен для бронирования.\n"+
			"Студент получил уведомление.",
		bookingID,
	)

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "⬅️ К расписанию", CallbackData: "view_schedule"}},
		},
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "✅ Отмена одобрена")
}

// HandleRejectCancel отклоняет запрос студента на отмену
func HandleRejectCancel(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	bookingID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}
	// TODO: Реализовать отклонение отмены (убрать флаг cancellation_requested)
	common.AnswerCallback(ctx, b, callback.ID, fmt.Sprintf("🚧 Отклонение отмены #%d (в разработке)", bookingID))
}
