package model

import "time"

type BookingStatus string

const (
	BookingStatusPending   BookingStatus = "pending"   // Ожидает одобрения учителя
	BookingStatusConfirmed BookingStatus = "confirmed" // Подтверждено
	BookingStatusCompleted BookingStatus = "completed" // Завершено
	BookingStatusCanceled  BookingStatus = "canceled"  // Отменено
	BookingStatusRejected  BookingStatus = "rejected"  // Отклонено учителем
)

type Booking struct {
	ID                      int64         `json:"id"`
	StudentID               int64         `json:"student_id"`
	TeacherID               int64         `json:"teacher_id"`
	SubjectID               int64         `json:"subject_id"`
	SlotID                  int64         `json:"slot_id"`
	Status                  BookingStatus `json:"status"`
	CancellationRequested   bool          `json:"cancellation_requested"`    // Запрос на отмену
	CancellationRequestedAt *time.Time    `json:"cancellation_requested_at"` // Когда запрошена отмену
	CreatedAt               time.Time     `json:"created_at"`
	UpdatedAt               time.Time     `json:"updated_at"`

	// Дополнительные поля для удобства (не из БД)
	Subject *Subject      `json:"subject,omitempty"`
	Slot    *ScheduleSlot `json:"slot,omitempty"`
	Student *User         `json:"student,omitempty"`
	Teacher *User         `json:"teacher,omitempty"`
}
