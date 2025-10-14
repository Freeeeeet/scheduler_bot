package model

import "time"

// RecurringSchedule представляет шаблон регулярного расписания
type RecurringSchedule struct {
	ID              int64     `json:"id"`
	GroupID         int64     `json:"group_id"` // идентификатор группы связанных расписаний
	TeacherID       int64     `json:"teacher_id"`
	SubjectID       int64     `json:"subject_id"`
	Weekday         int       `json:"weekday"`          // 0 = Sunday, 6 = Saturday
	StartHour       int       `json:"start_hour"`       // 0-23
	StartMinute     int       `json:"start_minute"`     // 0-59
	DurationMinutes int       `json:"duration_minutes"` // длительность в минутах
	IsActive        bool      `json:"is_active"`        // активен ли шаблон
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
