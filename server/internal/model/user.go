package model

import (
	"time"

	"gorm.io/gorm"
)

// User 系统用户。
type User struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	UUID      string    `gorm:"size:64;uniqueIndex;not null"`
	Account   string    `gorm:"size:64;uniqueIndex;not null"`
	Password  string    `gorm:"size:128;not null" json:"-"`
	Name      string    `gorm:"size:64;not null"`
	Avatar    string    `gorm:"size:512"`
	Phone     string    `gorm:"size:32;index"`
	Mail      string    `gorm:"size:128;index"`
	Gender    int       `gorm:"default:1;comment:1男 2女"`
	Status    int       `gorm:"default:1;comment:1正常 2冻结 3关闭"`
	Birthday  string    `gorm:"size:16"`
	Region    []string  `gorm:"serializer:json"`
	Desc      string    `gorm:"type:text"`
	Roles     []Role    `gorm:"many2many:sys_user_roles;"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName 表名。
func (User) TableName() string { return "sys_users" }

// StatusText 状态文案。
func (u User) StatusText() string {
	switch u.Status {
	case 1:
		return "正常"
	case 2:
		return "冻结"
	case 3:
		return "关闭"
	}
	return "未知"
}

// RoleCodes 返回角色标识列表。
func (u User) RoleCodes() []string {
	codes := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		codes = append(codes, r.Name)
	}
	return codes
}
