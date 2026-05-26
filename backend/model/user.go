package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	Username          string         `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash      string         `gorm:"size:255;not null" json:"-"`
	DisplayName       string         `gorm:"size:128;not null;default:''" json:"display_name"`
	Email             string         `gorm:"size:255;not null;default:''" json:"email"`
	Role              string         `gorm:"size:20;not null;default:'developer'" json:"role"` // developer, admin
	Remark            string         `gorm:"size:512;not null;default:''" json:"remark"`
	Status            string         `gorm:"size:16;not null;default:'active'" json:"status"` // active, disabled
	PasswordChangedAt *time.Time     `json:"password_changed_at,omitempty"`
	PasswordExpiresAt *time.Time     `json:"password_expires_at,omitempty"`
	FailedLoginCount  int            `gorm:"not null;default:0" json:"-"`
	LockedUntil       *time.Time     `json:"-"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}
