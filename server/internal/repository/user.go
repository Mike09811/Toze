// Package repository 封装数据访问。
package repository

import (
	"errors"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/pagination"
)

// ErrNotFound 记录不存在。
var ErrNotFound = errors.New("record not found")

// UserFilter 用户列表筛选条件。
type UserFilter struct {
	Account  string
	Name     string
	Phone    string
	Mail     string
	ID       string
	Gender   int
	Status   int
	DateFrom *time.Time
	DateTo   *time.Time
}

// UserRepository 用户仓储。
type UserRepository struct{ db *gorm.DB }

// NewUserRepository 构造函数。
func NewUserRepository(db *gorm.DB) *UserRepository { return &UserRepository{db: db} }

// DB 暴露底层连接（供事务使用）。
func (r *UserRepository) DB() *gorm.DB { return r.db }

var userSortColumns = map[string]string{
	"id": "id", "name": "name", "account": "account",
	"date": "created_at", "status": "status", "created_at": "created_at",
}

// List 分页查询用户。
func (r *UserRepository) List(f UserFilter, p pagination.Query) ([]model.User, int64, error) {
	q := r.db.Model(&model.User{})
	if f.Account != "" {
		q = q.Where("account LIKE ?", "%"+f.Account+"%")
	}
	if f.Name != "" {
		q = q.Where("name LIKE ?", "%"+f.Name+"%")
	}
	if f.Phone != "" {
		q = q.Where("phone LIKE ?", "%"+f.Phone+"%")
	}
	if f.Mail != "" {
		q = q.Where("mail LIKE ?", "%"+f.Mail+"%")
	}
	if f.ID != "" {
		if id, err := strconv.ParseUint(f.ID, 10, 64); err == nil {
			q = q.Where("id = ?", id)
		}
	}
	if f.Gender > 0 {
		q = q.Where("gender = ?", f.Gender)
	}
	if f.Status > 0 {
		q = q.Where("status = ?", f.Status)
	}
	if f.DateFrom != nil {
		q = q.Where("created_at >= ?", *f.DateFrom)
	}
	if f.DateTo != nil {
		q = q.Where("created_at <= ?", *f.DateTo)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []model.User
	err := q.Preload("Roles").
		Order(p.Order(userSortColumns, "id desc")).
		Offset(p.Offset).Limit(p.Limit).
		Find(&users).Error
	return users, total, err
}

// FindByID 按主键查询。
func (r *UserRepository) FindByID(id uint64) (*model.User, error) {
	var u model.User
	err := r.db.Preload("Roles").First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, err
}

// FindByAccount 按账号查询。
func (r *UserRepository) FindByAccount(account string) (*model.User, error) {
	var u model.User
	err := r.db.Preload("Roles").Where("account = ?", account).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, err
}

// ExistsByAccount 账号是否存在（可排除指定 ID）。
func (r *UserRepository) ExistsByAccount(account string, excludeID uint64) (bool, error) {
	var count int64
	q := r.db.Model(&model.User{}).Where("account = ?", account)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	err := q.Count(&count).Error
	return count > 0, err
}

// Create 创建用户并绑定角色。
func (r *UserRepository) Create(u *model.User, roles []model.Role) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(u).Error; err != nil {
			return err
		}
		if len(roles) > 0 {
			return tx.Model(u).Association("Roles").Append(roles)
		}
		return nil
	})
}

// Update 更新用户基础字段并重置角色。
func (r *UserRepository) Update(u *model.User, roles []model.Role) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(u).Error; err != nil {
			return err
		}
		if roles != nil {
			return tx.Model(u).Association("Roles").Replace(roles)
		}
		return nil
	})
}

// UpdatePassword 更新密码。
func (r *UserRepository) UpdatePassword(id uint64, hashed string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("password", hashed).Error
}

// Delete 软删除用户。
func (r *UserRepository) Delete(id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		u := model.User{ID: id}
		if err := tx.Model(&u).Association("Roles").Clear(); err != nil {
			return err
		}
		return tx.Delete(&model.User{}, id).Error
	})
}

// DeleteMany 批量软删除。
func (r *UserRepository) DeleteMany(ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			u := model.User{ID: id}
			if err := tx.Model(&u).Association("Roles").Clear(); err != nil {
				return err
			}
		}
		return tx.Where("id IN ?", ids).Delete(&model.User{}).Error
	})
}

// CountByRole 统计使用某角色的用户数。
func (r *UserRepository) CountByRole(roleID uint64) (int64, error) {
	var count int64
	err := r.db.Table("sys_user_roles").Where("role_id = ?", roleID).Count(&count).Error
	return count, err
}
