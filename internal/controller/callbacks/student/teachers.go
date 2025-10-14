package student

import (
	"context"
	"fmt"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common/keyboard"
	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

const itemsPerPage = 5

// HandlePublicTeachers показывает список публичных учителей с пагинацией
func HandlePublicTeachers(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	HandlePublicTeachersPage(ctx, b, callback, h, 1)
}

// HandlePublicTeachersPage показывает страницу публичных учителей
func HandlePublicTeachersPage(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler, page int) {
	// Получаем публичных учителей
	teachers, err := h.AccessService.GetPublicTeachers(ctx)
	if err != nil {
		h.Logger.Error("Failed to get public teachers", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка при загрузке учителей")
		return
	}

	totalTeachers := len(teachers)
	totalPages := (totalTeachers + itemsPerPage - 1) / itemsPerPage

	if page < 1 {
		page = 1
	}
	if page > totalPages && totalPages > 0 {
		page = totalPages
	}

	// Формируем текст
	text := "🌍 *Публичные учителя*\n\n"
	if totalTeachers == 0 {
		text += "Пока нет публичных учителей."
	} else {
		text += fmt.Sprintf("Доступно учителей: %d\n", totalTeachers)
		text += fmt.Sprintf("Страница %d из %d\n\n", page, totalPages)

		// Вычисляем диапазон для текущей страницы
		start := (page - 1) * itemsPerPage
		end := start + itemsPerPage
		if end > totalTeachers {
			end = totalTeachers
		}

		pageTeachers := teachers[start:end]
		for i, teacher := range pageTeachers {
			name := teacher.FirstName
			if teacher.LastName != "" {
				name += " " + teacher.LastName
			}

			// Получаем предметы учителя
			subjects, _ := h.TeacherService.GetTeacherSubjects(ctx, teacher.ID)
			subjectNames := ""
			if len(subjects) > 0 {
				for j, subj := range subjects {
					if subj.IsActive {
						if j > 0 {
							subjectNames += ", "
						}
						subjectNames += subj.Name
						if j >= 2 { // Показываем максимум 3 предмета
							subjectNames += "..."
							break
						}
					}
				}
			}

			text += fmt.Sprintf("%d. *%s*\n", start+i+1, name)
			if subjectNames != "" {
				text += fmt.Sprintf("   📚 %s\n", subjectNames)
			}
		}
	}

	// Формируем клавиатуру
	kb := keyboard.NewBuilder()

	// Добавляем кнопки учителей на текущей странице
	if totalTeachers > 0 {
		start := (page - 1) * itemsPerPage
		end := start + itemsPerPage
		if end > totalTeachers {
			end = totalTeachers
		}

		pageTeachers := teachers[start:end]
		for _, teacher := range pageTeachers {
			name := teacher.FirstName
			if teacher.LastName != "" {
				name += " " + teacher.LastName
			}
			kb.Row(keyboard.Button(
				fmt.Sprintf("👤 %s", name),
				fmt.Sprintf("teacher_profile:%d", teacher.ID),
			))
		}

		// Пагинация
		if totalPages > 1 {
			paginationRow := []models.InlineKeyboardButton{}
			if page > 1 {
				paginationRow = append(paginationRow, keyboard.Button("◀️ Назад", fmt.Sprintf("public_teachers_page:%d", page-1)))
			}
			paginationRow = append(paginationRow, keyboard.Button(
				fmt.Sprintf("%d/%d", page, totalPages),
				"noop",
			))
			if page < totalPages {
				paginationRow = append(paginationRow, keyboard.Button("Вперёд ▶️", fmt.Sprintf("public_teachers_page:%d", page+1)))
			}
			kb.AddRow(paginationRow)
		}
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

// HandleTeacherProfile показывает профиль учителя
func HandleTeacherProfile(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	teacherID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем учителя
	teacher, err := h.UserService.GetByID(ctx, teacherID)
	if err != nil || teacher == nil || !teacher.IsTeacher {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Учитель не найден")
		return
	}

	// Проверяем доступ
	canSee, err := h.AccessService.CanStudentSeeTeacher(ctx, user.ID, teacherID)
	if err != nil || !canSee {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Нет доступа к этому учителю")
		return
	}

	// Получаем предметы учителя
	subjects, err := h.TeacherService.GetTeacherSubjects(ctx, teacherID)
	if err != nil {
		h.Logger.Error("Failed to get teacher subjects", zap.Error(err))
		subjects = []*model.Subject{}
	}

	// Формируем текст
	teacherName := teacher.FirstName
	if teacher.LastName != "" {
		teacherName += " " + teacher.LastName
	}

	text := fmt.Sprintf("👤 *%s*\n\n", teacherName)

	if teacher.IsPublic {
		text += "🌍 Публичный учитель\n\n"
	} else {
		text += "🔒 Приватный учитель\n\n"
	}

	// Активные предметы
	activeSubjects := 0
	for _, subj := range subjects {
		if subj.IsActive {
			activeSubjects++
		}
	}

	text += fmt.Sprintf("📚 *Предметы* (%d активных):\n\n", activeSubjects)

	if activeSubjects == 0 {
		text += "Нет доступных предметов\n"
	} else {
		for _, subj := range subjects {
			if subj.IsActive {
				text += fmt.Sprintf("• *%s*\n", subj.Name)
				if subj.Description != "" {
					text += fmt.Sprintf("  %s\n", subj.Description)
				}
				text += fmt.Sprintf("  💰 %d₽/занятие • ⏱ %d мин\n\n", subj.Price, subj.Duration)
			}
		}
	}

	// Формируем клавиатуру
	kb := keyboard.NewBuilder()

	// Кнопки предметов
	for _, subj := range subjects {
		if subj.IsActive {
			kb.Row(keyboard.Button(
				fmt.Sprintf("📖 %s", subj.Name),
				fmt.Sprintf("subject:%d", subj.ID),
			))
		}
	}

	// Навигация
	kb.Row(keyboard.BackButton("public_teachers"))

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
