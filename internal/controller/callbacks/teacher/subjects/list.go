package subjects

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

	price := float64(subject.Price) / 100
	statusText := "✅ Активен"
	if !subject.IsActive {
		statusText = "⏸ Неактивен"
	}

	approvalText := "❌ Нет"
	if subject.RequiresBookingApproval {
		approvalText = "✅ Да"
	}

	text := fmt.Sprintf(
		"📚 <b>%s</b>\n\n"+
			"📝 Описание: %s\n"+
			"💰 Цена: %.2f ₽\n"+
			"⏱ Длительность: %d мин\n"+
			"📊 Статус: %s\n"+
			"⏳ Требуется одобрение: %s\n\n"+
			"Выберите действие:",
		subject.Name,
		subject.Description,
		price,
		subject.Duration,
		statusText,
		approvalText,
	)

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "📅 Посмотреть расписание", CallbackData: fmt.Sprintf("view_schedule_calendar:%d", subjectID)},
			},
			{
				{Text: "📊 Управление расписанием", CallbackData: fmt.Sprintf("subject_schedule:%d", subjectID)},
			},
			{
				{Text: "✏️ Редактировать", CallbackData: fmt.Sprintf("edit_subject:%d", subjectID)},
			},
			{
				{Text: "🗑 Удалить предмет", CallbackData: fmt.Sprintf("delete_subject:%d", subjectID)},
			},
			{
				{Text: "⬅️ Назад к списку", CallbackData: "back_to_subjects"},
			},
		},
	}

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
		h.Logger.Error("Failed to edit message", zap.Error(err))
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

	price := float64(subject.Price) / 100
	statusText := "Активен ✅"
	if !subject.IsActive {
		statusText = "Неактивен ⏸"
	}
	approvalText := "Нет ❌"
	if subject.RequiresBookingApproval {
		approvalText = "Да ✅"
	}

	text := fmt.Sprintf(
		"🛠 <b>Редактирование предмета</b>\n\n"+
			"📚 Название: %s\n"+
			"📝 Описание: %s\n"+
			"💰 Цена: %.2f ₽\n"+
			"⏱ Длительность: %d мин\n"+
			"⏳ Требуется одобрение: %s\n"+
			"📊 Статус: %s\n\n"+
			"Выберите, что хотите изменить:",
		subject.Name,
		subject.Description,
		price,
		subject.Duration,
		approvalText,
		statusText,
	)

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "📝 Название", CallbackData: fmt.Sprintf("edit_field_name:%d", subjectID)},
				{Text: "📄 Описание", CallbackData: fmt.Sprintf("edit_field_desc:%d", subjectID)},
			},
			{
				{Text: "💰 Цена", CallbackData: fmt.Sprintf("edit_field_price:%d", subjectID)},
				{Text: "⏱ Длительность", CallbackData: fmt.Sprintf("edit_field_duration:%d", subjectID)},
			},
			{
				{Text: "⏳ Требуется одобрение", CallbackData: fmt.Sprintf("toggle_approval:%d", subjectID)},
			},
			{
				{Text: "📊 Изменить статус", CallbackData: fmt.Sprintf("toggle_subject:%d", subjectID)},
			},
			{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("view_subject:%d", subjectID)},
			},
		},
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleToggleSubject переключает активность предмета
func HandleToggleSubject(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
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

	// Обновляем список предметов
	msg := common.GetMessageFromCallback(callback)
	if msg != nil {
		h.HandleMySubjects(ctx, b, &models.Update{
			Message: &models.Message{
				Chat: models.Chat{ID: msg.Chat.ID},
				From: &callback.From,
			},
		})
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

	warningText := ""
	if len(bookings) > 0 {
		warningText = fmt.Sprintf("\n\n⚠️ **ВНИМАНИЕ!** У этого предмета есть %d активных бронирований.\n"+
			"Все студенты будут уведомлены об отмене.", len(bookings))
	}

	text := fmt.Sprintf(
		"❓ Вы уверены, что хотите удалить предмет <b>%s</b>?\n\n"+
			"Это действие удалит:\n"+
			"• Сам предмет\n"+
			"• Все временные слоты\n"+
			"• Все связанные бронирования%s",
		subject.Name,
		warningText,
	)

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "✅ Да, удалить", CallbackData: fmt.Sprintf("confirm_delete:%d", subjectID)},
			},
			{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("view_subject:%d", subjectID)},
			},
		},
	}

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

	// Пагинация
	const pageSize = 10
	pageInt := int(page)

	text := fmt.Sprintf("📚 Ваши предметы (всего: %d):\n\n", len(subjects))
	var buttons [][]models.InlineKeyboardButton

	// Вычисляем индексы для текущей страницы
	startIdx := pageInt * pageSize
	endIdx := startIdx + pageSize
	if endIdx > len(subjects) {
		endIdx = len(subjects)
	}

	// Проверяем корректность страницы
	if startIdx >= len(subjects) {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверная страница")
		return
	}

	// Показываем предметы текущей страницы
	for i := startIdx; i < endIdx; i++ {
		subject := subjects[i]
		statusEmoji := "✅"
		statusText := "Активен"

		if !subject.IsActive {
			statusEmoji = "⏸"
			statusText = "Неактивен"
		}

		text += fmt.Sprintf(
			"%d. %s %s\n"+
				"   💰 Цена: %.2f ₽\n"+
				"   ⏱ Длительность: %d мин\n"+
				"   📝 %s\n"+
				"   Статус: %s\n\n",
			i+1,
			statusEmoji,
			subject.Name,
			float64(subject.Price)/100,
			subject.Duration,
			subject.Description,
			statusText,
		)

		// Кнопки для каждого предмета
		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: fmt.Sprintf("📝 %s", subject.Name), CallbackData: fmt.Sprintf("view_subject:%d", subject.ID)},
			{Text: "✏️", CallbackData: fmt.Sprintf("edit_subject:%d", subject.ID)},
			{Text: statusEmoji, CallbackData: fmt.Sprintf("toggle_subject:%d", subject.ID)},
		})
	}

	// Добавляем подсказку
	text += "\n💡 Совет: Создайте временные слоты через /myschedule чтобы студенты могли записываться!\n\n"

	// Кнопки пагинации
	totalPages := (len(subjects) + pageSize - 1) / pageSize
	if totalPages > 1 {
		var paginationButtons []models.InlineKeyboardButton

		// Кнопка "Предыдущая" только если не первая страница
		if pageInt > 0 {
			paginationButtons = append(paginationButtons,
				models.InlineKeyboardButton{Text: "⬅️ Предыдущая", CallbackData: fmt.Sprintf("subjects_page:%d", pageInt-1)})
		}

		// Показываем номер страницы
		paginationButtons = append(paginationButtons,
			models.InlineKeyboardButton{Text: fmt.Sprintf("📄 %d/%d", pageInt+1, totalPages), CallbackData: "noop"})

		// Кнопка "Следующая" только если не последняя страница
		if pageInt < totalPages-1 {
			paginationButtons = append(paginationButtons,
				models.InlineKeyboardButton{Text: "Следующая ➡️", CallbackData: fmt.Sprintf("subjects_page:%d", pageInt+1)})
		}

		buttons = append(buttons, paginationButtons)
	}

	// Кнопка создать новый предмет
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "➕ Создать новый предмет", CallbackData: "create_first_subject"},
	})

	// Кнопка для быстрого перехода к расписанию
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "📅 Управление расписанием", CallbackData: "view_schedule"},
	})

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

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
