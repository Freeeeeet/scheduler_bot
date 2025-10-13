package schedule

import (
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
// Week View Handlers
// ========================

func HandleViewScheduleWeeks(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleViewScheduleWeeks called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: view_schedule_weeks:0 (weekOffset)
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 2 {
		h.Logger.Error("Invalid callback format", zap.String("data", callback.Data))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	weekOffset, err := strconv.Atoi(parts[1])
	if err != nil {
		h.Logger.Error("Failed to parse week offset", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверное смещение")
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

	now := time.Now()
	var startDate, endDate time.Time

	if weekOffset == 0 {
		// Текущая неделя - от сегодня до воскресенья
		startDate = now
		// Найти ближайшее воскресенье
		daysUntilSunday := (7 - int(now.Weekday())) % 7
		if daysUntilSunday == 0 && now.Weekday() != time.Sunday {
			daysUntilSunday = 7
		}
		endDate = now.AddDate(0, 0, daysUntilSunday).Add(24*time.Hour - time.Second)
	} else {
		// Находим понедельник нужной недели
		// Сначала найдем понедельник текущей недели
		daysSinceMonday := int(now.Weekday()) - 1
		if now.Weekday() == time.Sunday {
			daysSinceMonday = 6
		}

		thisMonday := now.AddDate(0, 0, -daysSinceMonday)

		// Применяем смещение в неделях
		targetMonday := thisMonday.AddDate(0, 0, weekOffset*7)
		startDate = time.Date(targetMonday.Year(), targetMonday.Month(), targetMonday.Day(), 0, 0, 0, 0, targetMonday.Location())
		endDate = startDate.AddDate(0, 0, 7).Add(-time.Second)
	}

	var buttons [][]models.InlineKeyboardButton

	// Генерируем кнопки для каждого дня недели
	currentDate := startDate
	for currentDate.Before(endDate) || currentDate.Equal(endDate.Add(-24*time.Hour)) {
		dateStr := currentDate.Format("2006-01-02")
		weekdayShort := formatting.GetWeekdayShort(int(currentDate.Weekday()))
		displayText := fmt.Sprintf("%s, %s", weekdayShort, currentDate.Format("02.01"))

		// Добавляем метку "Сегодня" если это текущий день
		if currentDate.Format("2006-01-02") == now.Format("2006-01-02") {
			displayText = "Сегодня • " + displayText
		}

		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: displayText, CallbackData: fmt.Sprintf("view_schedule_week_day:%d:%s", weekOffset, dateStr)},
		})

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	// Кнопки навигации
	var navButtons []models.InlineKeyboardButton

	// Кнопка "предыдущая неделя" только если не первая (текущая) неделя
	if weekOffset > 0 {
		navButtons = append(navButtons, models.InlineKeyboardButton{
			Text:         "⬅️ Пред. неделя",
			CallbackData: fmt.Sprintf("view_schedule_weeks:%d", weekOffset-1),
		})
	}

	// Кнопка "следующая неделя" (до 12 недель вперед)
	if weekOffset < 12 {
		navButtons = append(navButtons, models.InlineKeyboardButton{
			Text:         "След. неделя ➡️",
			CallbackData: fmt.Sprintf("view_schedule_weeks:%d", weekOffset+1),
		})
	}

	if len(navButtons) > 0 {
		buttons = append(buttons, navButtons)
	}

	// Кнопка "Назад"
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⬅️ Назад", CallbackData: "back_to_myschedule"},
	})

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	// Форматируем заголовок недели согласно ТЗ
	var weekLabel string
	if weekOffset == 0 {
		weekLabel = "Текущая неделя"
	} else {
		// Для последующих недель показываем диапазон дат в формате DD.MM-DD.MM
		weekLabel = fmt.Sprintf("Неделя %s-%s",
			startDate.Format("02.01"),
			endDate.Format("02.01"))
	}

	text := fmt.Sprintf("📅 <b>Просмотр расписания</b>\n\n"+
		"📍 %s\n\n"+
		"Выберите день:",
		weekLabel)

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}

// HandleViewScheduleWeekDay показывает расписание на конкретный день недели
func HandleViewScheduleWeekDay(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	h.Logger.Info("HandleViewScheduleWeekDay called",
		zap.String("callback_data", callback.Data),
		zap.Int64("user_id", callback.From.ID))

	// Формат: view_schedule_week_day:0:2024-01-15 (weekOffset:date)
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 3 {
		h.Logger.Error("Invalid callback format", zap.String("data", callback.Data))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		return
	}

	weekOffset, err := strconv.Atoi(parts[1])
	if err != nil {
		h.Logger.Error("Failed to parse week offset", zap.Error(err))
		common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверное смещение")
		return
	}

	dateStr := parts[2]
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

	// Получаем слоты на этот день
	startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	endOfDay := startOfDay.AddDate(0, 0, 1)

	allSlots, err := h.TeacherService.GetTeacherSchedule(ctx, user.ID, startOfDay, endOfDay)
	if err != nil {
		h.Logger.Error("Failed to get schedule", zap.Error(err))
		allSlots = []*model.ScheduleSlot{}
	}

	// Группируем слоты по предметам
	slotsBySubject := make(map[int64][]*model.ScheduleSlot)
	subjectNames := make(map[int64]string)

	for _, slot := range allSlots {
		slotsBySubject[slot.SubjectID] = append(slotsBySubject[slot.SubjectID], slot)
		if _, exists := subjectNames[slot.SubjectID]; !exists {
			subject, err := h.TeacherService.GetSubjectByID(ctx, slot.SubjectID)
			if err == nil && subject != nil {
				subjectNames[slot.SubjectID] = subject.Name
			}
		}
	}

	text := fmt.Sprintf("📅 <b>Расписание на %s</b>\n\n", targetDate.Format("02.01.2006"))
	text += fmt.Sprintf("📆 День: %s\n\n", formatting.GetWeekdayName(int(targetDate.Weekday())))

	if len(allSlots) == 0 {
		text += "📭 <b>На этот день нет слотов</b>"
	} else {
		totalSlots := len(allSlots)
		bookedCount := 0
		freeCount := 0

		for _, slot := range allSlots {
			if slot.Status == model.SlotStatusBooked {
				bookedCount++
			} else if slot.Status == model.SlotStatusFree {
				freeCount++
			}
		}

		text += fmt.Sprintf("📊 <b>Всего слотов:</b> %d\n", totalSlots)
		text += fmt.Sprintf("🟢 Свободно: %d\n", freeCount)
		text += fmt.Sprintf("🔴 Забронировано: %d\n\n", bookedCount)
		text += "Нажмите на слот для просмотра деталей:"
	}

	// Создаем кнопки для слотов
	var buttons [][]models.InlineKeyboardButton

	if len(allSlots) > 0 {
		// Сортируем слоты по времени и группируем по предметам
		for subjectID, slots := range slotsBySubject {
			subjectName := subjectNames[subjectID]

			// Сортируем слоты по времени
			sort.Slice(slots, func(i, j int) bool {
				return slots[i].StartTime.Before(slots[j].StartTime)
			})

			// Добавляем заголовок предмета (как неактивную кнопку с noop)
			buttons = append(buttons, []models.InlineKeyboardButton{
				{Text: fmt.Sprintf("📚 %s", subjectName), CallbackData: "noop"},
			})

			// Добавляем кнопки для каждого слота
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

				buttonText := fmt.Sprintf("%s %s-%s (%s)",
					statusEmoji,
					slot.StartTime.Format("15:04"),
					slot.EndTime.Format("15:04"),
					statusText)

				buttons = append(buttons, []models.InlineKeyboardButton{
					{Text: buttonText, CallbackData: fmt.Sprintf("view_slot_details:%d:%d", slot.ID, weekOffset)},
				})
			}
		}
	}

	// Кнопка "Назад"
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "⬅️ Назад к неделе", CallbackData: fmt.Sprintf("view_schedule_weeks:%d", weekOffset)},
	})

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	common.AnswerCallback(ctx, b, callback.ID, "")
}
