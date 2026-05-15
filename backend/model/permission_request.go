package model

import "time"

type PermissionRequest struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"not null;index" json:"user_id"`
	Username   string     `gorm:"size:64;not null" json:"username"`
	ProjectID  uint       `gorm:"not null;index" json:"project_id"`
	Reason     string     `gorm:"size:512;not null;default:''" json:"reason"`
	Status     string     `gorm:"size:16;not null;default:'pending';index" json:"status"` // pending, approved, rejected, expired
	ReviewerID uint       `gorm:"not null;default:0" json:"reviewer_id"`
	Reviewer   string     `gorm:"size:64;not null;default:''" json:"reviewer"`
	ReviewNote string     `gorm:"size:512;not null;default:''" json:"review_note"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	User    *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Project *Project `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
}

func (PermissionRequest) TableName() string {
	return "permission_requests"
}
