package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks"
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
func (h *Handlers) HandleMySubjects(ctx context.Context, b *bot.Bot, update *models.Update) {
	user, ok := h.requireTeacher(ctx, b, update)
	if !ok {
		return
	}

	h.logger.Info("HandleMySubjects called",
		zap.Int64("user_id", user.ID),
		zap.Int64("telegram_id", user.TelegramID))

	// Получаем предметы учителя
	subjects, err := h.teacherService.GetTeacherSubjects(ctx, user.ID)
	if err != nil {
		h.logger.Error("Failed to get teacher subjects", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Не удалось загрузить ваши предметы.",
		})
		return
	}

	h.logger.Info("Retrieved teacher subjects",
		zap.Int64("teacher_id", user.ID),
		zap.Int("count", len(subjects)))

	if len(subjects) == 0 {
		keyboard := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "➕ Создать первый предмет", CallbackData: callbacks.CreateFirstSubject},
				},
			},
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        "📚 У вас пока нет предметов.\n\nСоздайте свой первый предмет для преподавания!",
			ReplyMarkup: keyboard,
		})
		return
	}

	text := "📚 Ваши предметы:\n\n"
	var buttons [][]models.InlineKeyboardButton

	for i, subject := range subjects {
		statusEmoji := "✅"
		statusText := "Активен"

		if !subject.IsActive {
			statusEmoji = "⏸"
			statusText = "Неактивен"
		}

		text += fmt.Sprintf(
			"%d. %s %s\n"+
				"   💰 Цена: %s\n"+
				"   ⏱ Длительность: %d мин\n"+
				"   📝 %s\n"+
				"   Статус: %s\n\n",
			i+1,
			statusEmoji,
			subject.Name,
			FormatPrice(subject.Price),
			subject.Duration,
			subject.Description,
			statusText,
		)

		// Кнопки для каждого предмета
		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: fmt.Sprintf("📝 %s", subject.Name), CallbackData: fmt.Sprintf("%s%d", callbacks.ViewSubject, subject.ID)},
			{Text: "✏️", CallbackData: fmt.Sprintf("%s%d", callbacks.EditSubject, subject.ID)},
			{Text: statusEmoji, CallbackData: fmt.Sprintf("%s%d", callbacks.ToggleSubject, subject.ID)},
		})
	}

	// Добавляем подсказку о создании слотов
	text += "\n💡 Совет: Создайте временные слоты через /myschedule чтобы студенты могли записываться!\n\n"

	// Кнопка создать новый предмет
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "➕ Создать новый предмет", CallbackData: callbacks.CreateFirstSubject},
	})

	// Кнопка для быстрого перехода к расписанию
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "📅 Управление расписанием", CallbackData: callbacks.ViewSchedule},
	})

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        text,
		ReplyMarkup: keyboard,
	})
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

	if len(slots) == 0 {
		// Получаем предметы для создания слотов
		subjects, _ := h.teacherService.GetTeacherSubjects(ctx, user.ID)

		h.logger.Info("No slots found, checking subjects",
			zap.Int("subjects_count", len(subjects)))

		if len(subjects) == 0 {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "📅 У вас пока нет слотов в расписании.\n\n💡 Сначала создайте предмет через /mysubjects, затем добавьте временные слоты.",
			})
			return
		}

		// Если есть предметы, показываем кнопки для создания слотов
		var buttons [][]models.InlineKeyboardButton

		text := "📅 У вас пока нет слотов в расписании на ближайшие 7 дней.\n\n" +
			"📚 Ваши активные предметы:\n\n"

		activeCount := 0
		for i, subject := range subjects {
			if !subject.IsActive {
				continue
			}
			activeCount++
			text += fmt.Sprintf("%d. %s (%.2f ₽, %d мин)\n", i+1, subject.Name, float64(subject.Price)/100, subject.Duration)
			buttons = append(buttons, []models.InlineKeyboardButton{
				{Text: fmt.Sprintf("➕ Добавить слоты для «%s»", subject.Name), CallbackData: fmt.Sprintf("create_slots:%d", subject.ID)},
			})
		}

		h.logger.Info("Active subjects for slot creation",
			zap.Int("active_count", activeCount))

		if len(buttons) == 0 {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "📅 У вас пока нет слотов в расписании.\n\n💡 Активируйте хотя бы один предмет через /mysubjects, чтобы создать слоты.",
			})
			return
		}

		text += "\n💡 Выберите предмет, для которого хотите создать временные слоты:"

		keyboard := &models.InlineKeyboardMarkup{
			InlineKeyboard: buttons,
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        text,
			ReplyMarkup: keyboard,
		})
		return
	}

	text := "📅 Ваше расписание (7 дней):\n\n"
	for i, slot := range slots {
		statusEmoji := "🟢"
		statusText := "Свободен"
		switch slot.Status {
		case "booked":
			statusEmoji = "🔴"
			statusText = "Забронирован"
		case "canceled":
			statusEmoji = "⚫️"
			statusText = "Отменён"
		}

		text += fmt.Sprintf(
			"%d. %s %s\n"+
				"   📅 %s\n"+
				"   🕐 %s - %s\n"+
				"   Статус: %s\n\n",
			i+1,
			statusEmoji,
			slot.StartTime.Format("02.01.2006"),
			slot.StartTime.Weekday(),
			slot.StartTime.Format("15:04"),
			slot.EndTime.Format("15:04"),
			statusText,
		)
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   text,
	})
}
