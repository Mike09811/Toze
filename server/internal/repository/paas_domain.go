package repository

import (
	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/pagination"
)

// PaasDomainFilter 域名列表筛选条件。
type PaasDomainFilter struct {
	Domain string
	AppID  uint64
}

// PaasDomainRepository 域名仓储。
type PaasDomainRepository struct{ db *gorm.DB }

// NewPaasDomainRepository 构造函数。
func NewPaasDomainRepository(db *gorm.DB) *PaasDomainRepository { return &PaasDomainRepository{db: db} }

var paasDomainSortColumns = map[string]string{
	"id": "id", "domain": "domain", "created_at": "created_at",
}

// List 分页查询。
func (r *PaasDomainRepository) List(f PaasDomainFilter, p pagination.Query) ([]model.PaasDomain, int64, error) {
	q := r.db.Model(&model.PaasDomain{})
	if f.Domain != "" {
		q = q.Where("domain LIKE ?", "%"+f.Domain+"%")
	}
	if f.AppID > 0 {
		q = q.Where("app_id = ?", f.AppID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var domains []model.PaasDomain
	err := q.Order(p.Order(paasDomainSortColumns, "id desc")).
		Offset(p.Offset).Limit(p.Limit).
		Find(&domains).Error
	return domains, total, err
}

// FindByID 按主键查询。
func (r *PaasDomainRepository) FindByID(id uint64) (*model.PaasDomain, error) {
	var d model.PaasDomain
	err := r.db.First(&d, id).Error
	if err != nil {
		return nil, paasServerNotFound(err)
	}
	return &d, nil
}

// FindByDomain 按域名查询。
func (r *PaasDomainRepository) FindByDomain(domain string) (*model.PaasDomain, error) {
	var d model.PaasDomain
	err := r.db.Where("domain = ?", domain).First(&d).Error
	if err != nil {
		return nil, paasServerNotFound(err)
	}
	return &d, nil
}

// ListByAppIDs 查询一组应用绑定的全部域名。
func (r *PaasDomainRepository) ListByAppIDs(appIDs []uint64) ([]model.PaasDomain, error) {
	if len(appIDs) == 0 {
		return nil, nil
	}
	var domains []model.PaasDomain
	err := r.db.Where("app_id IN ?", appIDs).Find(&domains).Error
	return domains, err
}

// CountByApp 应用绑定域名数量。
func (r *PaasDomainRepository) CountByApp(appID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&model.PaasDomain{}).Where("app_id = ?", appID).Count(&count).Error
	return count, err
}

// Create 新增。
func (r *PaasDomainRepository) Create(d *model.PaasDomain) error { return r.db.Create(d).Error }

// Delete 删除。
func (r *PaasDomainRepository) Delete(id uint64) error {
	return r.db.Delete(&model.PaasDomain{}, id).Error
}
