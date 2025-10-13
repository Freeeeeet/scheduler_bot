package formatting

import "github.com/Freeeeeet/scheduler_bot/internal/model"

// SlotStatusDisplay представляет отображение статуса слота
type SlotStatusDisplay struct {
	Emoji string
	Text  string
}

// GetSlotStatusDisplay возвращает emoji и текст для статуса слота
func GetSlotStatusDisplay(status model.SlotStatus) SlotStatusDisplay {
	displays := map[model.SlotStatus]SlotStatusDisplay{
		model.SlotStatusFree:     {"🟢", "Свободен"},
		model.SlotStatusBooked:   {"🔴", "Занят"},
		model.SlotStatusCanceled: {"⚫️", "Отменён"},
	}

	if display, ok := displays[status]; ok {
		return display
	}

	return SlotStatusDisplay{"❓", "Неизвестно"}
}

// BookingStatusDisplay представляет отображение статуса бронирования
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

	return BookingStatusDisplay{"❓", "Неизвестно"}
}
