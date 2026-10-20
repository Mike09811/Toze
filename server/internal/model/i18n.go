package model

import "time"

// I18nMessage 多语言词条。
type I18nMessage struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	Locale    string `gorm:"size:16;index:idx_i18n_key,unique,priority:1"`
	Module    string `gorm:"size:64;index:idx_i18n_key,unique,priority:2"`
	Key       string `gorm:"size:128;index:idx_i18n_key,unique,priority:3"`
	Value     string `gorm:"size:512"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName 表名。
func (I18nMessage) TableName() string { return "sys_i18n_messages" }
