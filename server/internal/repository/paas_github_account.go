package repository

import (
	"errors"

	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/model"
)

// PaasGitHubAccountRepository GitHub 授权账号仓储。
type PaasGitHubAccountRepository struct{ db *gorm.DB }

// NewPaasGitHubAccountRepository 构造函数。
func NewPaasGitHubAccountRepository(db *gorm.DB) *PaasGitHubAccountRepository {
	return &PaasGitHubAccountRepository{db: db}
}

// List 全部账号。
func (r *PaasGitHubAccountRepository) List() ([]model.PaasGitHubAccount, error) {
	var accounts []model.PaasGitHubAccount
	err := r.db.Order("is_default DESC, id ASC").Find(&accounts).Error
	return accounts, err
}

// FindByID 按主键查询。
func (r *PaasGitHubAccountRepository) FindByID(id uint64) (*model.PaasGitHubAccount, error) {
	var a model.PaasGitHubAccount
	err := r.db.First(&a, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

// FindByLogin 按 GitHub 登录名查询（重复授权时合并）。
func (r *PaasGitHubAccountRepository) FindByLogin(login string) (*model.PaasGitHubAccount, error) {
	var a model.PaasGitHubAccount
	err := r.db.Where("login = ?", login).First(&a).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

// Upsert 新增或更新（保持原 ID 与默认标记）。
func (r *PaasGitHubAccountRepository) Upsert(a *model.PaasGitHubAccount) error {
	return r.db.Save(a).Error
}

// SetDefault 设置默认账号：先清空全部，再标记目标。
func (r *PaasGitHubAccountRepository) SetDefault(id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.PaasGitHubAccount{}).Where("1 = 1").Update("is_default", false).Error; err != nil {
			return err
		}
		return tx.Model(&model.PaasGitHubAccount{}).Where("id = ?", id).Update("is_default", true).Error
	})
}

// Delete 删除账号；删掉的是默认账号时把默认转移给最早的一个。
func (r *PaasGitHubAccountRepository) Delete(id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var account model.PaasGitHubAccount
		if err := tx.First(&account, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if err := tx.Delete(&model.PaasGitHubAccount{}, id).Error; err != nil {
			return err
		}
		if account.IsDefault {
			return tx.Exec(`UPDATE paas_github_accounts SET is_default = 1
				WHERE id = (SELECT id FROM paas_github_accounts ORDER BY id ASC LIMIT 1)`).Error
		}
		return nil
	})
}

// Count 账号数量。
func (r *PaasGitHubAccountRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.PaasGitHubAccount{}).Count(&count).Error
	return count, err
}
