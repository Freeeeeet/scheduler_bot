package slots

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common/formatting"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

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

	// Показываем выбор режима создания слотов
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
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("subject_schedule:%d", subjectID)},
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

	h.Logger.Info("Mode selection message sent successfully")
	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleSetWeekday обрабатывает выбор дня недели и показывает выбор времени (для recurring)
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

	// Получаем предмет для определения длительности
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		h.Logger.Error("Failed to get subject", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	// Генерируем временные слоты на основе длительности занятия
	duration := subject.Duration // в минутах
	var buttons [][]models.InlineKeyboardButton

	// Генерируем слоты с 00:00 до 23:59 с шагом в длительность занятия
	now := time.Now()
	currentTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 0, 0, now.Location())

	var row []models.InlineKeyboardButton
	for currentTime.Before(endOfDay) {
		timeStr := currentTime.Format("15:04")
		hour := currentTime.Hour()
		minute := currentTime.Minute()

		row = append(row, models.InlineKeyboardButton{
			Text:         timeStr,
			CallbackData: fmt.Sprintf("set_time:%d:%d:%d:%d", subjectID, weekdayNum, hour, minute),
		})

		// По 3 кнопки в ряд для компактности
		if len(row) == 3 {
			buttons = append(buttons, row)
			row = []models.InlineKeyboardButton{}
		}

		currentTime = currentTime.Add(time.Duration(duration) * time.Minute)
	}

	// Добавляем оставшиеся кнопки
	if len(row) > 0 {
		buttons = append(buttons, row)
	}

	// Кнопка для ввода своего времени
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⌨️ Ввести своё время", CallbackData: fmt.Sprintf("custom_recurring_time:%d:%d", subjectID, weekdayNum)},
	})

	// Кнопка "Назад"
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("create_slots:%d", subjectID)},
	})

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	text := fmt.Sprintf("📅 Создание постоянного расписания (Шаг 2/2)\n\n"+
		"День недели: %s\n"+
		"⏱ Длительность: %d мин\n\n"+
		"Временные слоты рассчитаны автоматически.\n"+
		"Выберите время начала занятия:\n\n"+
		"🔄 Будет создано постоянное еженедельное расписание\n"+
		"📆 Слоты автоматически создаются на 4 недели вперёд", formatting.GetWeekdayName(weekdayNum), duration)

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

// HandleSetTime создает слоты для выбранного времени (recurring slots)
func HandleSetTime(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleSetTime called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: set_time:123:1:9:0  (subject_id:weekday:hour:minute)
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 5 {
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

	minute, err := strconv.Atoi(parts[4])
	if err != nil {
		h.Logger.Error("Failed to parse minute", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверные минуты")
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
		zap.Int("minute", minute),
		zap.Int("duration", subject.Duration))

	// Создаем слоты на 4 недели
	weekday := time.Weekday(weekdayNum)
	err = h.TeacherService.CreateWeeklySlots(ctx, user.ID, subjectID, weekday, hour, minute, subject.Duration)
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

	msg := common.GetMessageFromCallback(callback)
	if msg != nil {
		text := fmt.Sprintf("✅ Постоянное расписание создано!\n\n"+
			"📚 Предмет: %s\n"+
			"📅 День: %s\n"+
			"🕐 Время: %02d:%02d\n"+
			"⏱ Длительность: %d мин\n\n"+
			"🔄 Автоматически создаются слоты каждую неделю\n"+
			"📆 Сейчас доступны слоты на 4 недели вперёд\n\n"+
			"Посмотреть расписание: /myschedule",
			subject.Name,
			formatting.GetWeekdayName(weekdayNum),
			hour,
			minute,
			subject.Duration)

		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    msg.Chat.ID,
			MessageID: msg.ID,
			Text:      text,
		})
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Постоянное расписание создано!")
}

// HandleManualBook позволяет учителю вручную записать студента (в разработке)
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
