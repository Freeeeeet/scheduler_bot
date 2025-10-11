package student

import (
	"context"
	"fmt"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// ========================
// Student Booking Handlers
// ========================

// HandleBookLesson начинает процесс бронирования урока
func HandleBookLesson(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	slotID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат данных")
		return
	}

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка получения данных пользователя")
		return
	}

	// Бронируем слот
	booking, err := h.BookingService.BookSlot(ctx, user.ID, slotID)
	if err != nil {
		h.Logger.Error("Failed to book slot",
			zap.Error(err),
			zap.Int64("user_id", user.ID),
			zap.Int64("slot_id", slotID),
		)

		errorMsg := "❌ Не удалось забронировать слот."
		if err.Error() == "slot is not available" {
			errorMsg = "❌ Этот слот уже занят. Выберите другое время."
		} else if err.Error() == "slot is in the past" {
			errorMsg = "❌ Этот слот в прошлом. Выберите другое время."
		} else if err.Error() == "subject is not active" {
			errorMsg = "❌ Этот предмет больше не доступен для записи."
		}

		common.AnswerCallbackAlert(ctx, b, callback.ID, errorMsg)
		return
	}

	b.DeleteMessage(ctx, &bot.DeleteMessageParams{ChatID: msg.Chat.ID, MessageID: msg.ID})

	// Определяем текст статуса в зависимости от фактического статуса
	statusText := "Подтверждена ✅"
	additionalInfo := "Учитель получил уведомление о вашей записи."

	if booking.Status == model.BookingStatusPending {
		statusText = "Ожидает одобрения ⏳"
		additionalInfo = "Учитель получил запрос на одобрение.\nВы получите уведомление после проверки."
	}

	text := fmt.Sprintf(
		"✅ Запись успешно создана!\n\n"+
			"📝 Запись #%d\n"+
			"📅 Статус: %s\n"+
			"📍 ID слота: %d\n\n"+
			"%s\n"+
			"Детали занятия будут доступны в /mybookings",
		booking.ID,
		statusText,
		slotID,
		additionalInfo,
	)

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "📅 Мои записи", CallbackData: "back_to_main"}},
			{{Text: "➕ Записаться ещё", CallbackData: "book_another"}},
		},
	}

	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: text, ReplyMarkup: keyboard})
	common.AnswerCallback(ctx, b, callback.ID, "✅ Запись создана")

	// Отправляем уведомление учителю
	teacher, err := h.UserService.GetByID(ctx, booking.TeacherID)
	if err == nil && teacher != nil && booking.Subject != nil && booking.Slot != nil {
		var notificationText string

		if booking.Status == model.BookingStatusPending {
			// Для pending - запрос на одобрение с кнопками
			notificationText = fmt.Sprintf(
				"⏳ **Новый запрос на запись**\n\n"+
					"👤 Студент: %s\n"+
					"📚 Предмет: %s\n"+
					"📅 Дата: %s\n"+
					"🕐 Время: %s - %s\n\n"+
					"Требуется ваше одобрение:",
				user.FirstName,
				booking.Subject.Name,
				booking.Slot.StartTime.Format("02.01.2006"),
				booking.Slot.StartTime.Format("15:04"),
				booking.Slot.EndTime.Format("15:04"),
			)

			keyboard := &models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{
						{Text: "✅ Одобрить", CallbackData: fmt.Sprintf("approve_booking:%d", booking.ID)},
						{Text: "❌ Отклонить", CallbackData: fmt.Sprintf("reject_booking:%d", booking.ID)},
					},
				},
			}

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      teacher.TelegramID,
				Text:        notificationText,
				ParseMode:   models.ParseModeMarkdown,
				ReplyMarkup: keyboard,
			})
		} else {
			// Для confirmed - просто уведомление
			notificationText = fmt.Sprintf(
				"✅ **Новая запись**\n\n"+
					"👤 Студент: %s\n"+
					"📚 Предмет: %s\n"+
					"📅 Дата: %s\n"+
					"🕐 Время: %s - %s\n\n"+
					"Запись подтверждена автоматически.",
				user.FirstName,
				booking.Subject.Name,
				booking.Slot.StartTime.Format("02.01.2006"),
				booking.Slot.StartTime.Format("15:04"),
				booking.Slot.EndTime.Format("15:04"),
			)

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    teacher.TelegramID,
				Text:      notificationText,
				ParseMode: models.ParseModeMarkdown,
			})
		}
	}
}

// HandleCancelBooking начинает процесс отмены бронирования
func HandleCancelBooking(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	bookingID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат данных")
		return
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "✅ Да, отменить", CallbackData: fmt.Sprintf("confirm_cancel:%d", bookingID)},
				{Text: "❌ Нет", CallbackData: "back_to_main"},
			},
		},
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        fmt.Sprintf("❓ Вы уверены, что хотите отменить запись #%d?\n\nУчитель получит уведомление об отмене.", bookingID),
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "Подтверждение отмены")
}

// HandleConfirmCancel подтверждает отмену бронирования
func HandleConfirmCancel(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	bookingID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат данных")
		return
	}

	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка получения данных пользователя")
		return
	}

	err = h.BookingService.CancelBooking(ctx, bookingID, user.ID)
	if err != nil {
		h.Logger.Error("Failed to cancel booking", zap.Error(err))

		errorMsg := "❌ Не удалось отменить запись"
		if err.Error() == "booking not found" {
			errorMsg = "❌ Запись не найдена"
		} else if err.Error() == "no permission to cancel this booking" {
			errorMsg = "❌ У вас нет прав для отмены этой записи"
		} else if err.Error() == "booking is not active" {
			errorMsg = "❌ Эта запись уже отменена или завершена"
		}

		common.AnswerCallbackAlert(ctx, b, callback.ID, errorMsg)
		return
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "📅 Мои записи", CallbackData: "back_to_main"}},
			{{Text: "➕ Записаться на другое занятие", CallbackData: "book_another"}},
		},
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        fmt.Sprintf("✅ Запись #%d успешно отменена.\n\nУчитель получил уведомление.", bookingID),
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "✅ Запись отменена")
}
