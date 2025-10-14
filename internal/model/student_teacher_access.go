package model

import "time"

// StudentTeacherAccess represents access relationship between student and private teacher
type StudentTeacherAccess struct {
	ID         int64     `json:"id"`
	StudentID  int64     `json:"student_id"`
	TeacherID  int64     `json:"teacher_id"`
	AccessType string    `json:"access_type"` // 'invited', 'approved', 'subscribed'
	GrantedAt  time.Time `json:"granted_at"`
}

// Access type constants
const (
	AccessTypeInvited    = "invited"    // Access granted via invite code
	AccessTypeApproved   = "approved"   // Access granted via approved request
	AccessTypeSubscribed = "subscribed" // Access granted via subscription (future)
)
