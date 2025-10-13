package handlers

import (
	"fmt"

	cmdfmt "github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common/formatting"
	"github.com/Freeeeeet/scheduler_bot/internal/model"
)

// BookingStatusDisplay содержит emoji и текст для отображения статуса
type BookingStatusDisplay struct {
	Emoji string
	Text  string
}

// GetBookingStatusDisplay возвращает emoji и текст для статуса бронирования
// Deprecated: используйте cmdfmt.GetBookingStatusDisplay
func GetBookingStatusDisplay(status model.BookingStatus) BookingStatusDisplay {
	display := cmdfmt.GetBookingStatusDisplay(status)
	return BookingStatusDisplay{
		Emoji: display.Emoji,
		Text:  display.Text,
	}
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
// Deprecated: используйте cmdfmt.FormatPrice
func FormatPrice(priceInCents int) string {
	return cmdfmt.FormatPrice(priceInCents)
}
