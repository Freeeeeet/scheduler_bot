package model

import "time"

// TeacherInviteCode represents an invite code created by a teacher for student access
type TeacherInviteCode struct {
	ID          int64      `json:"id"`
	TeacherID   int64      `json:"teacher_id"`
	Code        string     `json:"code"`
	MaxUses     *int       `json:"max_uses"` // nil = unlimited uses
	CurrentUses int        `json:"current_uses"`
	ExpiresAt   *time.Time `json:"expires_at"` // nil = never expires
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
}

// IsValid checks if the invite code is valid for use
func (t *TeacherInviteCode) IsValid() bool {
	if !t.IsActive {
		return false
	}

	// Check if expired
	if t.ExpiresAt != nil && time.Now().After(*t.ExpiresAt) {
		return false
	}

	// Check if max uses reached
	if t.MaxUses != nil && t.CurrentUses >= *t.MaxUses {
		return false
	}

	return true
}

// CanUse checks if the code can be used (same as IsValid, kept for clarity)
func (t *TeacherInviteCode) CanUse() bool {
	return t.IsValid()
}
