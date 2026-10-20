package model

import (
	"time"

	"gorm.io/gorm"
)

// Role 角色，菜单与角色多对多。
type Role struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	Name        string    `gorm:"size:64;uniqueIndex;not null"`
	Desc        string    `gorm:"size:255"`
	DefaultRole int       `gorm:"default:0"`
	Menus       []Menu    `gorm:"many2many:sys_role_menus;"`
	CreatedAt   time.Time `gorm:"index"`
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// TableName 表名。
func (Role) TableName() string { return "sys_roles" }
