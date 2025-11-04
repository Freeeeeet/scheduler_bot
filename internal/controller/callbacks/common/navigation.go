package common

import (
	"context"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// ========================
// Common Navigation Handlers
// ========================
// These handlers manage common navigation actions used throughout the bot

// HandleBackToMain возвращает пользователя к главному меню
func HandleBackToMain(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	msg := GetMessageFromCallback(callback)
	if msg == nil {
		AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}
	telegramID := callback.From.ID

	// Очищаем состояние пользователя
	h.StateManager.ClearState(telegramID)

	// Удаляем старое сообщение
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
	})

	// Получаем пользователя для персонализированного меню
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "❌ Ошибка. Используйте /start",
		})
		return
	}

	menuText := "📋 Главное меню\n\n" +
		"Доступные команды:\n" +
		"/subjects - Посмотреть все предметы\n" +
		"/mybookings - Мои записи\n" +
		"/help - Справка\n"

	if user.IsTeacher {
		menuText += "\nКоманды учителя:\n" +
			"/mysubjects - Мои предметы\n" +
			"/myschedule - Моё расписание\n" +
			"/createsubject - Создать предмет"
	} else {
		menuText += "\n/becometeacher - Стать учителем"
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   menuText,
	})

	AnswerCallback(ctx, b, callback.ID, "Возврат в главное меню")
}

// HandleBookAnother показывает доступные предметы для записи
func HandleBookAnother(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	msg := GetMessageFromCallback(callback)
	if msg == nil {
		AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Удаляем старое сообщение
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
	})

	// Показываем список предметов (вызываем handleSubjects)
	update := &models.Update{
		Message: &models.Message{
			Chat: models.Chat{ID: msg.Chat.ID},
			From: &callback.From,
		},
	}

	h.HandleSubjects(ctx, b, update)
	AnswerCallback(ctx, b, callback.ID, "Показываем предметы")
}

// HandleBackToSubjects возвращает учителя к списку его предметов
func HandleBackToSubjects(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	msg := GetMessageFromCallback(callback)
	if msg == nil {
		AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Редактируем сообщение вместо удаления
	update := &models.Update{
		CallbackQuery: callback,
		Message: &models.Message{
			Chat: models.Chat{ID: msg.Chat.ID},
			From: &callback.From,
		},
	}

	h.HandleMySubjects(ctx, b, update, msg.ID)
	AnswerCallback(ctx, b, callback.ID, "")
}

// HandleBackToMySchedule возвращает к главному меню /myschedule
func HandleBackToMySchedule(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	msg := GetMessageFromCallback(callback)
	if msg == nil {
		AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	// Удаляем сообщение и вызываем HandleMySchedule
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
	})

	update := &models.Update{
		Message: &models.Message{
			Chat: models.Chat{ID: msg.Chat.ID},
			From: &callback.From,
		},
	}

	h.HandleMySchedule(ctx, b, update)
	AnswerCallback(ctx, b, callback.ID, "")
}
