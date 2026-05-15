package model

import (
	"time"
)

// Role 角色模型
type Role struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:64;unique;not null" json:"name"`
	Description string    `gorm:"size:256" json:"description"`
	Type        string    `gorm:"size:32;not null;default:'custom'" json:"type"` // system or custom
	ParentID    uint      `gorm:"default:null" json:"parent_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Parent     *Role           `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Permissions []RolePermission `gorm:"foreignKey:RoleID" json:"permissions,omitempty"`
}

// RolePermission 角色权限模型
type RolePermission struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	RoleID    uint      `gorm:"not null" json:"role_id"`
	Resource  string    `gorm:"size:64;not null" json:"resource"`
	Actions   string    `gorm:"size:128;not null" json:"actions"` // JSON array: ["view", "create", "update", "delete"]
	CreatedAt time.Time `json:"created_at"`

	Role *Role `gorm:"foreignKey:RoleID" json:"role,omitempty"`
}

// UserRole 用户角色关联模型
type UserRole struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	RoleID    uint      `gorm:"not null" json:"role_id"`
	ProjectID uint      `gorm:"default:null" json:"project_id"` // 项目级角色时指定项目
	ExpiresAt *time.Time `json:"expires_at,omitempty"` // 过期时间（临时权限）
	CreatedAt time.Time `json:"created_at"`

	User    *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Role    *Role `gorm:"foreignKey:RoleID" json:"role,omitempty"`
	Project *Project `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
}

// TableName 设置表名
func (Role) TableName() string {
	return "roles"
}

func (RolePermission) TableName() string {
	return "role_permissions"
}

func (UserRole) TableName() string {
	return "user_roles"
}