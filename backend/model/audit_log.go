package model

import "time"

type AuditLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"not null;index" json:"user_id"`
	Username     string    `gorm:"size:64;not null;index" json:"username"`
	Action       string    `gorm:"size:16;not null;index" json:"action"`
	ResourceType string    `gorm:"size:64;not null;default:''" json:"resource_type"`
	ResourceName string    `gorm:"size:253;not null;default:''" json:"resource_name"`
	Namespace    string    `gorm:"size:253;not null;default:'';index" json:"namespace"`
	RequestPath  string    `gorm:"size:1024;not null" json:"request_path"`
	StatusCode   int       `gorm:"not null;default:0" json:"status_code"`
	ClientIP     string    `gorm:"size:45;not null;default:''" json:"client_ip"`
	Detail       string    `gorm:"size:1024;not null;default:''" json:"detail"`
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

type LoginLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Username  string    `gorm:"size:64;not null;index" json:"username"`
	ClientIP  string    `gorm:"size:45;not null;default:''" json:"client_ip"`
	Result    string    `gorm:"size:16;not null" json:"result"` // success, failed, locked
	Reason    string    `gorm:"size:255;not null;default:''" json:"reason"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

func (LoginLog) TableName() string {
	return "login_logs"
}
