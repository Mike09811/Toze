package repository

import (
	"errors"

	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/pagination"
)

// PaasServerFilter 服务器列表筛选条件。
type PaasServerFilter struct {
	Name   string
	IP     string
	Status int
}

// PaasServerRepository 服务器仓储。
type PaasServerRepository struct{ db *gorm.DB }

// NewPaasServerRepository 构造函数。
func NewPaasServerRepository(db *gorm.DB) *PaasServerRepository {
	return &PaasServerRepository{db: db}
}

var paasServerSortColumns = map[string]string{
	"id": "id", "name": "name", "ip": "ip", "status": "status", "created_at": "created_at",
}

// List 分页查询。
func (r *PaasServerRepository) List(f PaasServerFilter, p pagination.Query) ([]model.PaasServer, int64, error) {
	q := r.db.Model(&model.PaasServer{})
	if f.Name != "" {
		q = q.Where("name LIKE ?", "%"+f.Name+"%")
	}
	if f.IP != "" {
		q = q.Where("ip LIKE ?", "%"+f.IP+"%")
	}
	if f.Status > 0 {
		q = q.Where("status = ?", f.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var servers []model.PaasServer
	err := q.Order(p.Order(paasServerSortColumns, "id desc")).
		Offset(p.Offset).Limit(p.Limit).
		Find(&servers).Error
	return servers, total, err
}

// FindByID 按主键查询。
func (r *PaasServerRepository) FindByID(id uint64) (*model.PaasServer, error) {
	var s model.PaasServer
	err := r.db.First(&s, id).Error
	if err != nil {
		return nil, paasServerNotFound(err)
	}
	return &s, nil
}

// UpdateStatus 更新在线状态与 Docker 版本。
func (r *PaasServerRepository) UpdateStatus(id uint64, status int, dockerVer string) error {
	return r.db.Model(&model.PaasServer{}).Where("id = ?", id).
		Updates(map[string]any{"status": status, "docker_ver": dockerVer}).Error
}

// UpdateError 记录最近一次失败原因。
func (r *PaasServerRepository) UpdateError(id uint64, status int, lastErr string) error {
	return r.db.Model(&model.PaasServer{}).Where("id = ?", id).
		Updates(map[string]any{"status": status, "last_error": lastErr}).Error
}

// Create 新增。
func (r *PaasServerRepository) Create(s *model.PaasServer) error {
	return r.db.Create(s).Error
}

// Update 保存全量字段。
func (r *PaasServerRepository) Update(s *model.PaasServer) error {
	return r.db.Save(s).Error
}

// Delete 删除。
func (r *PaasServerRepository) Delete(id uint64) error {
	return r.db.Delete(&model.PaasServer{}, id).Error
}

// CountApps 服务器下应用数量。
func (r *PaasServerRepository) CountApps(id uint64) (int64, error) {
	var count int64
	err := r.db.Model(&model.PaasApp{}).Where("server_id = ?", id).Count(&count).Error
	return count, err
}

// paasServerNotFound 统一转换记录不存在错误。
func paasServerNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
