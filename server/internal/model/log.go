package model

import "time"

// OperationLog 操作审计日志。
type OperationLog struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	UserID    uint64 `gorm:"index"`
	Username  string `gorm:"size:64;index"`
	Action    string `gorm:"size:64;index"`
	Method    string `gorm:"size:16"`
	Path      string `gorm:"size:255;index"`
	IP        string `gorm:"size:64"`
	UserAgent string `gorm:"size:512"`
	Status    int    `gorm:"index"`
	LatencyMS int64
	Detail    string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"index"`
}

// TableName 表名。
func (OperationLog) TableName() string { return "sys_operation_logs" }
