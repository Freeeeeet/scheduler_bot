package formatting

import (
	"fmt"
	"sort"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
)

// FormatSubjectInfo форматирует информацию о предмете
func FormatSubjectInfo(subject *model.Subject) string {
	statusEmoji := "✅"
	statusText := "Активен"
	if !subject.IsActive {
		statusEmoji = "⏸"
		statusText = "Неактивен"
	}

	approvalText := ""
	if subject.RequiresBookingApproval {
		approvalText = "\n⏳ Требуется одобрение для записи"
	}

	return fmt.Sprintf(
		"%s <b>%s</b>\n\n"+
			"💰 Цена: %s\n"+
			"⏱ Длительность: %s\n"+
			"📝 Описание: %s\n"+
			"📊 Статус: %s%s",
		statusEmoji,
		subject.Name,
		FormatPrice(subject.Price),
		FormatDuration(subject.Duration),
		subject.Description,
		statusText,
		approvalText,
	)
}

// FormatSubjectShort форматирует краткую информацию о предмете
func FormatSubjectShort(subject *model.Subject, index int) string {
	approvalEmoji := ""
	if subject.RequiresBookingApproval {
		approvalEmoji = " ⏳"
	}

	return fmt.Sprintf(
		"%d. %s%s\n"+
			"   💰 %s | ⏱ %s\n"+
			"   📝 %s",
		index,
		subject.Name,
		approvalEmoji,
		FormatPriceShort(subject.Price),
		FormatDuration(subject.Duration),
		subject.Description,
	)
}

// FormatSlotInfo форматирует информацию о слоте
func FormatSlotInfo(slot *model.ScheduleSlot, subject *model.Subject) string {
	statusDisplay := GetSlotStatusDisplay(slot.Status)

	text := fmt.Sprintf(
		"%s <b>Слот #%d</b>\n\n"+
			"📚 Предмет: %s\n"+
			"📅 Дата: %s\n"+
			"🕐 Время: %s\n"+
			"⏱ Длительность: %s\n"+
			"📊 Статус: %s",
		statusDisplay.Emoji,
		slot.ID,
		subject.Name,
		FormatDateWithWeekday(slot.StartTime),
		FormatTimeRange(slot.StartTime, slot.EndTime),
		FormatDuration(subject.Duration),
		statusDisplay.Text,
	)

	return text
}

// FormatBookingInfo форматирует информацию о бронировании
func FormatBookingInfo(booking *model.Booking) string {
	statusDisplay := GetBookingStatusDisplay(booking.Status)

	return fmt.Sprintf(
		"%s <b>Запись #%d</b>\n\n"+
			"📊 Статус: %s\n"+
			"📅 Создана: %s",
		statusDisplay.Emoji,
		booking.ID,
		statusDisplay.Text,
		FormatDateTime(booking.CreatedAt),
	)
}

// FormatWeekdayRange форматирует диапазон дней недели
// Например: [1,2,3] -> "Пн-Ср", [1,3,5] -> "Пн, Ср, Пт"
func FormatWeekdayRange(weekdays []int) string {
	if len(weekdays) == 0 {
		return ""
	}

	sorted := make([]int, len(weekdays))
	copy(sorted, weekdays)
	sort.Ints(sorted)

	// Проверяем на последовательность
	isSequence := true
	for i := 1; i < len(sorted); i++ {
		if sorted[i] != sorted[i-1]+1 {
			isSequence = false
			break
		}
	}

	if isSequence && len(sorted) > 2 {
		// Диапазон: Пн-Пт
		return fmt.Sprintf("%s-%s",
			GetWeekdayShort(sorted[0]),
			GetWeekdayShort(sorted[len(sorted)-1]))
	}

	// Перечисление: Пн, Ср, Пт
	result := ""
	for i, wd := range sorted {
		if i > 0 {
			result += ", "
		}
		result += GetWeekdayShort(wd)
	}
	return result
}

// FormatRecurringSchedule форматирует информацию о recurring schedule
func FormatRecurringSchedule(schedule *model.RecurringSchedule) string {
	weekdayName := GetWeekdayName(schedule.Weekday)
	timeStr := fmt.Sprintf("%02d:%02d", schedule.StartHour, schedule.StartMinute)

	statusEmoji := "✅"
	statusText := "Активно"
	if !schedule.IsActive {
		statusEmoji = "⏸"
		statusText = "Неактивно"
	}

	return fmt.Sprintf(
		"%s <b>Постоянное расписание #%d</b>\n\n"+
			"📅 День недели: %s\n"+
			"🕐 Время: %s\n"+
			"⏱ Длительность: %s\n"+
			"📊 Статус: %s",
		statusEmoji,
		schedule.ID,
		weekdayName,
		timeStr,
		FormatDuration(schedule.DurationMinutes),
		statusText,
	)
}
