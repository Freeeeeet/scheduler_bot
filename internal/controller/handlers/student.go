package handlers

import (
	"context"
	"fmt"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks"
	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// HandleSubjects обрабатывает команду /subjects
func (h *Handlers) HandleSubjects(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	h.logger.Info("HandleSubjects called",
		zap.Int64("user_id", update.Message.From.ID))

	// Получаем все активные предметы
	subjects, err := h.teacherService.GetAllActiveSubjects(ctx)
	if err != nil {
		h.logger.Error("Failed to get subjects", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Не удалось загрузить список предметов.",
		})
		return
	}

	h.logger.Info("Retrieved active subjects", zap.Int("count", len(subjects)))

	if len(subjects) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "📚 Пока нет доступных предметов.\n\nСтаньте первым учителем: /becometeacher",
		})
		return
	}

	text := "📚 Доступные предметы:\n\n"
	var buttons [][]models.InlineKeyboardButton
	hasApprovalRequired := false

	for i, subject := range subjects {
		approvalText := ""
		if subject.RequiresBookingApproval {
			approvalText = " ⏳"
			hasApprovalRequired = true
		}

		text += fmt.Sprintf(
			"%d. %s%s\n"+
				"   💰 Цена: %s\n"+
				"   ⏱ Длительность: %d мин\n"+
				"   📝 %s\n\n",
			i+1,
			subject.Name,
			approvalText,
			FormatPrice(subject.Price),
			subject.Duration,
			subject.Description,
		)

		// Добавляем кнопку для просмотра деталей
		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: fmt.Sprintf("📖 %s", subject.Name), CallbackData: fmt.Sprintf("view_subject:%d", subject.ID)},
		})
	}

	if hasApprovalRequired {
		text += "\n⏳ - требуется одобрение учителя"
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        text,
		ReplyMarkup: keyboard,
	})
}

// HandleMyBookings обрабатывает команду /mybookings (улучшенная версия с кнопками)
func (h *Handlers) HandleMyBookings(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	telegramID := update.Message.From.ID

	h.logger.Info("HandleMyBookings called",
		zap.Int64("telegram_id", telegramID))

	// Получаем пользователя
	user, err := h.userService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		h.logger.Error("User not found",
			zap.Int64("telegram_id", telegramID),
			zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Пользователь не найден. Используйте /start для регистрации.",
		})
		return
	}

	// Если пользователь учитель, показываем две секции
	if user.IsTeacher {
		h.logger.Info("Showing teacher bookings view",
			zap.Int64("user_id", user.ID))
		h.handleTeacherBookings(ctx, b, update, user)
		return
	}

	// Обычный студент - показываем только свои записи
	h.logger.Info("Showing student bookings view",
		zap.Int64("user_id", user.ID))
	h.handleStudentBookings(ctx, b, update, user)
}

// handleStudentBookings показывает записи студента
func (h *Handlers) handleStudentBookings(ctx context.Context, b *bot.Bot, update *models.Update, user *model.User) {
	// Получаем бронирования
	bookings, err := h.bookingService.GetStudentBookings(ctx, user.ID)
	if err != nil {
		h.logger.Error("Failed to get bookings", zap.Error(err))
		h.sendError(ctx, b, update.Message.Chat.ID, "❌ Не удалось загрузить ваши записи.")
		return
	}

	if len(bookings) == 0 {
		// Если нет записей, показываем кнопку для записи
		keyboard := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "📚 Посмотреть предметы", CallbackData: callbacks.BookAnother},
				},
			},
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        "📅 У вас пока нет записей на занятия.\n\nПосмотрите доступные предметы и запишитесь!",
			ReplyMarkup: keyboard,
		})
		return
	}

	// Отправляем каждую запись отдельным сообщением с кнопками
	for _, booking := range bookings {
		text := FormatBooking(booking)

		// Добавляем кнопку отмены только для активных записей
		if booking.Status == model.BookingStatusConfirmed || booking.Status == model.BookingStatusPending {
			keyboard := &models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{
						{Text: fmt.Sprintf("❌ Отменить запись #%d", booking.ID), CallbackData: fmt.Sprintf("%s%d", callbacks.CancelBooking, booking.ID)},
					},
				},
			}

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      update.Message.Chat.ID,
				Text:        text,
				ReplyMarkup: keyboard,
			})
		} else {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   text,
			})
		}
	}

	// В конце добавляем кнопку для новой записи
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "➕ Записаться на занятие", CallbackData: callbacks.BookAnother},
			},
		},
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        "───────────────",
		ReplyMarkup: keyboard,
	})
}

// handleTeacherBookings показывает записи учителя (как студента) + запросы на одобрение
func (h *Handlers) handleTeacherBookings(ctx context.Context, b *bot.Bot, update *models.Update, user *model.User) {
	// 1. Показываем записи учителя как студента
	studentBookings, err := h.bookingService.GetStudentBookings(ctx, user.ID)
	if err != nil {
		h.logger.Error("Failed to get student bookings", zap.Error(err))
	}

	// 2. Получаем pending запросы для одобрения
	pendingBookings, err := h.bookingService.GetPendingBookings(ctx, user.ID)
	if err != nil {
		h.logger.Error("Failed to get pending bookings", zap.Error(err))
	}

	// Если нет ни записей, ни запросов
	if len(studentBookings) == 0 && len(pendingBookings) == 0 {
		keyboard := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "📚 Записаться на занятие", CallbackData: callbacks.BookAnother},
				},
			},
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        "📅 У вас нет записей и нет запросов на одобрение.\n\nВы можете записаться на занятия к другим учителям!",
			ReplyMarkup: keyboard,
		})
		return
	}

	// Отправляем заголовок
	headerText := "📋 **Ваши записи и запросы**\n\n"
	if len(pendingBookings) > 0 {
		headerText += fmt.Sprintf("⏳ **Запросы на одобрение: %d**\n", len(pendingBookings))
	}
	if len(studentBookings) > 0 {
		headerText += fmt.Sprintf("👤 **Ваши записи как студент: %d**\n", len(studentBookings))
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      headerText,
		ParseMode: models.ParseModeMarkdown,
	})

	// Показываем pending запросы (для одобрения учителем)
	if len(pendingBookings) > 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "⏳ **Запросы на одобрение:**",
			ParseMode: models.ParseModeMarkdown,
		})

		for _, booking := range pendingBookings {
			student, err := h.userService.GetByID(ctx, booking.StudentID)
			if err != nil {
				h.logger.Warn("Failed to get student info",
					zap.Int64("student_id", booking.StudentID),
					zap.Error(err),
				)
			}
			studentName := "Неизвестный студент"
			if student != nil {
				studentName = student.FirstName
			}

			text := fmt.Sprintf(
				"⏳ Запрос #%d\n\n"+
					"👤 Студент: %s\n"+
					"📅 Создан: %s",
				booking.ID,
				studentName,
				booking.CreatedAt.Format("02.01.2006 15:04"),
			)

			keyboard := &models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{
						{Text: "✅ Одобрить", CallbackData: fmt.Sprintf("%s%d", callbacks.ApproveBooking, booking.ID)},
						{Text: "❌ Отклонить", CallbackData: fmt.Sprintf("%s%d", callbacks.RejectBooking, booking.ID)},
					},
				},
			}

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      update.Message.Chat.ID,
				Text:        text,
				ReplyMarkup: keyboard,
			})
		}
	}

	// Показываем записи учителя как студента
	if len(studentBookings) > 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "\n👤 **Ваши записи как студент:**",
			ParseMode: models.ParseModeMarkdown,
		})

		for _, booking := range studentBookings {
			text := FormatBooking(booking)

			// Добавляем кнопку отмены только для активных записей
			if booking.Status == model.BookingStatusConfirmed || booking.Status == model.BookingStatusPending {
				keyboard := &models.InlineKeyboardMarkup{
					InlineKeyboard: [][]models.InlineKeyboardButton{
						{
							{Text: fmt.Sprintf("❌ Отменить запись #%d", booking.ID), CallbackData: fmt.Sprintf("%s%d", callbacks.CancelBooking, booking.ID)},
						},
					},
				}

				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:      update.Message.Chat.ID,
					Text:        text,
					ReplyMarkup: keyboard,
				})
			} else {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   text,
				})
			}
		}
	}
}
