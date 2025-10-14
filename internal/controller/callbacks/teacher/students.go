package teacher

import (
	"context"
	"fmt"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common/keyboard"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// HandleViewAccessRequests показывает заявки на доступ
func HandleViewAccessRequests(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil || !user.IsTeacher {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Доступ запрещен")
		return
	}

	// Получаем pending заявки
	requests, err := h.AccessService.GetPendingRequests(ctx, user.ID)
	if err != nil {
		h.Logger.Error("Failed to get pending requests", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка при загрузке заявок")
		return
	}

	// Формируем текст
	text := fmt.Sprintf("📩 *Заявки на доступ* (%d)\n\n", len(requests))

	if len(requests) == 0 {
		text += "Новых заявок нет."
	} else {
		for i, req := range requests {
			student, _ := h.UserService.GetByID(ctx, req.StudentID)
			if student != nil {
				studentName := student.FirstName
				if student.LastName != "" {
					studentName += " " + student.LastName
				}

				text += fmt.Sprintf("*%d. %s*", i+1, studentName)
				if student.Username != "" {
					text += fmt.Sprintf(" (@%s)", student.Username)
				}
				text += "\n"

				text += fmt.Sprintf("📅 Отправлена: %s\n", req.CreatedAt.Format("02.01.2006 15:04"))

				if req.Message != "" {
					text += fmt.Sprintf("💬 Сообщение:\n_%s_\n", req.Message)
				} else {
					text += "💬 _Сообщение не указано_\n"
				}

				text += "\n"
			}
		}
	}

	// Формируем клавиатуру
	kb := keyboard.NewBuilder()

	// Кнопки для каждой заявки (первые 5)
	for i, req := range requests {
		if i >= 5 {
			break
		}

		student, _ := h.UserService.GetByID(ctx, req.StudentID)
		if student != nil {
			studentName := student.FirstName
			if len(studentName) > 15 {
				studentName = studentName[:15] + "..."
			}

			kb.AddRow([]models.InlineKeyboardButton{
				keyboard.Button(fmt.Sprintf("%d. %s", i+1, studentName), "noop"),
				keyboard.Button("✅", fmt.Sprintf("approve_request:%d", req.ID)),
				keyboard.Button("❌", fmt.Sprintf("reject_request:%d", req.ID)),
			})
		}
	}

	kb.Row(keyboard.BackButton("teacher_settings"))

	common.AnswerCallback(ctx, b, callback.ID, "")
	msg := common.GetMessageFromCallback(callback)
	if msg != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      msg.Chat.ID,
			MessageID:   msg.ID,
			Text:        text,
			ParseMode:   models.ParseModeMarkdown,
			ReplyMarkup: kb.Build(),
		})
	}
}

// HandleApproveAccessRequest одобряет заявку на доступ
func HandleApproveAccessRequest(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil || !user.IsTeacher {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Доступ запрещен")
		return
	}

	requestID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	// Получаем заявку для уведомления студента
	request, _ := h.AccessRequestRepo.GetByID(ctx, requestID)

	// Одобряем заявку
	err = h.AccessService.ApproveAccessRequest(ctx, user.ID, requestID, "Добро пожаловать!")
	if err != nil {
		h.Logger.Error("Failed to approve access request",
			zap.Int64("request_id", requestID),
			zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось одобрить заявку")
		return
	}

	// Уведомляем студента
	if request != nil {
		student, _ := h.UserService.GetByID(ctx, request.StudentID)
		if student != nil {
			teacherName := user.FirstName
			if user.LastName != "" {
				teacherName += " " + user.LastName
			}

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: student.TelegramID,
				Text: fmt.Sprintf(
					"✅ *Заявка одобрена!*\n\n"+
						"Учитель *%s* одобрил вашу заявку на доступ.\n\n"+
						"💬 _Добро пожаловать!_\n\n"+
						"Теперь вы можете просматривать предметы и записываться на занятия.",
					teacherName,
				),
				ParseMode: models.ParseModeMarkdown,
			})
		}
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Заявка одобрена")
	HandleViewAccessRequests(ctx, b, callback, h)
}

// HandleRejectAccessRequest отклоняет заявку на доступ
func HandleRejectAccessRequest(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil || !user.IsTeacher {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Доступ запрещен")
		return
	}

	requestID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	// Получаем заявку для уведомления студента
	request, _ := h.AccessRequestRepo.GetByID(ctx, requestID)

	// Отклоняем заявку
	err = h.AccessService.RejectAccessRequest(ctx, user.ID, requestID, "Извините, сейчас не могу принять новых студентов.")
	if err != nil {
		h.Logger.Error("Failed to reject access request",
			zap.Int64("request_id", requestID),
			zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось отклонить заявку")
		return
	}

	// Уведомляем студента
	if request != nil {
		student, _ := h.UserService.GetByID(ctx, request.StudentID)
		if student != nil {
			teacherName := user.FirstName
			if user.LastName != "" {
				teacherName += " " + user.LastName
			}

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: student.TelegramID,
				Text: fmt.Sprintf(
					"❌ *Заявка отклонена*\n\n"+
						"Учитель *%s* отклонил вашу заявку на доступ.\n\n"+
						"💬 _Извините, сейчас не могу принять новых студентов._",
					teacherName,
				),
				ParseMode: models.ParseModeMarkdown,
			})
		}
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Заявка отклонена")
	HandleViewAccessRequests(ctx, b, callback, h)
}

// HandleViewMyStudents показывает список студентов учителя
func HandleViewMyStudents(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil || !user.IsTeacher {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Доступ запрещен")
		return
	}

	// Получаем студентов
	students, err := h.AccessService.GetMyStudents(ctx, user.ID)
	if err != nil {
		h.Logger.Error("Failed to get teacher students", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка при загрузке студентов")
		return
	}

	// Формируем текст
	text := fmt.Sprintf("👥 *Мои студенты* (%d)\n\n", len(students))

	if len(students) == 0 {
		text += "У вас пока нет студентов."
	} else {
		for i, student := range students {
			if i >= 10 { // Показываем первых 10
				text += fmt.Sprintf("\n_...и ещё %d студентов_", len(students)-10)
				break
			}

			studentName := student.FirstName
			if student.LastName != "" {
				studentName += " " + student.LastName
			}

			text += fmt.Sprintf("%d. *%s*", i+1, studentName)
			if student.Username != "" {
				text += fmt.Sprintf(" (@%s)", student.Username)
			}
			text += "\n"

			// Получаем информацию о доступе
			accessInfo, _ := h.AccessRepo.GetAccessInfo(ctx, student.ID, user.ID)
			if accessInfo != nil {
				var accessTypeText string
				switch accessInfo.AccessType {
				case "invited":
					accessTypeText = "🎟️ по коду"
				case "approved":
					accessTypeText = "✅ по заявке"
				case "subscribed":
					accessTypeText = "⭐ подписка"
				default:
					accessTypeText = accessInfo.AccessType
				}
				text += fmt.Sprintf("   Доступ: %s\n", accessTypeText)
				text += fmt.Sprintf("   Дата: %s\n", accessInfo.GrantedAt.Format("02.01.2006"))
			}

			text += "\n"
		}
	}

	// Формируем клавиатуру
	kb := keyboard.NewBuilder()

	// Кнопки для управления (первые 5 студентов)
	for i, student := range students {
		if i >= 5 {
			break
		}

		studentName := student.FirstName
		if len(studentName) > 20 {
			studentName = studentName[:20] + "..."
		}

		kb.AddRow([]models.InlineKeyboardButton{
			keyboard.Button(fmt.Sprintf("%d. %s", i+1, studentName), "noop"),
			keyboard.Button("❌ Отозвать", fmt.Sprintf("revoke_access:%d", student.ID)),
		})
	}

	kb.Row(keyboard.BackButton("teacher_settings"))

	common.AnswerCallback(ctx, b, callback.ID, "")
	msg := common.GetMessageFromCallback(callback)
	if msg != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      msg.Chat.ID,
			MessageID:   msg.ID,
			Text:        text,
			ParseMode:   models.ParseModeMarkdown,
			ReplyMarkup: kb.Build(),
		})
	}
}

// HandleRevokeStudentAccess отзывает доступ студента
func HandleRevokeStudentAccess(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil || !user.IsTeacher {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Доступ запрещен")
		return
	}

	studentID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	// Получаем студента для уведомления
	student, _ := h.UserService.GetByID(ctx, studentID)

	// Отзываем доступ
	err = h.AccessService.RevokeStudentAccess(ctx, user.ID, studentID)
	if err != nil {
		h.Logger.Error("Failed to revoke student access",
			zap.Int64("student_id", studentID),
			zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось отозвать доступ")
		return
	}

	// Уведомляем студента
	if student != nil {
		teacherName := user.FirstName
		if user.LastName != "" {
			teacherName += " " + user.LastName
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: student.TelegramID,
			Text: fmt.Sprintf(
				"⚠️ *Доступ отозван*\n\n"+
					"Учитель *%s* отозвал ваш доступ к своим предметам.\n\n"+
					"Если это ошибка, свяжитесь с учителем напрямую.",
				teacherName,
			),
			ParseMode: models.ParseModeMarkdown,
		})
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Доступ отозван")
	HandleViewMyStudents(ctx, b, callback, h)
}
