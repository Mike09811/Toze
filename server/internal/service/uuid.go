package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// NewUUID 生成用户唯一标识（32 位十六进制）。
func NewUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("u%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
