package schedule

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common/formatting"
	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// ========================
// Calendar View Handlers
// ========================

func HandleViewScheduleCalendar(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleViewScheduleCalendar called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	subjectID, err := common.ParseIDFromCallback(callback.Data)
	if err != nil {
		h.Logger.Error("Failed to parse subject ID", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	showScheduleCalendar(ctx, b, callback, h, msg, subjectID, 0)
}

// HandleViewScheduleCalendarPage обрабатывает пагинацию календаря
func HandleViewScheduleCalendarPage(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleViewScheduleCalendarPage called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: schedule_calendar_page:subjectID:offset
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 3 {
		h.Logger.Error("Invalid callback format", zap.String("data", callback.Data))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	subjectID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		h.Logger.Error("Failed to parse subject ID", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный ID")
		return
	}

	offset, err := strconv.Atoi(parts[2])
	if err != nil {
		h.Logger.Error("Failed to parse offset", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверное смещение")
		return
	}

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	showScheduleCalendar(ctx, b, callback, h, msg, subjectID, offset)
}

// showScheduleCalendar показывает календарь для выбора дня (7 дней с пагинацией)
func showScheduleCalendar(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler, msg *models.Message, subjectID int64, offset int) {
	// Получаем предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		h.Logger.Error("Subject not found", zap.Int64("subject_id", subjectID), zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	now := time.Now()

	var buttons [][]models.InlineKeyboardButton

	// Генерируем кнопки для 7 дней начиная с offset
	for i := 0; i < 7; i++ {
		dayOffset := offset + i
		date := now.AddDate(0, 0, dayOffset)
		dateStr := date.Format("2006-01-02")
		weekdayShort := formatting.GetWeekdayShort(int(date.Weekday()))
		weekdayFull := formatting.GetWeekdayName(int(date.Weekday()))
		displayText := fmt.Sprintf("%s, %s", weekdayShort, date.Format("02.01"))

		// Добавляем специальные метки для сегодня и завтра (только на первой странице)
		if offset == 0 && i == 0 {
			displayText = "Сегодня • " + displayText
		} else if offset == 0 && i == 1 {
			displayText = "Завтра • " + displayText
		}

		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: displayText, CallbackData: fmt.Sprintf("view_schedule_day:%d:%s:%s", subjectID, dateStr, weekdayFull)},
		})
	}

	// Кнопки навигации (вперед/назад)
	var navButtons []models.InlineKeyboardButton

	// Кнопка "назад" только если не на первой странице
	if offset > 0 {
		prevOffset := offset - 7
		if prevOffset < 0 {
			prevOffset = 0
		}
		navButtons = append(navButtons, models.InlineKeyboardButton{
			Text:         "⬅️ Пред. неделя",
			CallbackData: fmt.Sprintf("schedule_calendar_page:%d:%d", subjectID, prevOffset),
		})
	}

	// Кнопка "вперед" (показываем до 12 недель вперед)
	if offset < 84 {
		nextOffset := offset + 7
		navButtons = append(navButtons, models.InlineKeyboardButton{
			Text:         "След. неделя ➡️",
			CallbackData: fmt.Sprintf("schedule_calendar_page:%d:%d", subjectID, nextOffset),
		})
	}

	if len(navButtons) > 0 {
		buttons = append(buttons, navButtons)
	}

	// Кнопка "Назад к предмету"
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⬅️ Назад к предмету", CallbackData: fmt.Sprintf("view_subject:%d", subjectID)},
	})

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	weekNum := (offset / 7) + 1
	text := fmt.Sprintf("📅 <b>Расписание: %s</b>\n\n📍 Неделя %d\n\nВыберите день для просмотра:", subject.Name, weekNum)

	// Вычисляем даты недели для изображения (нормализуем к понедельнику)
	startDate := now.AddDate(0, 0, offset)
	normalizedStart := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	daysSinceMonday := int(normalizedStart.Weekday()) - 1
	if normalizedStart.Weekday() == time.Sunday {
		daysSinceMonday = 6
	}
	weekStart := normalizedStart.AddDate(0, 0, -daysSinceMonday)
	weekEnd := weekStart.AddDate(0, 0, 7) // воскресенье + 1 день для включения воскресенья

	// Получаем слоты для недели
	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err == nil && user != nil {
		weekSlots, err := h.TeacherService.GetTeacherSchedule(ctx, user.ID, weekStart, weekEnd)
		if err == nil {
			// Собираем ID студентов для получения имен
			studentIDsMap := make(map[int64]bool)
			for _, slot := range weekSlots {
				if slot.StudentID != nil {
					studentIDsMap[*slot.StudentID] = true
				}
			}
			studentIDs := make([]int64, 0, len(studentIDsMap))
			for id := range studentIDsMap {
				studentIDs = append(studentIDs, id)
			}
			studentNames := make(map[int64]string)
			if len(studentIDs) > 0 {
				students, _ := h.UserService.GetByIDs(ctx, studentIDs)
				for _, student := range students {
					name := student.FirstName
					if student.LastName != "" {
						name += " " + student.LastName
					}
					studentNames[student.ID] = name
				}
			}
			// Генерируем изображение недели
			imageData, err := common.GenerateWeekImage(weekStart, weekEnd, weekSlots, subjectID, studentNames)
			if err == nil {
				// Отправляем изображение с подписью
				b.SendPhoto(ctx, &bot.SendPhotoParams{
					ChatID:      msg.Chat.ID,
					Photo:       &models.InputFileUpload{Filename: "week.png", Data: bytes.NewReader(imageData)},
					Caption:     text,
					ParseMode:   models.ParseModeHTML,
					ReplyMarkup: keyboard,
				})
				// Удаляем старое сообщение
				b.DeleteMessage(ctx, &bot.DeleteMessageParams{
					ChatID:    msg.Chat.ID,
					MessageID: msg.ID,
				})
				common.AnswerCallback(ctx, b, callback.ID, "")
				return
			}
		}
	}

	// Если не удалось сгенерировать изображение, отправляем текст
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleViewScheduleDay показывает расписание на конкретный день
func HandleViewScheduleDay(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleViewScheduleDay called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: view_schedule_day:subjectID:2024-01-15:Понедельник
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 4 {
		h.Logger.Error("Invalid callback format", zap.String("data", callback.Data))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	subjectID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		h.Logger.Error("Failed to parse subject ID", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный ID")
		return
	}

	dateStr := parts[2]
	weekday := parts[3]

	targetDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		h.Logger.Error("Failed to parse date", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверная дата")
		return
	}

	msg := common.GetMessageFromCallback(callback)
	if msg == nil {
		h.Logger.Error("Failed to get message from callback")
		common.AnswerCallback(ctx, b, callback.ID, "❌ Ошибка")
		return
	}

	telegramID := callback.From.ID
	user, err := h.UserService.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Пользователь не найден")
		return
	}

	// Получаем предмет
	subject, err := h.TeacherService.GetSubjectByID(ctx, subjectID)
	if err != nil || subject == nil {
		h.Logger.Error("Subject not found", zap.Int64("subject_id", subjectID), zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
		return
	}

	// Получаем слоты на этот день
	startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	endOfDay := startOfDay.AddDate(0, 0, 1)

	allSlots, err := h.TeacherService.GetTeacherSchedule(ctx, user.ID, startOfDay, endOfDay)
	if err != nil {
		h.Logger.Error("Failed to get schedule", zap.Error(err))
		allSlots = []*model.ScheduleSlot{}
	}

	// Фильтруем только слоты для этого предмета
	var slots []*model.ScheduleSlot
	for _, slot := range allSlots {
		if slot.SubjectID == subjectID {
			slots = append(slots, slot)
		}
	}

	// Вычисляем начало недели для изображения
	daysSinceMonday := int(targetDate.Weekday()) - 1
	if targetDate.Weekday() == time.Sunday {
		daysSinceMonday = 6
	}
	weekStart := targetDate.AddDate(0, 0, -daysSinceMonday)
	weekEnd := weekStart.AddDate(0, 0, 7)

	// Получаем все слоты недели для изображения
	weekSlots, err := h.TeacherService.GetTeacherSchedule(ctx, user.ID, weekStart, weekEnd)
	if err != nil {
		h.Logger.Error("Failed to get week schedule", zap.Error(err))
		weekSlots = []*model.ScheduleSlot{}
	}

	text := fmt.Sprintf("📅 <b>Расписание на %s</b>\n\n", targetDate.Format("02.01.2006"))
	text += fmt.Sprintf("📚 Предмет: <b>%s</b>\n", subject.Name)
	text += fmt.Sprintf("📆 День: %s\n\n", weekday)

	if len(slots) == 0 {
		text += "📭 <b>На этот день нет слотов</b>\n\n"
		text += "Вы можете создать слоты через \"📊 Управление расписанием\""
	} else {
		text += fmt.Sprintf("📊 <b>Всего слотов:</b> %d\n\n", len(slots))

		// Группируем по статусу
		var freeSlots, bookedSlots, canceledSlots int
		for _, slot := range slots {
			switch slot.Status {
			case model.SlotStatusFree:
				freeSlots++
			case model.SlotStatusBooked:
				bookedSlots++
			case model.SlotStatusCanceled:
				canceledSlots++
			}
		}

		text += "<b>Статистика:</b>\n"
		text += fmt.Sprintf("🟢 Свободно: %d\n", freeSlots)
		text += fmt.Sprintf("🔴 Забронировано: %d\n", bookedSlots)
		if canceledSlots > 0 {
			text += fmt.Sprintf("⚫️ Отменено: %d\n", canceledSlots)
		}
		text += "\n<b>Выберите слот для бронирования:</b>\n"
	}

	// Создаем кнопки для слотов
	var buttons [][]models.InlineKeyboardButton

	// Сортируем слоты по времени
	sort.Slice(slots, func(i, j int) bool {
		return slots[i].StartTime.Before(slots[j].StartTime)
	})

	for _, slot := range slots {
		statusEmoji := "🟢"
		statusText := "Свободен"
		switch slot.Status {
		case model.SlotStatusBooked:
			statusEmoji = "🔴"
			statusText = "Забронирован"
		case model.SlotStatusCanceled:
			statusEmoji = "⚫️"
			statusText = "Отменён"
		}

		buttonText := fmt.Sprintf("%s %s-%s", statusEmoji, slot.StartTime.Format("15:04"), slot.EndTime.Format("15:04"))

		// Для свободных слотов показываем разные кнопки для преподавателя и студента
		if slot.Status == model.SlotStatusFree {
			if user.IsTeacher {
				// Для преподавателя - одна кнопка, которая открывает экран выбора действия
				buttons = append(buttons, []models.InlineKeyboardButton{
					{Text: buttonText, CallbackData: fmt.Sprintf("slot_action:%d:%d:%s", slot.ID, subjectID, dateStr)},
				})
			} else {
				// Для студента - кнопка для бронирования
				buttons = append(buttons, []models.InlineKeyboardButton{
					{Text: buttonText, CallbackData: fmt.Sprintf("book_lesson:%d", slot.ID)},
				})
			}
		} else {
			// Для забронированных/отмененных - неактивная кнопка
			buttons = append(buttons, []models.InlineKeyboardButton{
				{Text: buttonText + " (" + statusText + ")", CallbackData: "noop"},
			})
		}
	}

	// Кнопка назад
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⬅️ Назад к календарю", CallbackData: fmt.Sprintf("view_schedule_calendar:%d", subjectID)},
	})

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	// Собираем ID студентов для получения имен
	studentIDsMap := make(map[int64]bool)
	for _, slot := range weekSlots {
		if slot.StudentID != nil {
			studentIDsMap[*slot.StudentID] = true
		}
	}
	studentIDs := make([]int64, 0, len(studentIDsMap))
	for id := range studentIDsMap {
		studentIDs = append(studentIDs, id)
	}
	studentNames := make(map[int64]string)
	if len(studentIDs) > 0 {
		students, _ := h.UserService.GetByIDs(ctx, studentIDs)
		for _, student := range students {
			name := student.FirstName
			if student.LastName != "" {
				name += " " + student.LastName
			}
			studentNames[student.ID] = name
		}
	}
	// Генерируем изображение недели
	imageData, err := common.GenerateWeekImage(weekStart, weekEnd, weekSlots, subjectID, studentNames)
	if err == nil {
		// Отправляем изображение с подписью и кнопками
		b.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID:      msg.Chat.ID,
			Photo:       &models.InputFileUpload{Filename: "week.png", Data: bytes.NewReader(imageData)},
			Caption:     text,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: keyboard,
		})
		// Удаляем старое сообщение
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    msg.Chat.ID,
			MessageID: msg.ID,
		})
	} else {
		// Если не удалось сгенерировать изображение, отправляем текст
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      msg.Chat.ID,
			MessageID:   msg.ID,
			Text:        text,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: keyboard,
		})
	}

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleViewScheduleWeeks показывает расписание по неделям с пагинацией
