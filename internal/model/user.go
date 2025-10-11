package model

import "time"

type User struct {
	ID                  int64     `json:"id"`
	TelegramID          int64     `json:"telegram_id"`
	Username            string    `json:"username"`
	FirstName           string    `json:"first_name"`
	LastName            string    `json:"last_name"`
	LanguageCode        string    `json:"language_code"`
	IsTeacher           bool      `json:"is_teacher"`
	AutoApproveBookings bool      `json:"auto_approve_bookings"` // Автоматически одобрять записи
	CreatedAt           time.Time `json:"created_at"`
}
