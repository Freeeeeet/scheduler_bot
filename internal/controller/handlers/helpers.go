package handlers

import (
	"fmt"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
)

// BookingStatusDisplay содержит emoji и текст для отображения статуса
type BookingStatusDisplay struct {
	Emoji string
	Text  string
}

// GetBookingStatusDisplay возвращает emoji и текст для статуса бронирования
func GetBookingStatusDisplay(status model.BookingStatus) BookingStatusDisplay {
	displays := map[model.BookingStatus]BookingStatusDisplay{
		model.BookingStatusPending:   {"⏳", "Ожидает одобрения"},
		model.BookingStatusConfirmed: {"✅", "Подтверждена"},
		model.BookingStatusCompleted: {"✔️", "Завершена"},
		model.BookingStatusCanceled:  {"❌", "Отменена"},
		model.BookingStatusRejected:  {"🚫", "Отклонена"},
	}

	if display, ok := displays[status]; ok {
		return display
	}

	// Fallback для неизвестных статусов
	return BookingStatusDisplay{"❓", "Неизвестно"}
}

// FormatBooking форматирует бронирование для отображения
func FormatBooking(booking *model.Booking) string {
	display := GetBookingStatusDisplay(booking.Status)

	return fmt.Sprintf(
		"%s Запись #%d\n\n"+
			"📊 Статус: %s\n"+
			"📅 Создана: %s",
		display.Emoji,
		booking.ID,
		display.Text,
		booking.CreatedAt.Format("02.01.2006 15:04"),
	)
}

// FormatPrice форматирует цену из копеек в рубли
func FormatPrice(priceInCents int) string {
	price := float64(priceInCents) / 100
	return fmt.Sprintf("%.2f ₽", price)
}
