package model

import "time"

type Project struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;size:128;not null" json:"name"`
	Description string    `gorm:"size:512;not null;default:''" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Namespaces []ProjectNamespace `gorm:"foreignKey:ProjectID" json:"namespaces,omitempty"`
	Users      []UserProject      `gorm:"foreignKey:ProjectID" json:"users,omitempty"`
}

func (Project) TableName() string {
	return "projects"
}

type ProjectNamespace struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProjectID uint      `gorm:"not null;uniqueIndex:uk_project_ns" json:"project_id"`
	Namespace string    `gorm:"size:253;not null;uniqueIndex:uk_project_ns" json:"namespace"`
	CreatedAt time.Time `json:"created_at"`
}

func (ProjectNamespace) TableName() string {
	return "project_namespaces"
}

type UserProject struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"not null;uniqueIndex:uk_user_project" json:"user_id"`
	ProjectID  uint      `gorm:"not null;uniqueIndex:uk_user_project" json:"project_id"`
	Permission string    `gorm:"size:16;not null;default:'read'" json:"permission"` // read, readwrite
	CreatedAt  time.Time `json:"created_at"`

	User    *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Project *Project `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
}

func (UserProject) TableName() string {
	return "user_projects"
}
