package handlers

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/state"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// HandleStart обрабатывает команду /start
func (h *Handlers) HandleStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	user := update.Message.From

	// Регистрируем пользователя
	registeredUser, err := h.userService.RegisterUser(
		ctx,
		user.ID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.LanguageCode,
	)

	if err != nil {
		h.logger.Error("Failed to register user", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Произошла ошибка при регистрации. Попробуйте позже.",
		})
		return
	}

	welcomeText := fmt.Sprintf(
		"👋 Привет, %s!\n\n"+
			"Добро пожаловать в Scheduler Bot - бот для записи на занятия к учителям.\n\n"+
			"Доступные команды:\n"+
			"/subjects - Посмотреть все предметы\n"+
			"/findteachers - Найти публичных учителей\n"+
			"/mybookings - Мои записи\n"+
			"/help - Справка\n\n"+
			"Для учителей:\n"+
			"/becometeacher - Стать учителем\n"+
			"/mysubjects - Мои предметы\n"+
			"/myschedule - Моё расписание",
		registeredUser.FirstName,
	)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   welcomeText,
	})
}

// HandleHelp обрабатывает команду /help
func (h *Handlers) HandleHelp(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	helpText := "📚 Справка по командам:\n\n" +
		"Для студентов:\n" +
		"/start - Начать работу с ботом\n" +
		"/subjects - Список всех предметов\n" +
		"/mybookings - Мои записи на занятия\n" +
		"/help - Показать эту справку\n\n" +
		"Для учителей:\n" +
		"/becometeacher - Зарегистрироваться как учитель\n" +
		"/mysubjects - Управление своими предметами\n" +
		"/myschedule - Посмотреть расписание\n\n" +
		"Для записи на занятие выберите предмет из списка /subjects"

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   helpText,
	})
}

// HandleCancel обрабатывает команду /cancel - отмена текущего диалога
func (h *Handlers) HandleCancel(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	telegramID := update.Message.From.ID
	currentState := h.stateManager.GetState(telegramID)

	if currentState == state.StateNone {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Нет активных операций для отмены.",
		})
		return
	}

	// Очищаем состояние
	h.stateManager.ClearState(telegramID)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "✅ Операция отменена.\n\nИспользуйте /help для просмотра доступных команд.",
	})
}

// HandleTextMessage обрабатывает текстовые сообщения в зависимости от состояния пользователя
func (h *Handlers) HandleTextMessage(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}

	// Игнорируем команды (они обрабатываются другими handlers)
	if strings.HasPrefix(update.Message.Text, "/") {
		return
	}

	telegramID := update.Message.From.ID
	currentState := h.stateManager.GetState(telegramID)

	h.logger.Info("HandleTextMessage called",
		zap.Int64("telegram_id", telegramID),
		zap.String("text", update.Message.Text),
		zap.String("state", string(currentState)))

	// Если нет активного состояния, игнорируем
	if currentState == state.StateNone {
		h.logger.Debug("No active state, ignoring message",
			zap.Int64("telegram_id", telegramID))
		return
	}

	// Обрабатываем в зависимости от состояния
	switch currentState {
	case state.StateCreateSubjectName:
		h.logger.Info("Handling create subject name step",
			zap.Int64("telegram_id", telegramID))
		h.handleCreateSubjectNameStep(ctx, b, update)
	case state.StateCreateSubjectDescription:
		h.logger.Info("Handling create subject description step",
			zap.Int64("telegram_id", telegramID))
		h.handleCreateSubjectDescriptionStep(ctx, b, update)
	case state.StateCreateSubjectPrice:
		h.logger.Info("Handling create subject price step",
			zap.Int64("telegram_id", telegramID))
		h.handleCreateSubjectPriceStep(ctx, b, update)
	case state.StateCreateSubjectDuration:
		h.logger.Info("Handling create subject duration step",
			zap.Int64("telegram_id", telegramID))
		h.handleCreateSubjectDurationStep(ctx, b, update)
	case state.StateEditSubjectName:
		h.handleEditSubjectName(ctx, b, update)
	case state.StateEditSubjectDescription:
		h.handleEditSubjectDescription(ctx, b, update)
	case state.StateEditSubjectPrice:
		h.handleEditSubjectPrice(ctx, b, update)
	case state.StateEditSubjectDuration:
		h.handleEditSubjectDuration(ctx, b, update)
	case state.StateEnteringInviteCode:
		h.handleEnteringInviteCode(ctx, b, update)
	case state.StateMarkSlotBusyComment:
		h.handleMarkSlotBusyComment(ctx, b, update)
	case "custom_slot_time":
		h.handleCustomSlotTime(ctx, b, update)
	default:
		h.logger.Warn("Unknown state", zap.String("state", string(currentState)))
	}
}

// handleCustomSlotTime обрабатывает ввод кастомного времени для слота
func (h *Handlers) handleCustomSlotTime(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.Message.From.ID
	timeText := strings.TrimSpace(update.Message.Text)

	h.logger.Info("Processing custom slot time",
		zap.Int64("telegram_id", telegramID),
		zap.String("time", timeText))

	// Получаем сохранённые данные
	subjectIDData, ok1 := h.stateManager.GetData(telegramID, "subject_id")
	dateStrData, ok2 := h.stateManager.GetData(telegramID, "date_str")

	if !ok1 || !ok2 {
		h.logger.Error("Missing data for custom slot time",
			zap.Int64("telegram_id", telegramID))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка: данные не найдены. Начните заново через /mysubjects",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	subjectID, ok := subjectIDData.(int64)
	if !ok {
		h.logger.Error("Invalid subject ID type", zap.Any("data", subjectIDData))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка: неверный формат данных",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	dateStr, ok := dateStrData.(string)
	if !ok {
		h.logger.Error("Invalid date string type", zap.Any("data", dateStrData))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка: неверный формат данных",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	// Используем логику из teacher package
	// Встраиваем обработку времени здесь (можно вынести в service позже)
	h.processCustomSlotTime(ctx, b, update, timeText, subjectID, dateStr)
}

// processCustomSlotTime обрабатывает введенное пользователем время
func (h *Handlers) processCustomSlotTime(ctx context.Context, b *bot.Bot, update *models.Update, timeText string, subjectID int64, dateStr string) {
	telegramID := update.Message.From.ID

	// Проверяем формат времени (ЧЧ:ММ)
	timeRegex := regexp.MustCompile(`^([0-1][0-9]|2[0-3]):([0-5][0-9])$`)
	if !timeRegex.MatchString(timeText) {
		h.logger.Warn("Invalid time format",
			zap.String("time", timeText))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text: "❌ Неверный формат времени!\n\n" +
				"Используйте формат <b>ЧЧ:ММ</b> (например, 09:30 или 14:45)\n\n" +
				"Попробуйте еще раз или отправьте /cancel для отмены.",
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	user, err := h.userService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		h.logger.Error("User not found", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Пользователь не найден",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	// Получаем предмет
	subject, err := h.teacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		h.logger.Error("Subject not found", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Предмет не найден",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	// Парсим дату и время
	dateTimeStr := fmt.Sprintf("%s %s", dateStr, timeText)
	startTime, err := time.Parse("2006-01-02 15:04", dateTimeStr)
	if err != nil {
		h.logger.Error("Failed to parse datetime", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Не удалось обработать дату/время",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	endTime := startTime.Add(time.Duration(subject.Duration) * time.Minute)

	// Создаем слот
	slot, err := h.teacherService.CreateSlot(ctx, user.ID, subjectID, startTime, endTime)
	if err != nil {
		h.logger.Error("Failed to create slot", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("❌ Не удалось создать слот: %v", err),
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	h.logger.Info("Slot created successfully via custom time",
		zap.Int64("slot_id", slot.ID),
		zap.Time("start_time", startTime))

	// Очищаем состояние
	h.stateManager.ClearState(telegramID)

	text := fmt.Sprintf("✅ <b>Слот создан!</b>\n\n"+
		"📚 Предмет: %s\n"+
		"📅 Дата: %s\n"+
		"🕐 Время: %s - %s\n"+
		"⏱ Длительность: %d мин\n\n"+
		"Посмотреть расписание: /myschedule",
		subject.Name,
		startTime.Format("02.01.2006 (Monday)"),
		startTime.Format("15:04"),
		endTime.Format("15:04"),
		subject.Duration)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
}

// handleMarkSlotBusyComment обрабатывает ввод комментария для пометки слота занятым
func (h *Handlers) handleMarkSlotBusyComment(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.Message.From.ID
	commentText := strings.TrimSpace(update.Message.Text)

	h.logger.Info("Processing mark slot busy comment",
		zap.Int64("telegram_id", telegramID),
		zap.String("comment", commentText))

	// Получаем сохранённые данные
	slotIDData, ok := h.stateManager.GetData(telegramID, "slot_id")
	if !ok {
		h.logger.Error("Missing slot_id in state", zap.Int64("telegram_id", telegramID))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка: данные не найдены. Начните заново.",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	slotID, ok := slotIDData.(int64)
	if !ok {
		h.logger.Error("Invalid slot_id type", zap.Any("data", slotIDData))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка: неверный формат данных.",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	// Получаем пользователя
	user, err := h.userService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		h.logger.Error("Failed to get user", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка получения данных пользователя.",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	// Подготавливаем комментарий (если пустой, то nil)
	var comment *string
	if commentText != "" && commentText != "/skip" {
		comment = &commentText
	}

	// Получаем subject_id и date для возврата ДО очистки состояния
	subjectIDData, hasSubjectID := h.stateManager.GetData(telegramID, "subject_id")
	dateData, hasDate := h.stateManager.GetData(telegramID, "date")

	// Помечаем слот как занятый с комментарием
	err = h.teacherService.MarkSlotBusyWithComment(ctx, slotID, user.ID, comment)
	if err != nil {
		h.logger.Error("Failed to mark slot busy with comment", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Не удалось пометить слот как занятый.",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	// Очищаем состояние
	h.stateManager.ClearState(telegramID)

	// Получаем информацию о слоте для отображения
	slot, err := h.teacherService.GetSlotByID(ctx, slotID)
	if err != nil || slot == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "✅ Слот помечен как занятый.",
		})
		return
	}

	// Формируем сообщение об успехе
	timeStr := fmt.Sprintf("%s - %s", slot.StartTime.Format("15:04"), slot.EndTime.Format("15:04"))
	text := fmt.Sprintf("✅ <b>Слот помечен как занятый</b>\n\n"+
		"🕐 Время: %s\n"+
		"📅 Дата: %s\n",
		timeStr,
		slot.StartTime.Format("02.01.2006"))

	if comment != nil {
		text += fmt.Sprintf("📝 Комментарий: %s\n", *comment)
	}

	// Формируем кнопку возврата, если есть данные
	var keyboard *models.InlineKeyboardMarkup
	if hasSubjectID && hasDate {
		subjectIDStr, _ := subjectIDData.(string)
		dateStr, _ := dateData.(string)
		// Получаем weekday из даты слота
		slot, err := h.teacherService.GetSlotByID(ctx, slotID)
		if err == nil && slot != nil {
			weekdayName := slot.StartTime.Weekday().String()
			// Преобразуем английское название в русское
			weekdayMap := map[string]string{
				"Monday":    "Понедельник",
				"Tuesday":   "Вторник",
				"Wednesday": "Среда",
				"Thursday":  "Четверг",
				"Friday":    "Пятница",
				"Saturday":  "Суббота",
				"Sunday":    "Воскресенье",
			}
			if ruWeekday, ok := weekdayMap[weekdayName]; ok {
				weekdayName = ruWeekday
			}
			keyboard = &models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{{Text: "⬅️ Вернуться к расписанию", CallbackData: fmt.Sprintf("view_schedule_day:%s:%s:%s", subjectIDStr, dateStr, weekdayName)}},
				},
			}
		}
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})
}
