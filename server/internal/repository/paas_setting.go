package repository

import (
	"errors"

	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/model"
)

// PaasSettingRepository PaaS 设置仓储（单行表，ID 固定为 1）。
type PaasSettingRepository struct{ db *gorm.DB }

// NewPaasSettingRepository 构造函数。
func NewPaasSettingRepository(db *gorm.DB) *PaasSettingRepository {
	return &PaasSettingRepository{db: db}
}

// Get 读取全局设置，不存在时返回空结构。
func (r *PaasSettingRepository) Get() (*model.PaasSetting, error) {
	var s model.PaasSetting
	err := r.db.First(&s, 1).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &model.PaasSetting{ID: 1}, nil
		}
		return nil, err
	}
	return &s, nil
}

// Save 保存全局设置。
func (r *PaasSettingRepository) Save(s *model.PaasSetting) error {
	return r.db.Save(s).Error
}
