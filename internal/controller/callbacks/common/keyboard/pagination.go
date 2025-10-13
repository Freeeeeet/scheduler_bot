package keyboard

import (
	"fmt"

	"github.com/go-telegram/bot/models"
)

// PaginationButtons создаёт ряд кнопок пагинации
// prefix - префикс для callback (например "subjects_page:")
// currentPage - текущая страница (0-based)
// totalPages - всего страниц
func PaginationButtons(prefix string, currentPage, totalPages int) []models.InlineKeyboardButton {
	if totalPages <= 1 {
		return nil
	}

	var buttons []models.InlineKeyboardButton

	// Кнопка "Предыдущая"
	if currentPage > 0 {
		buttons = append(buttons, Button("⬅️", fmt.Sprintf("%s%d", prefix, currentPage-1)))
	}

	// Индикатор страницы
	buttons = append(buttons, Button(
		fmt.Sprintf("📄 %d/%d", currentPage+1, totalPages),
		"noop",
	))

	// Кнопка "Следующая"
	if currentPage < totalPages-1 {
		buttons = append(buttons, Button("➡️", fmt.Sprintf("%s%d", prefix, currentPage+1)))
	}

	return buttons
}

// AddPagination добавляет пагинацию к builder
func (b *Builder) AddPagination(prefix string, currentPage, totalPages int) *Builder {
	buttons := PaginationButtons(prefix, currentPage, totalPages)
	if len(buttons) > 0 {
		b.Row(buttons...)
	}
	return b
}

// CalendarPagination создаёт пагинацию для календаря (месяц/год)
func CalendarPagination(prefix string, currentMonth, currentYear int) []models.InlineKeyboardButton {
	return []models.InlineKeyboardButton{
		Button("◀️", fmt.Sprintf("%s%d:%d", prefix, currentMonth-1, currentYear)),
		Button(fmt.Sprintf("📅 %02d/%d", currentMonth, currentYear), "noop"),
		Button("▶️", fmt.Sprintf("%s%d:%d", prefix, currentMonth+1, currentYear)),
	}
}

// WeekPagination создаёт пагинацию по неделям
func WeekPagination(prefix string, weekOffset int) []models.InlineKeyboardButton {
	return []models.InlineKeyboardButton{
		Button("◀️ Предыдущая неделя", fmt.Sprintf("%s%d", prefix, weekOffset-1)),
		Button("▶️ Следующая неделя", fmt.Sprintf("%s%d", prefix, weekOffset+1)),
	}
}
