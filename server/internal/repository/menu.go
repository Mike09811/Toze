package repository

import (
	"errors"

	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/model"
)

// MenuRepository 菜单仓储。
type MenuRepository struct{ db *gorm.DB }

// NewMenuRepository 构造函数。
func NewMenuRepository(db *gorm.DB) *MenuRepository { return &MenuRepository{db: db} }

// All 查询全部菜单（按父级、排序）。
func (r *MenuRepository) All() ([]*model.Menu, error) {
	var menus []*model.Menu
	err := r.db.Order("parent_id asc, sort asc, id asc").Find(&menus).Error
	return menus, err
}

// FindByID 主键查询。
func (r *MenuRepository) FindByID(id uint64) (*model.Menu, error) {
	var m model.Menu
	err := r.db.First(&m, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &m, err
}

// FindByIDs 批量查询。
func (r *MenuRepository) FindByIDs(ids []uint64) ([]model.Menu, error) {
	var menus []model.Menu
	if len(ids) == 0 {
		return menus, nil
	}
	err := r.db.Where("id IN ?", ids).Order("parent_id asc, sort asc").Find(&menus).Error
	return menus, err
}

// Create 创建菜单。
func (r *MenuRepository) Create(m *model.Menu) error { return r.db.Create(m).Error }

// Update 更新菜单。
func (r *MenuRepository) Update(m *model.Menu) error { return r.db.Save(m).Error }

// Delete 删除菜单（需先校验子节点与角色引用）。
func (r *MenuRepository) Delete(id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM sys_role_menus WHERE menu_id = ?", id).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Menu{}, id).Error
	})
}

// DeleteMany 批量删除。
func (r *MenuRepository) DeleteMany(ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM sys_role_menus WHERE menu_id IN ?", ids).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", ids).Delete(&model.Menu{}).Error
	})
}

// HasChildren 是否存在子菜单。
func (r *MenuRepository) HasChildren(id uint64) (bool, error) {
	var count int64
	err := r.db.Model(&model.Menu{}).Where("parent_id = ?", id).Count(&count).Error
	return count > 0, err
}

// CountRoles 引用该菜单的角色数量。
func (r *MenuRepository) CountRoles(id uint64) (int64, error) {
	var count int64
	err := r.db.Table("sys_role_menus").Where("menu_id = ?", id).Count(&count).Error
	return count, err
}
