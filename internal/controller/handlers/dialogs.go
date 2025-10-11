package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/state"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// HandleCreateSubjectStart начинает процесс создания предмета
func (h *Handlers) HandleCreateSubjectStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	user, ok := h.requireTeacher(ctx, b, update)
	if !ok {
		return
	}

	telegramID := update.Message.From.ID

	h.logger.Info("Starting subject creation",
		zap.Int64("telegram_id", telegramID),
		zap.Int64("teacher_id", user.ID))

	// Сохраняем teacher_id в данных
	h.stateManager.SetState(telegramID, state.StateCreateSubjectName)
	h.stateManager.SetData(telegramID, "teacher_id", user.ID)

	h.logger.Info("Set initial state and data",
		zap.Int64("telegram_id", telegramID),
		zap.String("state", string(state.StateCreateSubjectName)))

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: "📝 Создание нового предмета\n\n" +
			"Шаг 1 из 4: Как будет называться предмет?\n\n" +
			"Например: Математика, Английский язык, Программирование\n\n" +
			"Для отмены используйте /cancel",
	})
}

// handleCreateSubjectNameStep обрабатывает ввод названия предмета
func (h *Handlers) handleCreateSubjectNameStep(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.Message.From.ID
	name := strings.TrimSpace(update.Message.Text)

	h.logger.Info("Processing name step",
		zap.Int64("telegram_id", telegramID),
		zap.String("name", name))

	if len(name) < SubjectNameMinLength {
		h.logger.Warn("Name too short",
			zap.Int("length", len(name)),
			zap.Int("min", SubjectNameMinLength))
		h.sendError(ctx, b, update.Message.Chat.ID,
			fmt.Sprintf("❌ Название слишком короткое. Минимум %d символа.\n\nПопробуйте ещё раз:", SubjectNameMinLength))
		return
	}

	if len(name) > SubjectNameMaxLength {
		h.logger.Warn("Name too long",
			zap.Int("length", len(name)),
			zap.Int("max", SubjectNameMaxLength))
		h.sendError(ctx, b, update.Message.Chat.ID,
			fmt.Sprintf("❌ Название слишком длинное. Максимум %d символов.\n\nПопробуйте ещё раз:", SubjectNameMaxLength))
		return
	}

	// Сохраняем название и переходим к следующему шагу
	h.stateManager.SetData(telegramID, "name", name)
	h.stateManager.SetState(telegramID, state.StateCreateSubjectDescription)

	h.logger.Info("Name saved, moving to description step",
		zap.Int64("telegram_id", telegramID))

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: fmt.Sprintf("✅ Название: %s\n\n"+
			"Шаг 2 из 4: Напишите краткое описание предмета\n\n"+
			"Например: Подготовка к ЕГЭ, Разговорный английский, Веб-разработка для начинающих\n\n"+
			"Для отмены используйте /cancel", name),
	})
}

// handleCreateSubjectDescriptionStep обрабатывает ввод описания
func (h *Handlers) handleCreateSubjectDescriptionStep(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.Message.From.ID
	description := strings.TrimSpace(update.Message.Text)

	h.logger.Info("Processing description step",
		zap.Int64("telegram_id", telegramID),
		zap.String("description", description))

	if len(description) < SubjectDescriptionMinLength {
		h.logger.Warn("Description too short",
			zap.Int("length", len(description)),
			zap.Int("min", SubjectDescriptionMinLength))
		h.sendError(ctx, b, update.Message.Chat.ID,
			fmt.Sprintf("❌ Описание слишком короткое. Минимум %d символов.\n\nПопробуйте ещё раз:", SubjectDescriptionMinLength))
		return
	}

	if len(description) > SubjectDescriptionMaxLength {
		h.logger.Warn("Description too long",
			zap.Int("length", len(description)),
			zap.Int("max", SubjectDescriptionMaxLength))
		h.sendError(ctx, b, update.Message.Chat.ID,
			fmt.Sprintf("❌ Описание слишком длинное. Максимум %d символов.\n\nПопробуйте ещё раз:", SubjectDescriptionMaxLength))
		return
	}

	// Сохраняем описание и переходим к следующему шагу
	h.stateManager.SetData(telegramID, "description", description)
	h.stateManager.SetState(telegramID, state.StateCreateSubjectPrice)

	name, _ := h.stateManager.GetData(telegramID, "name")

	h.logger.Info("Description saved, moving to price step",
		zap.Int64("telegram_id", telegramID),
		zap.String("name", name.(string)))

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: fmt.Sprintf("✅ Название: %s\n"+
			"✅ Описание: %s\n\n"+
			"Шаг 3 из 4: Укажите стоимость занятия в рублях\n\n"+
			"Например: 1500, 2000, 500\n\n"+
			"Для отмены используйте /cancel", name, description),
	})
}

// handleCreateSubjectPriceStep обрабатывает ввод цены
func (h *Handlers) handleCreateSubjectPriceStep(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.Message.From.ID
	priceStr := strings.TrimSpace(update.Message.Text)

	h.logger.Info("Processing price step",
		zap.Int64("telegram_id", telegramID),
		zap.String("price_input", priceStr))

	price, err := strconv.Atoi(priceStr)
	if err != nil || price < 0 {
		h.logger.Warn("Invalid price format",
			zap.Error(err),
			zap.String("input", priceStr))
		h.sendError(ctx, b, update.Message.Chat.ID,
			"❌ Неверный формат цены. Введите целое число (например: 1500).\n\nПопробуйте ещё раз:")
		return
	}

	if price > SubjectMaxPrice {
		h.logger.Warn("Price too high",
			zap.Int("price", price),
			zap.Int("max", SubjectMaxPrice))
		h.sendError(ctx, b, update.Message.Chat.ID,
			fmt.Sprintf("❌ Цена слишком большая. Максимум %s.\n\nПопробуйте ещё раз:", FormatPrice(SubjectMaxPrice*100)))
		return
	}

	// Конвертируем в копейки для хранения в БД
	priceInCents := price * 100

	// Сохраняем цену и переходим к выбору длительности (кнопками)
	h.stateManager.SetData(telegramID, "price", priceInCents)
	h.stateManager.SetState(telegramID, state.StateCreateSubjectDuration)

	name, _ := h.stateManager.GetData(telegramID, "name")
	description, _ := h.stateManager.GetData(telegramID, "description")

	h.logger.Info("Price saved, showing duration buttons",
		zap.Int64("telegram_id", telegramID),
		zap.Int("price_cents", priceInCents))

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "30 мин", CallbackData: "create_subject_set_duration:30"},
				{Text: "1 час", CallbackData: "create_subject_set_duration:60"},
			},
			{
				{Text: "1.5 часа", CallbackData: "create_subject_set_duration:90"},
				{Text: "2 часа", CallbackData: "create_subject_set_duration:120"},
			},
			{
				{Text: "2.5 часа", CallbackData: "create_subject_set_duration:150"},
				{Text: "3 часа", CallbackData: "create_subject_set_duration:180"},
			},
			{
				{Text: "3.5 часа", CallbackData: "create_subject_set_duration:210"},
				{Text: "4 часа", CallbackData: "create_subject_set_duration:240"},
			},
			{
				{Text: "✏️ Свой вариант (ввести вручную)", CallbackData: "create_subject_set_duration:custom"},
			},
		},
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: fmt.Sprintf("✅ Название: %s\n"+
			"✅ Описание: %s\n"+
			"✅ Цена: %d ₽\n\n"+
			"Шаг 4 из 5: Выберите длительность занятия:",
			name, description, price),
		ReplyMarkup: keyboard,
	})
}

// handleCreateSubjectDurationStep обрабатывает ввод длительности и создаёт предмет
func (h *Handlers) handleCreateSubjectDurationStep(ctx context.Context, b *bot.Bot, update *models.Update) {
	telegramID := update.Message.From.ID
	durationStr := strings.TrimSpace(update.Message.Text)

	h.logger.Info("Processing duration step",
		zap.Int64("telegram_id", telegramID),
		zap.String("duration_input", durationStr))

	duration, err := strconv.Atoi(durationStr)
	if err != nil || duration <= 0 {
		h.logger.Warn("Invalid duration format",
			zap.Error(err),
			zap.String("input", durationStr))
		h.sendError(ctx, b, update.Message.Chat.ID,
			"❌ Неверный формат длительности. Введите целое число минут (например: 60).\n\nПопробуйте ещё раз:")
		return
	}

	if duration < SubjectMinDuration || duration > SubjectMaxDuration {
		h.logger.Warn("Duration out of range",
			zap.Int("duration", duration),
			zap.Int("min", SubjectMinDuration),
			zap.Int("max", SubjectMaxDuration))
		h.sendError(ctx, b, update.Message.Chat.ID,
			fmt.Sprintf("❌ Длительность должна быть от %d до %d минут.\n\nПопробуйте ещё раз:", SubjectMinDuration, SubjectMaxDuration))
		return
	}

	// Получаем все сохранённые данные
	allData := h.stateManager.GetAllData(telegramID)
	name, _ := allData["name"].(string)
	description, _ := allData["description"].(string)
	price, _ := allData["price"].(int)

	h.logger.Info("Retrieved subject data from state",
		zap.String("name", name),
		zap.String("description", description),
		zap.Int("price", price),
		zap.Int("duration", duration))

	// Переходим к вопросу об одобрении
	h.stateManager.SetState(telegramID, state.StateCreateSubjectApproval)
	h.stateManager.SetData(telegramID, "duration", duration)

	h.logger.Info("Set state to approval step", zap.Int64("telegram_id", telegramID))

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "✅ Да, требуется одобрение", CallbackData: "create_subject_approval_yes"},
			},
			{
				{Text: "❌ Нет, записываться свободно", CallbackData: "create_subject_approval_no"},
			},
		},
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: fmt.Sprintf("✅ Название: %s\n"+
			"✅ Описание: %s\n"+
			"✅ Цена: %s\n"+
			"✅ Длительность: %d минут\n\n"+
			"Шаг 5 из 5: Требуется ли ваше одобрение для записи на этот предмет?\n\n"+
			"• 🟢 Да - студенты отправляют запрос, вы одобряете\n"+
			"• 🔵 Нет - студенты записываются автоматически\n\n"+
			"Для отмены используйте /cancel",
			name, description, FormatPrice(price), duration),
		ReplyMarkup: keyboard,
	})

	if err != nil {
		h.logger.Error("Failed to send approval message",
			zap.Error(err),
			zap.Int64("telegram_id", telegramID))
		return
	}

	h.logger.Info("Successfully sent approval step message", zap.Int64("telegram_id", telegramID))
}
