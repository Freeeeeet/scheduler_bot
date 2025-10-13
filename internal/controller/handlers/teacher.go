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

	// Пагинация: показываем по 10 предметов на странице
	const pageSize = 10
	page := 0 // первая страница по умолчанию

	text := fmt.Sprintf("📚 Ваши предметы (всего: %d):\n\n", len(subjects))
	var buttons [][]models.InlineKeyboardButton

	// Вычисляем индексы для текущей страницы
	startIdx := page * pageSize
	endIdx := startIdx + pageSize
	if endIdx > len(subjects) {
		endIdx = len(subjects)
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

	// Кнопки пагинации
	totalPages := (len(subjects) + pageSize - 1) / pageSize
	if totalPages > 1 {
		var paginationButtons []models.InlineKeyboardButton

		// Кнопка "Предыдущая" только если не первая страница
		if page > 0 {
			paginationButtons = append(paginationButtons,
				models.InlineKeyboardButton{Text: "⬅️ Предыдущая", CallbackData: fmt.Sprintf("subjects_page:%d", page-1)})
		}

		// Показываем номер страницы
		paginationButtons = append(paginationButtons,
			models.InlineKeyboardButton{Text: fmt.Sprintf("📄 %d/%d", page+1, totalPages), CallbackData: "noop"})

		// Кнопка "Следующая" только если не последняя страница
		if page < totalPages-1 {
			paginationButtons = append(paginationButtons,
				models.InlineKeyboardButton{Text: "Следующая ➡️", CallbackData: fmt.Sprintf("subjects_page:%d", page+1)})
		}

		buttons = append(buttons, paginationButtons)
	}

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
