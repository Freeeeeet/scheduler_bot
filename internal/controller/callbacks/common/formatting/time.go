package formatting

import (
	"fmt"
	"time"
)

// FormatDateTime форматирует дату и время
func FormatDateTime(t time.Time) string {
	return t.Format("02.01.2006 15:04")
}

// FormatDate форматирует только дату
func FormatDate(t time.Time) string {
	return t.Format("02.01.2006")
}

// FormatDateWithWeekday форматирует дату с днём недели
func FormatDateWithWeekday(t time.Time) string {
	return t.Format("02.01.2006 (Monday)")
}

// FormatTime форматирует только время
func FormatTime(t time.Time) string {
	return t.Format("15:04")
}

// FormatTimeRange форматирует диапазон времени
func FormatTimeRange(start, end time.Time) string {
	return fmt.Sprintf("%s-%s", start.Format("15:04"), end.Format("15:04"))
}

// FormatDuration форматирует длительность в минутах
func FormatDuration(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%d мин", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return fmt.Sprintf("%d ч", hours)
	}
	return fmt.Sprintf("%d ч %d мин", hours, mins)
}

// GetWeekdayName возвращает название дня недели на русском
func GetWeekdayName(weekday int) string {
	names := []string{
		"Воскресенье",
		"Понедельник",
		"Вторник",
		"Среда",
		"Четверг",
		"Пятница",
		"Суббота",
	}
	if weekday >= 0 && weekday < len(names) {
		return names[weekday]
	}
	return "Неизвестно"
}

// GetWeekdayShortName возвращает краткое название дня недели на русском
func GetWeekdayShortName(weekday int) string {
	names := []string{
		"Вс",
		"Пн",
		"Вт",
		"Ср",
		"Чт",
		"Пт",
		"Сб",
	}
	if weekday >= 0 && weekday < len(names) {
		return names[weekday]
	}
	return "?"
}

// GetWeekdayShort возвращает короткое название дня недели
func GetWeekdayShort(weekday int) string {
	names := []string{"Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"}
	if weekday >= 0 && weekday < len(names) {
		return names[weekday]
	}
	return "?"
}

// GetMonthName возвращает название месяца на русском
func GetMonthName(month time.Month) string {
	names := map[time.Month]string{
		time.January:   "Январь",
		time.February:  "Февраль",
		time.March:     "Март",
		time.April:     "Апрель",
		time.May:       "Май",
		time.June:      "Июнь",
		time.July:      "Июль",
		time.August:    "Август",
		time.September: "Сентябрь",
		time.October:   "Октябрь",
		time.November:  "Ноябрь",
		time.December:  "Декабрь",
	}
	return names[month]
}
