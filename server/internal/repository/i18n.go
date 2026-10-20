package repository

import (
	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/model"
)

// I18nRepository 多语言仓储。
type I18nRepository struct{ db *gorm.DB }

// NewI18nRepository 构造函数。
func NewI18nRepository(db *gorm.DB) *I18nRepository { return &I18nRepository{db: db} }

// All 查询全部词条。
func (r *I18nRepository) All() ([]model.I18nMessage, error) {
	var items []model.I18nMessage
	err := r.db.Order("locale asc, module asc, `key` asc").Find(&items).Error
	return items, err
}
