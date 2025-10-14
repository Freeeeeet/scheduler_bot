package student

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

// HandleMyTeachers показывает список учителей студента
func HandleMyTeachers(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем учителей студента
	teachers, err := h.AccessService.GetMyTeachers(ctx, user.ID)
	if err != nil {
		h.Logger.Error("Failed to get student teachers", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка при загрузке учителей")
		return
	}

	// Формируем текст и клавиатуру
	text := "🎓 *Мои учителя*\n\n"
	if len(teachers) == 0 {
		text += "У вас пока нет доступа к учителям.\n\n"
		text += "💡 Используйте код приглашения или отправьте заявку учителю."
	} else {
		text += fmt.Sprintf("Учителя, к которым у вас есть доступ (%d):\n\n", len(teachers))
	}

	kb := keyboard.NewBuilder()

	// Добавляем кнопки учителей
	for _, teacher := range teachers {
		name := teacher.FirstName
		if teacher.LastName != "" {
			name += " " + teacher.LastName
		}
		kb.Row(keyboard.Button(
			fmt.Sprintf("👤 %s", name),
			fmt.Sprintf("teacher_profile:%d", teacher.ID),
		))
	}

	// Навигация
	kb.Row(keyboard.BackButton("subjects_menu"))

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

// HandleFindTeacher показывает варианты поиска учителя
func HandleFindTeacher(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	text := "🔍 *Найти учителя*\n\n" +
		"Выберите способ поиска:\n\n" +
		"🎟️ *Код приглашения* - если у вас есть код от учителя\n" +
		"📝 *Отправить заявку* - запросить доступ у приватного учителя\n"

	kb := keyboard.NewBuilder()
	kb.Row(keyboard.Button("🎟️ У меня есть код", "enter_invite_code"))
	kb.Row(keyboard.Button("📝 Отправить заявку", "send_access_request"))
	kb.Row(keyboard.BackButton("subjects_menu"))

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

// HandleEnterInviteCode показывает информацию о вводе кода
func HandleEnterInviteCode(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	text := "🎟️ *Код приглашения*\n\n" +
		"Чтобы использовать код приглашения:\n\n" +
		"1. Получите код от вашего учителя\n" +
		"2. Напишите боту код одним сообщением\n\n" +
		"Пример кода: `ABC12XYZ`\n\n" +
		"После отправки кода, бот автоматически предоставит вам доступ к учителю.\n\n" +
		"_Примечание: В текущей версии нужно использовать веб-форму или API для ввода кода._"

	kb := keyboard.NewBuilder()
	kb.Row(keyboard.Button("🔙 Назад", "find_teacher"))

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

// HandleSendAccessRequest показывает форму для отправки заявки
func HandleSendAccessRequest(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	text := "📝 *Отправить заявку учителю*\n\n" +
		"Чтобы получить доступ к приватному учителю:\n\n" +
		"1. Узнайте Telegram username учителя\n" +
		"2. Напишите боту сообщение с username\n" +
		"3. Опционально добавьте сообщение для учителя\n\n" +
		"Пример: `@username_teacher`\n\n" +
		"После отправки, учитель получит вашу заявку и сможет её одобрить или отклонить.\n\n" +
		"_Примечание: Функция отправки заявок будет доступна в следующей версии._"

	kb := keyboard.NewBuilder()
	kb.Row(keyboard.Button("🔙 Назад", "find_teacher"))

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

// ProcessInviteCode обрабатывает введенный код приглашения
func ProcessInviteCode(ctx context.Context, b *bot.Bot, message *models.Message, h *callbacktypes.Handler) {
	telegramID := message.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: message.Chat.ID,
			Text:   "❌ Пользователь не найден",
		})
		return
	}

	code := message.Text

	// Используем код
	err = h.AccessService.UseInviteCode(ctx, user.ID, code)
	if err != nil {
		h.Logger.Error("Failed to use invite code",
			zap.String("code", code),
			zap.Int64("user_id", user.ID),
			zap.Error(err))

		errMsg := "❌ Не удалось использовать код.\n\n"
		if err.Error() == "invite code not found" {
			errMsg += "Код не найден. Проверьте правильность ввода."
		} else if err.Error() == "invite code is not valid" {
			errMsg += "Код недействителен (истек или исчерпан лимит использований)."
		} else if err.Error() == "access already granted" {
			errMsg += "У вас уже есть доступ к этому учителю."
		} else {
			errMsg += "Произошла ошибка. Попробуйте позже."
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: message.Chat.ID,
			Text:   errMsg,
		})
		return
	}

	// Получаем информацию об учителе
	inviteCode, _ := h.InviteCodeRepo.GetByCode(ctx, code)
	var teacherName string
	if inviteCode != nil {
		teacher, _ := h.UserService.GetByID(ctx, inviteCode.TeacherID)
		if teacher != nil {
			teacherName = teacher.FirstName
			if teacher.LastName != "" {
				teacherName += " " + teacher.LastName
			}
		}
	}

	// Очищаем состояние
	h.StateManager.ClearState(telegramID)

	text := "✅ *Доступ получен!*\n\n"
	if teacherName != "" {
		text += fmt.Sprintf("Учитель *%s* добавлен в 'Мои учителя'.\n\n", teacherName)
	} else {
		text += "Учитель добавлен в 'Мои учителя'.\n\n"
	}
	text += "Теперь вы можете просматривать предметы и записываться на занятия."

	kb := keyboard.NewBuilder()
	kb.Row(keyboard.Button("👤 Мои учителя", "my_teachers"))
	kb.Row(keyboard.BackButton("subjects_menu"))

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      message.Chat.ID,
		Text:        text,
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: kb.Build(),
	})
}

// HandleMyRequests показывает заявки студента
func HandleMyRequests(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем заявки студента
	requests, err := h.AccessService.GetStudentRequests(ctx, user.ID)
	if err != nil {
		h.Logger.Error("Failed to get student requests", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка при загрузке заявок")
		return
	}

	// Группируем заявки по статусам
	var pending, approved, rejected int
	for _, req := range requests {
		switch req.Status {
		case "pending":
			pending++
		case "approved":
			approved++
		case "rejected":
			rejected++
		}
	}

	text := "📋 *Мои заявки на доступ*\n\n"
	if len(requests) == 0 {
		text += "У вас пока нет заявок."
	} else {
		text += fmt.Sprintf("⏳ Ожидают ответа: %d\n", pending)
		text += fmt.Sprintf("✅ Одобрены: %d\n", approved)
		text += fmt.Sprintf("❌ Отклонены: %d\n\n", rejected)

		// Показываем детали pending заявок
		if pending > 0 {
			text += "*Ожидают ответа:*\n"
			for _, req := range requests {
				if req.Status == "pending" {
					teacher, _ := h.UserService.GetByID(ctx, req.TeacherID)
					if teacher != nil {
						teacherName := teacher.FirstName
						if teacher.LastName != "" {
							teacherName += " " + teacher.LastName
						}
						text += fmt.Sprintf("• %s - отправлено %s\n",
							teacherName,
							req.CreatedAt.Format("02.01.2006"))
					}
				}
			}
		}
	}

	kb := keyboard.NewBuilder()
	kb.Row(keyboard.BackButton("subjects_menu"))

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
