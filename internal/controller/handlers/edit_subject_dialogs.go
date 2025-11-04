package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// showEditSubjectScreen показывает экран редактирования предмета
func (h *Handlers) showEditSubjectScreen(ctx context.Context, b *bot.Bot, chatID int64, subjectID int64) {
	subject, err := h.teacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		h.logger.Error("Subject not found for edit screen",
			zap.Int64("subject_id", subjectID),
			zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Предмет не найден",
		})
		return
	}

	// Используем билдер экрана из общего пакета
	text, keyboard := buildEditSubjectScreen(subject)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})
}

// buildEditSubjectScreen - локальная обёртка для билдера из common
// (можно было бы импортировать напрямую, но для избежания циклических импортов дублируем)
func buildEditSubjectScreen(subject *model.Subject) (string, *models.InlineKeyboardMarkup) {
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

	// Формируем текст для кнопок с текущим состоянием
	approvalButtonText := "⏳ Требуется одобрение: нет"
	if subject.RequiresBookingApproval {
		approvalButtonText = "⏳ Требуется одобрение: да"
	}

	statusButtonText := "📊 Статус: активен"
	if !subject.IsActive {
		statusButtonText = "📊 Статус: неактивен"
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "📝 Название", CallbackData: fmt.Sprintf("edit_field_name:%d", subject.ID)},
				{Text: "📄 Описание", CallbackData: fmt.Sprintf("edit_field_desc:%d", subject.ID)},
			},
			{
				{Text: "💰 Цена", CallbackData: fmt.Sprintf("edit_field_price:%d", subject.ID)},
				{Text: "⏱ Длительность", CallbackData: fmt.Sprintf("edit_field_duration:%d", subject.ID)},
			},
			{
				{Text: approvalButtonText, CallbackData: fmt.Sprintf("toggle_approval:%d", subject.ID)},
			},
			{
				{Text: statusButtonText, CallbackData: fmt.Sprintf("toggle_subject:%d", subject.ID)},
			},
			{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("view_subject:%d", subject.ID)},
			},
		},
	}

	return text, keyboard
}

// handleEditSubjectName обрабатывает ввод нового названия предмета
func (h *Handlers) handleEditSubjectName(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	telegramID := update.Message.From.ID
	name := strings.TrimSpace(update.Message.Text)

	if len(name) == 0 || len(name) > 100 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Название должно быть от 1 до 100 символов. Попробуйте ещё раз:",
		})
		return
	}

	// Получаем ID предмета из state
	subjectIDRaw, ok := h.stateManager.GetData(telegramID, "subject_id")
	if !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка: предмет не найден",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	subjectID, ok := subjectIDRaw.(int64)
	if !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка: неверный формат",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	user, err := h.userService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка авторизации",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	// Получаем предмет и обновляем
	subject, err := h.teacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Предмет не найден",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	subject.Name = name
	err = h.teacherService.UpdateSubject(ctx, user.ID, subject)
	if err != nil {
		h.logger.Error("Failed to update subject name", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Не удалось обновить название",
		})
		return
	}

	h.stateManager.ClearState(telegramID)

	// Показываем экран редактирования предмета
	h.showEditSubjectScreen(ctx, b, update.Message.Chat.ID, subjectID)
}

// handleEditSubjectDescription обрабатывает ввод нового описания предмета
func (h *Handlers) handleEditSubjectDescription(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	telegramID := update.Message.From.ID
	description := strings.TrimSpace(update.Message.Text)

	if len(description) == 0 || len(description) > 500 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Описание должно быть от 1 до 500 символов. Попробуйте ещё раз:",
		})
		return
	}

	subjectIDRaw, ok := h.stateManager.GetData(telegramID, "subject_id")
	if !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка: предмет не найден",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	subjectID, ok := subjectIDRaw.(int64)
	if !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка: неверный формат",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	user, err := h.userService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка авторизации",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	subject, err := h.teacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Предмет не найден",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	subject.Description = description
	err = h.teacherService.UpdateSubject(ctx, user.ID, subject)
	if err != nil {
		h.logger.Error("Failed to update subject description", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Не удалось обновить описание",
		})
		return
	}

	h.stateManager.ClearState(telegramID)

	// Показываем экран редактирования предмета
	h.showEditSubjectScreen(ctx, b, update.Message.Chat.ID, subjectID)
}

// handleEditSubjectPrice обрабатывает ввод новой цены предмета
func (h *Handlers) handleEditSubjectPrice(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	telegramID := update.Message.From.ID
	priceText := strings.TrimSpace(update.Message.Text)

	// Парсим цену
	price, err := strconv.ParseFloat(priceText, 64)
	if err != nil || price < 0 || price > 1000000 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Неверная цена. Введите число от 0 до 1000000:",
		})
		return
	}

	priceInCents := int(price * 100)

	subjectIDRaw, ok := h.stateManager.GetData(telegramID, "subject_id")
	if !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка: предмет не найден",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	subjectID, ok := subjectIDRaw.(int64)
	if !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка: неверный формат",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	user, err := h.userService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка авторизации",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	subject, err := h.teacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Предмет не найден",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	subject.Price = priceInCents
	err = h.teacherService.UpdateSubject(ctx, user.ID, subject)
	if err != nil {
		h.logger.Error("Failed to update subject price", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Не удалось обновить цену",
		})
		return
	}

	h.stateManager.ClearState(telegramID)

	// Показываем экран редактирования предмета
	h.showEditSubjectScreen(ctx, b, update.Message.Chat.ID, subjectID)
}

// handleEditSubjectDuration обрабатывает ввод новой длительности предмета
func (h *Handlers) handleEditSubjectDuration(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	telegramID := update.Message.From.ID
	durationText := strings.TrimSpace(update.Message.Text)

	duration, err := strconv.Atoi(durationText)
	if err != nil || duration < 15 || duration > 480 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Неверная длительность. Введите число от 15 до 480 минут:",
		})
		return
	}

	subjectIDRaw, ok := h.stateManager.GetData(telegramID, "subject_id")
	if !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка: предмет не найден",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	subjectID, ok := subjectIDRaw.(int64)
	if !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка: неверный формат",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	user, err := h.userService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка авторизации",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	subject, err := h.teacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Предмет не найден",
		})
		h.stateManager.ClearState(telegramID)
		return
	}

	subject.Duration = duration
	err = h.teacherService.UpdateSubject(ctx, user.ID, subject)
	if err != nil {
		h.logger.Error("Failed to update subject duration", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Не удалось обновить длительность",
		})
		return
	}

	h.stateManager.ClearState(telegramID)

	// Показываем экран редактирования предмета
	h.showEditSubjectScreen(ctx, b, update.Message.Chat.ID, subjectID)
}
