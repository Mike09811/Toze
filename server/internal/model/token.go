package model

import "time"

// RevokedToken 已注销的令牌（jti 黑名单）。
type RevokedToken struct {
	JTI       string    `gorm:"size:64;primaryKey"`
	ExpiresAt time.Time `gorm:"index"`
	CreatedAt time.Time
}

// TableName 表名。
func (RevokedToken) TableName() string { return "sys_revoked_tokens" }
