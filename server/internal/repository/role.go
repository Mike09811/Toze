package repository

import (
	"errors"

	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/pagination"
)

// RoleRepository 角色仓储。
type RoleRepository struct{ db *gorm.DB }

// NewRoleRepository 构造函数。
func NewRoleRepository(db *gorm.DB) *RoleRepository { return &RoleRepository{db: db} }

var roleSortColumns = map[string]string{
	"id": "id", "name": "name", "created_at": "created_at", "default_role": "default_role",
}

// List 分页查询角色。
func (r *RoleRepository) List(p pagination.Query) ([]model.Role, int64, error) {
	q := r.db.Model(&model.Role{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var roles []model.Role
	err := q.Preload("Menus").
		Order(p.Order(roleSortColumns, "id asc")).
		Offset(p.Offset).Limit(p.Limit).
		Find(&roles).Error
	return roles, total, err
}

// All 查询全部角色（用于用户编辑下拉）。
func (r *RoleRepository) All() ([]model.Role, error) {
	var roles []model.Role
	err := r.db.Order("id asc").Find(&roles).Error
	return roles, err
}

// FindByID 主键查询（含菜单）。
func (r *RoleRepository) FindByID(id uint64) (*model.Role, error) {
	var role model.Role
	err := r.db.Preload("Menus").First(&role, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &role, err
}

// FindByIDs 批量查询。
func (r *RoleRepository) FindByIDs(ids []uint64) ([]model.Role, error) {
	var roles []model.Role
	if len(ids) == 0 {
		return roles, nil
	}
	err := r.db.Where("id IN ?", ids).Find(&roles).Error
	return roles, err
}

// FindByNames 按角色标识查询。
func (r *RoleRepository) FindByNames(names []string) ([]model.Role, error) {
	var roles []model.Role
	if len(names) == 0 {
		return roles, nil
	}
	err := r.db.Where("name IN ?", names).Find(&roles).Error
	return roles, err
}

// ExistsByName 角色标识是否存在。
func (r *RoleRepository) ExistsByName(name string, excludeID uint64) (bool, error) {
	var count int64
	q := r.db.Model(&model.Role{}).Where("name = ?", name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	err := q.Count(&count).Error
	return count > 0, err
}

// Create 创建角色。
func (r *RoleRepository) Create(role *model.Role, menus []model.Menu) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(role).Error; err != nil {
			return err
		}
		if len(menus) > 0 {
			return tx.Model(role).Association("Menus").Append(menus)
		}
		return nil
	})
}

// Update 更新角色。
func (r *RoleRepository) Update(role *model.Role, menus []model.Menu) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(role).Error; err != nil {
			return err
		}
		if menus != nil {
			return tx.Model(role).Association("Menus").Replace(menus)
		}
		return nil
	})
}

// UpdateMenus 覆盖角色的菜单。
func (r *RoleRepository) UpdateMenus(role *model.Role, menus []model.Menu) error {
	return r.db.Model(role).Association("Menus").Replace(menus)
}

// Delete 删除角色并清理关联。
func (r *RoleRepository) Delete(id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		role := model.Role{ID: id}
		if err := tx.Model(&role).Association("Menus").Clear(); err != nil {
			return err
		}
		return tx.Delete(&model.Role{}, id).Error
	})
}

// DeleteMany 批量删除。
func (r *RoleRepository) DeleteMany(ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			role := model.Role{ID: id}
			if err := tx.Model(&role).Association("Menus").Clear(); err != nil {
				return err
			}
		}
		return tx.Where("id IN ?", ids).Delete(&model.Role{}).Error
	})
}
