package model

import "time"

type Subject struct {
	ID                      int64     `json:"id"`
	TeacherID               int64     `json:"teacher_id"`
	Name                    string    `json:"name"`
	Description             string    `json:"description"`
	Price                   int       `json:"price"`    // в копейках/центах
	Duration                int       `json:"duration"` // в минутах
	IsActive                bool      `json:"is_active"`
	RequiresBookingApproval bool      `json:"requires_booking_approval"` // требуется ли одобрение для записи
	CreatedAt               time.Time `json:"created_at"`
}
