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

// ========================
// Schedule Management Handlers
// ========================

// HandleViewSchedule показывает расписание учителя
func HandleViewSchedule(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleViewSchedule called",
		zap.Int64("user_id", callback.From.ID))

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Удаляем старое сообщение
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
	})

	// Отправляем новое (через HandleMySchedule)
	update := &models.Update{
		Message: &models.Message{
			Chat: models.Chat{ID: msg.Chat.ID},
			From: &callback.From,
		},
	}

	h.HandleMySchedule(ctx, b, update)
	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleAddSlots начинает процесс добавления слотов
func HandleAddSlots(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	text := "🕐 Добавление временных слотов\n\n" +
		"Для добавления слотов используйте команду:\n" +
		"/addslots\n\n" +
		"Или создайте слоты через API."

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
		Text:      text,
	})

	common.AnswerCallback(ctx, b, callback.ID, "Добавление слотов")
}

// HandleCreateSlotsStart начинает процесс создания слотов для предмета
func HandleCreateSlotsStart(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleCreateSlotsStart called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	subjectID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		h.Logger.Error("Failed to parse subject ID", zap.Error(err), zap.String("data", callback.Data))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	h.Logger.Info("Parsed subject ID for slot creation", zap.Int64("subject_id", subjectID))

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Сохраняем ID предмета в state
	telegramID := callback.From.ID
	h.StateManager.SetState(telegramID, "create_slots_weekday")
	h.StateManager.SetData(telegramID, "subject_id", subjectID)

	h.Logger.Info("Set state for slot creation",
		zap.Int64("telegram_id", telegramID),
		zap.Int64("subject_id", subjectID))

	// Сначала спрашиваем режим создания слотов
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "📆 Один раз (на конкретный день)", CallbackData: fmt.Sprintf("slot_mode:%d:single", subjectID)},
			},
			{
				{Text: "🔄 Постоянное расписание", CallbackData: fmt.Sprintf("slot_mode:%d:recurring", subjectID)},
			},
			{
				{Text: "📅 На период (несколько недель)", CallbackData: fmt.Sprintf("slot_mode:%d:period", subjectID)},
			},
			{
				{Text: "⚡️ Заполнить рабочий день (9-18)", CallbackData: fmt.Sprintf("slot_mode:%d:workday", subjectID)},
			},
			{
				{Text: "❌ Отмена", CallbackData: "back_to_main"},
			},
		},
	}

	h.Logger.Info("Sending slot mode selection message",
		zap.Int64("chat_id", msg.Chat.ID),
		zap.Int("message_id", msg.ID))

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        "📅 Создание слотов\n\nВыберите режим создания:",
		ReplyMarkup: keyboard,
	})

	if err != nil {
		h.Logger.Error("Failed to edit message",
			zap.Error(err),
			zap.Int64("chat_id", msg.Chat.ID),
			zap.Int("message_id", msg.ID))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка отправки сообщения")
		return
	}

	h.Logger.Info("Weekday selection message sent successfully")
	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleSetWeekday обрабатывает выбор дня недели и показывает выбор времени
func HandleSetWeekday(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleSetWeekday called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: set_weekday:123:1  (subject_id:weekday)
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 3 {
		h.Logger.Error("Invalid callback format for set_weekday", zap.String("data", callback.Data))
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
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный день недели")
		return
	}

	h.Logger.Info("Parsed weekday selection",
		zap.Int64("subject_id", subjectID),
		zap.Int("weekday", weekdayNum))

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Мапа для названий дней недели
	weekdayNames := map[int]string{
		0: "Воскресенье",
		1: "Понедельник",
		2: "Вторник",
		3: "Среда",
		4: "Четверг",
		5: "Пятница",
		6: "Суббота",
	}

	// Кнопки для выбора времени (популярные часы)
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "09:00", CallbackData: fmt.Sprintf("set_time:%d:%d:9", subjectID, weekdayNum)},
				{Text: "10:00", CallbackData: fmt.Sprintf("set_time:%d:%d:10", subjectID, weekdayNum)},
				{Text: "11:00", CallbackData: fmt.Sprintf("set_time:%d:%d:11", subjectID, weekdayNum)},
			},
			{
				{Text: "12:00", CallbackData: fmt.Sprintf("set_time:%d:%d:12", subjectID, weekdayNum)},
				{Text: "13:00", CallbackData: fmt.Sprintf("set_time:%d:%d:13", subjectID, weekdayNum)},
				{Text: "14:00", CallbackData: fmt.Sprintf("set_time:%d:%d:14", subjectID, weekdayNum)},
			},
			{
				{Text: "15:00", CallbackData: fmt.Sprintf("set_time:%d:%d:15", subjectID, weekdayNum)},
				{Text: "16:00", CallbackData: fmt.Sprintf("set_time:%d:%d:16", subjectID, weekdayNum)},
				{Text: "17:00", CallbackData: fmt.Sprintf("set_time:%d:%d:17", subjectID, weekdayNum)},
			},
			{
				{Text: "18:00", CallbackData: fmt.Sprintf("set_time:%d:%d:18", subjectID, weekdayNum)},
				{Text: "19:00", CallbackData: fmt.Sprintf("set_time:%d:%d:19", subjectID, weekdayNum)},
				{Text: "20:00", CallbackData: fmt.Sprintf("set_time:%d:%d:20", subjectID, weekdayNum)},
			},
			{
				{Text: "❌ Отмена", CallbackData: "back_to_main"},
			},
		},
	}

	text := fmt.Sprintf("📅 Создание постоянного расписания (Шаг 2/2)\n\n"+
		"День недели: %s\n\n"+
		"Выберите время начала занятия:\n\n"+
		"🔄 Будет создано постоянное еженедельное расписание\n"+
		"📆 Слоты автоматически создаются на 4 недели вперёд", weekdayNames[weekdayNum])

	h.Logger.Info("Sending time selection message",
		zap.Int64("chat_id", msg.Chat.ID),
		zap.Int("message_id", msg.ID),
		zap.Int("weekday", weekdayNum))

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ReplyMarkup: keyboard,
	})

	if err != nil {
		h.Logger.Error("Failed to edit message for time selection",
			zap.Error(err),
			zap.Int64("chat_id", msg.Chat.ID),
			zap.Int("message_id", msg.ID))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка отправки сообщения")
		return
	}

	h.Logger.Info("Time selection message sent successfully")
	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleSetTime создает слоты для выбранного времени
func HandleSetTime(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleSetTime called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: set_time:123:1:9  (subject_id:weekday:hour)
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

	h.Logger.Info("Parsed slot creation parameters",
		zap.Int64("subject_id", subjectID),
		zap.Int("weekday", weekdayNum),
		zap.Int("hour", hour))

	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		h.Logger.Error("User not found", zap.Int64("telegram_id", telegramID), zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем предмет чтобы узнать длительность
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		h.Logger.Error("Subject not found", zap.Int64("subject_id", subjectID), zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	h.Logger.Info("Creating weekly slots",
		zap.Int64("teacher_id", user.ID),
		zap.Int64("subject_id", subjectID),
		zap.Int("weekday", weekdayNum),
		zap.Int("hour", hour),
		zap.Int("duration", subject.Duration))

	// Создаем слоты на 4 недели
	weekday := time.Weekday(weekdayNum)
	err = h.TeacherService.CreateWeeklySlots(ctx, user.ID, subjectID, weekday, hour, 0, subject.Duration)
	if err != nil {
		h.Logger.Error("Failed to create weekly slots",
			zap.Int64("teacher_id", user.ID),
			zap.Int64("subject_id", subjectID),
			zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось создать слоты")
		return
	}

	h.Logger.Info("Weekly slots created successfully",
		zap.Int64("teacher_id", user.ID),
		zap.Int64("subject_id", subjectID))

	weekdayNames := map[int]string{
		0: "Воскресенье",
		1: "Понедельник",
		2: "Вторник",
		3: "Среда",
		4: "Четверг",
		5: "Пятница",
		6: "Суббота",
	}

	msg := common.GetMessageFromCallback(callback)
	if msg != nil {
		text := fmt.Sprintf("✅ Постоянное расписание создано!\n\n"+
			"📚 Предмет: %s\n"+
			"📅 День: %s\n"+
			"🕐 Время: %02d:00\n"+
			"⏱ Длительность: %d мин\n\n"+
			"🔄 Автоматически создаются слоты каждую неделю\n"+
			"📆 Сейчас доступны слоты на 4 недели вперёд\n\n"+
			"Посмотреть расписание: /myschedule",
			subject.Name,
			weekdayNames[weekdayNum],
			hour,
			subject.Duration)

		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    msg.Chat.ID,
			MessageID: msg.ID,
			Text:      text,
		})
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Постоянное расписание создано!")
}

// HandleManualBook позволяет учителю вручную записать студента
func HandleManualBook(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	text := "📝 Ручная запись студента\n\n" +
		"Функция находится в разработке.\n\n" +
		"В будущем вы сможете:\n" +
		"• Записать студента на свободный слот\n" +
		"• Указать имя или ID студента\n" +
		"• Выбрать предмет и время"

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "⬅️ Назад", CallbackData: "back_to_main"}},
		},
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "Ручная запись (в разработке)")
}
