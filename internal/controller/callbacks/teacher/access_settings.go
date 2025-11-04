package teacher

import (
	"context"
	"fmt"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common/keyboard"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// HandleTeacherSettings показывает настройки учителя
func HandleTeacherSettings(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil || !user.IsTeacher {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Доступ запрещен")
		return
	}

	// Получаем статистику
	studentsCount, _ := h.AccessService.CountStudents(ctx, user.ID)
	pendingRequests, _ := h.AccessService.CountPendingRequests(ctx, user.ID)
	activeCodes, _ := h.InviteCodeRepo.CountActiveCodesByTeacher(ctx, user.ID)

	// Формируем текст
	text := "⚙️ *Настройки учителя*\n\n"

	text += "*Видимость профиля:*\n"
	if user.IsPublic {
		text += "✅ Публичный - любой студент может найти вас\n\n"
	} else {
		text += "🔒 Приватный - доступ только по приглашению\n\n"
	}

	text += "───────────────\n\n"
	text += fmt.Sprintf("👥 Мои студенты: *%d*\n", studentsCount)
	text += fmt.Sprintf("📩 Заявки на доступ: *%d* новых\n", pendingRequests)
	text += fmt.Sprintf("🎟️ Активных кодов: *%d*\n", activeCodes)

	// Формируем клавиатуру
	kb := keyboard.NewBuilder()

	if user.IsPublic {
		kb.Row(keyboard.Button("🔒 Сделать приватным", "toggle_public_status"))
	} else {
		kb.Row(keyboard.Button("🌍 Сделать публичным", "toggle_public_status"))
	}

	kb.Row(keyboard.Button(fmt.Sprintf("📩 Заявки (%d)", pendingRequests), "view_access_requests"))
	kb.Row(keyboard.Button(fmt.Sprintf("🎟️ Коды приглашения (%d)", activeCodes), "manage_invite_codes"))
	kb.Row(keyboard.Button(fmt.Sprintf("👥 Мои студенты (%d)", studentsCount), "view_my_students"))
	kb.Row(keyboard.BackButton("mysubjects"))

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка получения сообщения")
		return
	}

	common.AnswerCallback(ctx, b, callback.ID, "")
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: kb.Build(),
	})
}

// HandleTogglePublicStatus переключает публичность учителя
func HandleTogglePublicStatus(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil || !user.IsTeacher {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Доступ запрещен")
		return
	}

	// Переключаем статус
	newStatus := !user.IsPublic
	err = h.UserRepo.UpdatePublicStatus(ctx, user.ID, newStatus)
	if err != nil {
		h.Logger.Error("Failed to update public status", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка при обновлении")
		return
	}

	var alertText string
	if newStatus {
		alertText = "✅ Теперь вы публичный учитель"
	} else {
		alertText = "✅ Теперь вы приватный учитель"
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, alertText)

	// Обновляем отображение
	user.IsPublic = newStatus
	HandleTeacherSettings(ctx, b, callback, h)
}

// HandleManageInviteCodes показывает управление кодами приглашения
func HandleManageInviteCodes(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil || !user.IsTeacher {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Доступ запрещен")
		return
	}

	// Получаем коды
	codes, err := h.AccessService.GetTeacherInviteCodes(ctx, user.ID)
	if err != nil {
		h.Logger.Error("Failed to get invite codes", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка при загрузке кодов")
		return
	}

	// Группируем коды
	var activeCodes, inactiveCodes int
	for _, code := range codes {
		if code.IsActive && code.IsValid() {
			activeCodes++
		} else {
			inactiveCodes++
		}
	}

	// Формируем текст
	text := "🎟️ *Коды приглашения*\n\n"

	if activeCodes == 0 {
		text += "У вас нет активных кодов.\n\n"
	} else {
		text += fmt.Sprintf("*Активные коды (%d):*\n\n", activeCodes)

		count := 0
		for _, code := range codes {
			if code.IsActive && code.IsValid() {
				count++
				text += fmt.Sprintf("%d. `%s`\n", count, code.Code)

				// Использования
				if code.MaxUses != nil {
					text += fmt.Sprintf("   Использований: %d/%d\n", code.CurrentUses, *code.MaxUses)
				} else {
					text += fmt.Sprintf("   Использований: %d/∞\n", code.CurrentUses)
				}

				// Срок действия
				if code.ExpiresAt != nil {
					daysLeft := int(time.Until(*code.ExpiresAt).Hours() / 24)
					if daysLeft > 0 {
						text += fmt.Sprintf("   Истекает через: %d дн.\n", daysLeft)
					} else {
						text += "   Истекает: сегодня\n"
					}
				} else {
					text += "   Срок: бессрочный\n"
				}

				text += "\n"
			}
		}
	}

	if inactiveCodes > 0 {
		text += fmt.Sprintf("\n_Неактивных кодов: %d_\n", inactiveCodes)
	}

	// Формируем клавиатуру
	kb := keyboard.NewBuilder()
	kb.Row(keyboard.Button("➕ Создать новый код", "create_invite_code"))

	// Кнопки для деактивации кодов (первые 5 активных)
	if activeCodes > 0 {
		count := 0
		for _, code := range codes {
			if code.IsActive && code.IsValid() && count < 5 {
				kb.Row(
					keyboard.Button(fmt.Sprintf("📋 %s", code.Code), "noop"),
					keyboard.Button("❌", fmt.Sprintf("deactivate_code:%d", code.ID)),
				)
				count++
			}
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

// HandleCreateInviteCode создает новый код приглашения
func HandleCreateInviteCode(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil || !user.IsTeacher {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Доступ запрещен")
		return
	}

	// Создаем код с настройками по умолчанию (без ограничений)
	inviteCode, err := h.AccessService.CreateInviteCode(ctx, user.ID, nil, nil)
	if err != nil {
		h.Logger.Error("Failed to create invite code", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось создать код")
		return
	}

	text := "✅ *Код создан!*\n\n"
	text += fmt.Sprintf("Ваш код: `%s`\n\n", inviteCode.Code)
	text += "Отправьте этот код студентам для предоставления доступа.\n\n"
	text += "⚙️ Код создан с настройками:\n"
	text += "• Без ограничений по количеству\n"
	text += "• Бессрочный"

	kb := keyboard.NewBuilder()
	kb.Row(keyboard.Button("🔙 К кодам", "manage_invite_codes"))

	common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Код создан")

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

// HandleDeactivateInviteCode деактивирует код приглашения
func HandleDeactivateInviteCode(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil || !user.IsTeacher {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Доступ запрещен")
		return
	}

	codeID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	// Деактивируем код
	err = h.AccessService.DeactivateInviteCode(ctx, user.ID, codeID)
	if err != nil {
		h.Logger.Error("Failed to deactivate invite code", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Не удалось деактивировать код")
		return
	}

	common.AnswerCallbackAlert(ctx, b, callback.ID, "✅ Код деактивирован")
	HandleManageInviteCodes(ctx, b, callback, h)
}
