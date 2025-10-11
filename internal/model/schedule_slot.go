package model

import "time"

type SlotStatus string

const (
	SlotStatusFree     SlotStatus = "free"
	SlotStatusBooked   SlotStatus = "booked"
	SlotStatusCanceled SlotStatus = "canceled"
)

type ScheduleSlot struct {
	ID        int64      `json:"id"`
	TeacherID int64      `json:"teacher_id"`
	SubjectID int64      `json:"subject_id"`
	StartTime time.Time  `json:"start_time"`
	EndTime   time.Time  `json:"end_time"`
	Status    SlotStatus `json:"status"`
	StudentID *int64     `json:"student_id"` // указатель - может быть nil
	CreatedAt time.Time  `json:"created_at"`
}
