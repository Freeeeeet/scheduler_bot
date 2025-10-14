package model

import "time"

// AccessRequest represents a student's request for access to a private teacher
type AccessRequest struct {
	ID              int64      `json:"id"`
	StudentID       int64      `json:"student_id"`
	TeacherID       int64      `json:"teacher_id"`
	Status          string     `json:"status"` // 'pending', 'approved', 'rejected'
	Message         string     `json:"message"`
	TeacherResponse string     `json:"teacher_response"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"`
}

// Request status constants
const (
	RequestStatusPending  = "pending"
	RequestStatusApproved = "approved"
	RequestStatusRejected = "rejected"
)

// IsPending checks if request is pending
func (r *AccessRequest) IsPending() bool {
	return r.Status == RequestStatusPending
}

// IsApproved checks if request is approved
func (r *AccessRequest) IsApproved() bool {
	return r.Status == RequestStatusApproved
}

// IsRejected checks if request is rejected
func (r *AccessRequest) IsRejected() bool {
	return r.Status == RequestStatusRejected
}
