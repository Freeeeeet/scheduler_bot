package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// HandleBecomeTeacher обрабатывает команду /becometeacher
func (h *Handlers) HandleBecomeTeacher(ctx context.Context, b *bot.Bot, update *models.Update) {
	user, ok := h.requireUser(ctx, b, update)
	if !ok {
		return
	}

	if user.IsTeacher {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "✅ Вы уже являетесь учителем!\n\nИспользуйте:\n/mysubjects - Управление предметами\n/myschedule - Расписание",
		})
		return
	}

	// Создаём inline клавиатуру с подтверждением
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "✅ Да, стать учителем", CallbackData: callbacks.BecomeTeacher},
			},
			{
				{Text: "❌ Отмена", CallbackData: callbacks.CancelBecomeTeacher},
			},
		},
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: "🎓 Стать учителем\n\n" +
			"Как учитель вы сможете:\n" +
			"• Создавать предметы для преподавания\n" +
			"• Управлять своим расписанием\n" +
			"• Принимать записи от студентов\n" +
			"• Получать уведомления о новых записях\n\n" +
			"⚠️ Обратите внимание: вы также сможете оставаться студентом и записываться на занятия к другим учителям.\n\n" +
			"Продолжить?",
		ReplyMarkup: keyboard,
	})
}

// HandleMySubjects обрабатывает команду /mysubjects
// Если передан messageID, редактирует существующее сообщение, иначе отправляет новое
func (h *Handlers) HandleMySubjects(ctx context.Context, b *bot.Bot, update *models.Update, messageID ...int) {
	user, ok := h.requireTeacher(ctx, b, update)
	if !ok {
		return
	}

	h.logger.Info("HandleMySubjects called",
		zap.Int64("user_id", user.ID),
		zap.Int64("telegram_id", user.TelegramID))

	var chatID int64
	if update.Message != nil {
		chatID = update.Message.Chat.ID
	} else if update.CallbackQuery != nil {
		// Получаем ChatID из CallbackQuery через helper
		msg := common.GetMessageFromCallback(update.CallbackQuery)
		if msg != nil {
			chatID = msg.Chat.ID
		} else {
			h.logger.Error("Cannot determine chat ID from CallbackQuery")
			return
		}
	} else {
		h.logger.Error("Cannot determine chat ID")
		return
	}

	// Получаем предметы учителя
	subjects, err := h.teacherService.GetTeacherSubjects(ctx, user.ID)
	if err != nil {
		h.logger.Error("Failed to get teacher subjects", zap.Error(err))
		if len(messageID) > 0 && messageID[0] > 0 {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: messageID[0],
				Text:      "❌ Не удалось загрузить ваши предметы.",
			})
		} else {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "❌ Не удалось загрузить ваши предметы.",
			})
		}
		return
	}

	h.logger.Info("Retrieved teacher subjects",
		zap.Int64("teacher_id", user.ID),
		zap.Int("count", len(subjects)))

	if len(subjects) == 0 {
		keyboard := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "➕ Создать первый предмет", CallbackData: "create_first_subject"},
				},
			},
		}

		text := "📚 У вас пока нет предметов.\n\nСоздайте свой первый предмет для преподавания!"
		if len(messageID) > 0 && messageID[0] > 0 {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:      chatID,
				MessageID:   messageID[0],
				Text:        text,
				ReplyMarkup: keyboard,
			})
		} else {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      chatID,
				Text:        text,
				ReplyMarkup: keyboard,
			})
		}
		return
	}

	// Используем билдер экрана списка предметов
	page := 0 // первая страница по умолчанию
	text, keyboard := common.BuildSubjectsListScreen(subjects, page)

	// Добавляем дополнительную кнопку настроек доступа
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, []models.InlineKeyboardButton{
		{Text: "⚙️ Настройки доступа", CallbackData: "teacher_settings"},
	})

	if len(messageID) > 0 && messageID[0] > 0 {
		// Редактируем существующее сообщение
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID[0],
			Text:        text,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: keyboard,
		})
	} else {
		// Отправляем новое сообщение
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        text,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: keyboard,
		})
	}
}

// HandleMySchedule обрабатывает команду /myschedule
func (h *Handlers) HandleMySchedule(ctx context.Context, b *bot.Bot, update *models.Update) {
	user, ok := h.requireTeacher(ctx, b, update)
	if !ok {
		return
	}

	h.logger.Info("HandleMySchedule called",
		zap.Int64("user_id", user.ID),
		zap.Int64("telegram_id", user.TelegramID))

	// Получаем расписание на следующие 7 дней
	now := time.Now()
	endDate := now.AddDate(0, 0, 7)

	slots, err := h.teacherService.GetTeacherSchedule(ctx, user.ID, now, endDate)
	if err != nil {
		h.logger.Error("Failed to get teacher schedule", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Не удалось загрузить расписание.",
		})
		return
	}

	h.logger.Info("Retrieved teacher schedule",
		zap.Int64("teacher_id", user.ID),
		zap.Int("slots_count", len(slots)))

	// Подсчитываем статистику
	totalSlots := len(slots)
	bookedSlots := 0
	for _, slot := range slots {
		if slot.Status == "booked" {
			bookedSlots++
		}
	}
	freeSlots := totalSlots - bookedSlots

	var text string
	var buttons [][]models.InlineKeyboardButton

	if totalSlots == 0 {
		// Получаем предметы для создания слотов
		subjects, _ := h.teacherService.GetTeacherSubjects(ctx, user.ID)

		if len(subjects) == 0 {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "📅 У вас пока нет слотов в расписании.\n\n💡 Сначала создайте предмет через /mysubjects, затем добавьте временные слоты.",
			})
			return
		}

		text = "📅 <b>Моё расписание</b>\n\n" +
			"У вас пока нет слотов на ближайшие 7 дней.\n\n" +
			"Создайте слоты через управление расписанием."

		buttons = [][]models.InlineKeyboardButton{
			{
				{Text: "📊 Управление расписанием", CallbackData: "view_schedule"},
			},
		}
	} else {
		text = fmt.Sprintf(
			"📅 <b>Моё расписание</b>\n\n"+
				"📊 <b>Статистика на 7 дней:</b>\n"+
				"📋 Всего занятий: %d\n"+
				"👥 Записались учеников: %d\n"+
				"🟢 Свободных слотов: %d\n\n"+
				"Выберите действие:",
			totalSlots,
			bookedSlots,
			freeSlots,
		)

		buttons = [][]models.InlineKeyboardButton{
			{
				{Text: "📊 Управление расписанием", CallbackData: "view_schedule"},
			},
			{
				{Text: "📅 Просмотреть расписание", CallbackData: "view_schedule_weeks:0"},
			},
		}
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})
}
