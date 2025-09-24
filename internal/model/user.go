package model

import "time"

type User struct {
	ID         int64     `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	Username   string    `json:"username"`
	Firstname  string    `json:"firstname"`
	IsTeacher  bool      `json:"is_teacher"`
	CreatedAt  time.Time `json:"created_at"`
}
