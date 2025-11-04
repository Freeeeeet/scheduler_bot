package subjects

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// ========================
// Subject Management Handlers
// ========================

// HandleCreateFirstSubject начинает создание первого предмета
func HandleCreateFirstSubject(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
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

	b.DeleteMessage(ctx, &bot.DeleteMessageParams{ChatID: msg.Chat.ID, MessageID: msg.ID})

	h.StateManager.SetState(telegramID, "create_subject_name")
	h.StateManager.SetData(telegramID, "teacher_id", user.ID)

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{{
			{Text: "⬅️ Назад", CallbackData: "back_to_subjects"},
		}},
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text: "📝 Создание нового предмета\n\n" +
			"Шаг 1 из 4: Как будет называться предмет?\n\n" +
			"Например: Математика, Английский язык, Программирование",
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "Создаём предмет")
}

// HandleSkipFirstSubject пропускает создание первого предмета
func HandleSkipFirstSubject(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	b.DeleteMessage(ctx, &bot.DeleteMessageParams{ChatID: msg.Chat.ID, MessageID: msg.ID})

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text: "✅ Хорошо!\n\n" +
			"Вы можете создать предмет позже через:\n" +
			"/mysubjects → Создать предмет\n\n" +
			"Или используйте /help для просмотра всех команд.",
	})

	common.AnswerCallback(ctx, b, callback.ID, "Пропущено")
}

// HandleCreateSubjectApprovalYes финализирует создание предмета с одобрением
func HandleCreateSubjectApprovalYes(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	finalizeSubjectCreation(ctx, b, callback, h, true)
}

// HandleCreateSubjectApprovalNo финализирует создание предмета без одобрения
func HandleCreateSubjectApprovalNo(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	finalizeSubjectCreation(ctx, b, callback, h, false)
}

// finalizeSubjectCreation завершает создание предмета
func finalizeSubjectCreation(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler, requiresApproval bool) {
	telegramID := callback.From.ID

	h.Logger.Info("Finalizing subject creation",
		zap.Int64("telegram_id", telegramID),
		zap.Bool("requires_approval", requiresApproval))

	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		h.Logger.Error("User not found in finalization",
			zap.Int64("telegram_id", telegramID),
			zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем все данные
	allData := h.StateManager.GetAllData(telegramID)
	h.Logger.Info("Retrieved all data from state",
		zap.Int64("telegram_id", telegramID),
		zap.Any("data", allData))

	teacherID, okTeacher := allData["teacher_id"].(int64)
	name, okName := allData["name"].(string)
	description, okDesc := allData["description"].(string)
	price, okPrice := allData["price"].(int)
	duration, okDuration := allData["duration"].(int)

	h.Logger.Info("Data type assertions",
		zap.Bool("teacher_id_ok", okTeacher),
		zap.Bool("name_ok", okName),
		zap.Bool("description_ok", okDesc),
		zap.Bool("price_ok", okPrice),
		zap.Bool("duration_ok", okDuration))

	if !okTeacher || !okName || !okDesc || !okPrice || !okDuration {
		h.Logger.Error("Missing or invalid data for subject creation",
			zap.Int64("telegram_id", telegramID),
			zap.Bool("teacher_id_ok", okTeacher),
			zap.Bool("name_ok", okName),
			zap.Bool("description_ok", okDesc),
			zap.Bool("price_ok", okPrice),
			zap.Bool("duration_ok", okDuration))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Данные предмета не найдены. Попробуйте создать предмет заново.")
		h.StateManager.ClearState(telegramID)
		return
	}

	h.Logger.Info("Creating subject with data",
		zap.Int64("teacher_id", teacherID),
		zap.String("name", name),
		zap.String("description", description),
		zap.Int("price", price),
		zap.Int("duration", duration))

	// Создаём предмет
	subject, err := h.TeacherService.CreateSubject(ctx, teacherID, name, description, price, duration, requiresApproval)
	if err != nil {
		h.Logger.Error("Failed to create subject",
			zap.Error(err),
			zap.Int64("teacher_id", teacherID))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось создать предмет")
		return
	}

	h.Logger.Info("Subject created successfully",
		zap.Int64("subject_id", subject.ID),
		zap.String("name", subject.Name))

	// Очищаем состояние
	h.StateManager.ClearState(telegramID)

	priceInRubles := float64(price) / 100
	approvalText := "❌ Нет"
	if requiresApproval {
		approvalText = "✅ Да"
	}

	msg := common.GetMessageFromCallback(callback)
	if msg != nil {
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    msg.Chat.ID,
			MessageID: msg.ID,
		})
	}

	chatID := callback.From.ID
	if callback.Message.Message != nil {
		chatID = callback.Message.Message.Chat.ID
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text: fmt.Sprintf("🎉 Предмет успешно создан!\n\n"+
			"📚 %s\n"+
			"📝 %s\n"+
			"💰 %.2f ₽\n"+
			"⏱ %d минут\n"+
			"⏳ Требуется одобрение: %s\n"+
			"ID: %d\n\n"+
			"Теперь вы можете:\n"+
			"• Добавить временные слоты: /addslots\n"+
			"• Управлять предметами: /mysubjects",
			name, description, priceInRubles, duration, approvalText, subject.ID),
	})

	common.AnswerCallback(ctx, b, callback.ID, "✅ Предмет создан!")
}

// HandleViewSubject показывает детали предмета (для учителя-владельца)
func HandleViewSubject(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleViewSubject called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	subjectID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		h.Logger.Error("Failed to parse subject ID", zap.Error(err), zap.String("data", callback.Data))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	h.Logger.Info("Viewing subject", zap.Int64("subject_id", subjectID))

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		h.Logger.Error("Subject not found",
			zap.Int64("subject_id", subjectID),
			zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	h.Logger.Info("Subject found, showing details",
		zap.Int64("subject_id", subjectID),
		zap.String("name", subject.Name))

	// Используем билдер экрана
	text, keyboard := common.BuildViewSubjectScreen(subject)

	h.Logger.Info("Sending view subject message",
		zap.Int64("chat_id", msg.Chat.ID),
		zap.Int("message_id", msg.ID))

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	if err != nil {
		errStr := err.Error()
		// Игнорируем ошибку "message is not modified" - это не настоящая ошибка
		if common.IsMessageNotModifiedError(err) {
			// Сообщение уже имеет нужное содержимое, ничего не делаем
		} else if common.IsNoTextInMessageError(err) {
			// Сообщение содержит медиа (фото и т.д.)
			// В Telegram Bot API нельзя превратить медиа-сообщение в текстовое через редактирование
			// Приходится удалить и отправить новое
			h.Logger.Info("Message has no text, deleting and sending new",
				zap.Int64("chat_id", msg.Chat.ID),
				zap.Int("message_id", msg.ID),
				zap.String("error", errStr))
			// Удаляем старое сообщение (игнорируем ошибку удаления)
			b.DeleteMessage(ctx, &bot.DeleteMessageParams{
				ChatID:    msg.Chat.ID,
				MessageID: msg.ID,
			})
			// Отправляем новое сообщение
			_, sendErr := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      msg.Chat.ID,
				Text:        text,
				ParseMode:   models.ParseModeHTML,
				ReplyMarkup: keyboard,
			})
			if sendErr != nil {
				h.Logger.Error("Failed to send new message", zap.Error(sendErr))
			}
		} else {
			h.Logger.Error("Failed to edit message",
				zap.Error(err),
				zap.String("error_string", errStr),
				zap.Bool("is_no_text_error", common.IsNoTextInMessageError(err)))
		}
	}

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleEditSubject показывает меню редактирования предмета
func HandleEditSubject(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleEditSubject called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	subjectID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		h.Logger.Error("Failed to parse subject ID", zap.Error(err), zap.String("data", callback.Data))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	showEditSubjectScreen(ctx, b, callback, h, subjectID)
}

// showEditSubjectScreen отображает экран редактирования предмета
func showEditSubjectScreen(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler, subjectID int64) {
	h.Logger.Info("Editing subject", zap.Int64("subject_id", subjectID))

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		h.Logger.Error("Subject not found for editing",
			zap.Int64("subject_id", subjectID),
			zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	h.Logger.Info("Subject found, showing edit menu",
		zap.Int64("subject_id", subjectID),
		zap.String("name", subject.Name))

	// Используем билдер экрана
	text, keyboard := common.BuildEditSubjectScreen(subject)

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	if err != nil {
		// Игнорируем ошибку "message is not modified" - это не настоящая ошибка
		if common.IsMessageNotModifiedError(err) {
			// Сообщение уже имеет нужное содержимое, ничего не делаем
		} else if common.IsNoTextInMessageError(err) {
			// Сообщение содержит медиа (фото и т.д.)
			// В Telegram Bot API нельзя превратить медиа-сообщение в текстовое через редактирование
			// Приходится удалить и отправить новое
			h.Logger.Info("Message has no text, deleting and sending new",
				zap.Int64("chat_id", msg.Chat.ID),
				zap.Int("message_id", msg.ID))
			// Удаляем старое сообщение (игнорируем ошибку удаления)
			b.DeleteMessage(ctx, &bot.DeleteMessageParams{
				ChatID:    msg.Chat.ID,
				MessageID: msg.ID,
			})
			// Отправляем новое сообщение
			_, sendErr := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      msg.Chat.ID,
				Text:        text,
				ParseMode:   models.ParseModeHTML,
				ReplyMarkup: keyboard,
			})
			if sendErr != nil {
				h.Logger.Error("Failed to send new message", zap.Error(sendErr))
			}
		} else {
			h.Logger.Error("Failed to edit message in showEditSubjectScreen", zap.Error(err))
		}
	}

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleToggleSubject переключает активность предмета
func HandleToggleSubject(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	// Формат: toggle_subject:{id}:source (source = "list" или "edit")
	parts := strings.Split(callback.Data, ":")
	if len(parts) < 2 {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	subjectID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный ID")
		return
	}

	// Определяем источник (откуда пришли)
	source := "list" // по умолчанию
	if len(parts) >= 3 {
		source = parts[2]
	}

	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	subject, err := h.TeacherService.ToggleSubjectActive(ctx, user.ID, subjectID)
	if err != nil {
		h.Logger.Error("Failed to toggle subject", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось изменить статус")
		return
	}

	statusText := "активирован"
	if !subject.IsActive {
		statusText = "деактивирован"
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, fmt.Sprintf("✅ Предмет %s", statusText))

	// Возвращаемся туда, откуда пришли
	if source == "edit" {
		// Возвращаемся к экрану редактирования (передаем subjectID напрямую)
		showEditSubjectScreen(ctx, b, callback, h, subjectID)
	} else {
		// Возвращаемся к списку предметов
		msg := common.GetMessageFromCallback(callback)
		if msg != nil {
			h.HandleMySubjects(ctx, b, &models.Update{
				CallbackQuery: callback,
				Message: &models.Message{
					Chat: models.Chat{ID: msg.Chat.ID},
					From: &callback.From,
				},
			}, msg.ID)
		}
	}
}

// HandleDeleteSubject показывает подтверждение удаления
func HandleDeleteSubject(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	subjectID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	// Проверяем есть ли активные бронирования
	bookings, err := h.BookingService.GetBookingsBySubject(ctx, subjectID)
	if err != nil {
		h.Logger.Error("Failed to get bookings", zap.Error(err))
		bookings = []*model.Booking{}
	}

	// Используем билдер экрана
	text, keyboard := common.BuildDeleteSubjectConfirmScreen(subject, len(bookings))

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleConfirmDeleteSubject подтверждает удаление предмета
func HandleConfirmDeleteSubject(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	subjectID, err := common.ParseIDFromCallback(callback.Data)
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

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Получаем предмет перед удалением
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	// Получаем все бронирования для уведомлений
	bookings, err := h.BookingService.GetBookingsBySubject(ctx, subjectID)
	if err != nil {
		h.Logger.Error("Failed to get bookings", zap.Error(err))
		bookings = []*model.Booking{}
	}

	// Удаляем предмет
	err = h.TeacherService.DeleteSubject(ctx, user.ID, subjectID)
	if err != nil {
		h.Logger.Error("Failed to delete subject", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось удалить предмет")
		return
	}

	// Отправляем уведомления студентам
	notifyStudentsAboutSubjectDeletion(ctx, b, h, subject, bookings)

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
		Text: fmt.Sprintf(
			"✅ Предмет <b>%s</b> успешно удален.\n\n"+
				"Уведомления отправлены %d студентам.",
			subject.Name,
			len(bookings),
		),
		ParseMode: models.ParseModeHTML,
	})

	common.AnswerCallback(ctx, b, callback.ID, "✅ Предмет удален")
}

// HandleSubjectsPage обрабатывает пагинацию списка предметов
func HandleSubjectsPage(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleSubjectsPage called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	page, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		h.Logger.Error("Failed to parse page", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
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
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем предметы учителя
	subjects, err := h.TeacherService.GetTeacherSubjects(ctx, user.ID)
	if err != nil {
		h.Logger.Error("Failed to get teacher subjects", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось загрузить предметы")
		return
	}

	// Проверяем корректность страницы
	pageInt := int(page)
	const pageSize = 10
	if pageInt*pageSize >= len(subjects) {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверная страница")
		return
	}

	// Используем билдер экрана
	text, keyboard := common.BuildSubjectsListScreen(subjects, pageInt)

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// notifyStudentsAboutSubjectDeletion отправляет уведомления студентам об удалении предмета
func notifyStudentsAboutSubjectDeletion(ctx context.Context, b *bot.Bot, h *callbacktypes.Handler, subject *model.Subject, bookings []*model.Booking) {
	h.Logger.Info("Notifying students about subject deletion",
		zap.Int64("subject_id", subject.ID),
		zap.Int("bookings_count", len(bookings)))

	successCount := 0
	for _, booking := range bookings {
		student, err := h.UserService.GetByID(ctx, booking.StudentID)
		if err != nil || student == nil {
			h.Logger.Warn("Failed to get student for notification",
				zap.Int64("student_id", booking.StudentID),
				zap.Error(err))
			continue
		}

		notificationText := fmt.Sprintf(
			"❌ Отмена занятия\n\n"+
				"К сожалению, предмет \"%s\" был удален учителем.\n"+
				"Ваше бронирование #%d было автоматически отменено.",
			subject.Name,
			booking.ID,
		)

		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: student.TelegramID,
			Text:   notificationText,
		})

		if err != nil {
			h.Logger.Error("Failed to send notification to student",
				zap.Int64("student_id", student.ID),
				zap.Int64("telegram_id", student.TelegramID),
				zap.Error(err))
		} else {
			successCount++
		}
	}

	h.Logger.Info("Notifications sent",
		zap.Int("success_count", successCount),
		zap.Int("total_bookings", len(bookings)))
}
