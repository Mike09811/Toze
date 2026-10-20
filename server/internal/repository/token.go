package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/model"
)

// TokenRepository 令牌黑名单仓储。
type TokenRepository struct{ db *gorm.DB }

// NewTokenRepository 构造函数。
func NewTokenRepository(db *gorm.DB) *TokenRepository { return &TokenRepository{db: db} }

// Revoke 注销令牌。
func (r *TokenRepository) Revoke(jti string, expire time.Time) error {
	return r.db.Create(&model.RevokedToken{JTI: jti, ExpiresAt: expire}).Error
}

// IsRevoked 令牌是否已注销。
func (r *TokenRepository) IsRevoked(jti string) (bool, error) {
	var count int64
	err := r.db.Model(&model.RevokedToken{}).Where("jti = ?", jti).Count(&count).Error
	return count > 0, err
}

// Cleanup 清理过期记录。
func (r *TokenRepository) Cleanup(now time.Time) error {
	return r.db.Where("expires_at < ?", now).Delete(&model.RevokedToken{}).Error
}
