package repository

import (
	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/pagination"
)

// PaasAppFilter 应用列表筛选条件。
type PaasAppFilter struct {
	Name     string
	ServerID uint64
	Status   int
}

// PaasAppRepository 应用仓储。
type PaasAppRepository struct{ db *gorm.DB }

// NewPaasAppRepository 构造函数。
func NewPaasAppRepository(db *gorm.DB) *PaasAppRepository { return &PaasAppRepository{db: db} }

var paasAppSortColumns = map[string]string{
	"id": "id", "name": "name", "status": "status", "created_at": "created_at",
}

// List 分页查询。
func (r *PaasAppRepository) List(f PaasAppFilter, p pagination.Query) ([]model.PaasApp, int64, error) {
	q := r.db.Model(&model.PaasApp{})
	if f.Name != "" {
		q = q.Where("name LIKE ?", "%"+f.Name+"%")
	}
	if f.ServerID > 0 {
		q = q.Where("server_id = ?", f.ServerID)
	}
	if f.Status > 0 {
		q = q.Where("status = ?", f.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var apps []model.PaasApp
	err := q.Order(p.Order(paasAppSortColumns, "id desc")).
		Offset(p.Offset).Limit(p.Limit).
		Find(&apps).Error
	return apps, total, err
}

// FindByID 按主键查询。
func (r *PaasAppRepository) FindByID(id uint64) (*model.PaasApp, error) {
	var a model.PaasApp
	err := r.db.First(&a, id).Error
	if err != nil {
		return nil, paasServerNotFound(err)
	}
	return &a, nil
}

// FindByName 按应用名查询。
func (r *PaasAppRepository) FindByName(name string) (*model.PaasApp, error) {
	var a model.PaasApp
	err := r.db.Where("name = ?", name).First(&a).Error
	if err != nil {
		return nil, paasServerNotFound(err)
	}
	return &a, nil
}

// ExistsHostPort 同一服务器下宿主机端口是否被其它应用占用。
func (r *PaasAppRepository) ExistsHostPort(serverID uint64, port int, excludeID uint64) (bool, error) {
	var count int64
	q := r.db.Model(&model.PaasApp{}).Where("server_id = ? AND host_port = ?", serverID, port)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	err := q.Count(&count).Error
	return count > 0, err
}

// ListByServer 查询服务器下全部应用。
func (r *PaasAppRepository) ListByServer(serverID uint64) ([]model.PaasApp, error) {
	var apps []model.PaasApp
	err := r.db.Where("server_id = ?", serverID).Find(&apps).Error
	return apps, err
}

// UpdateStatus 仅更新状态字段。
func (r *PaasAppRepository) UpdateStatus(id uint64, status int) error {
	return r.db.Model(&model.PaasApp{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateDeployLog 更新构建日志（截断保留最后一段，防止超大日志撑爆字段）。
func (r *PaasAppRepository) UpdateDeployLog(id uint64, log string) error {
	const max = 4 << 20 // 4MB，mediumtext 上限内
	if len(log) > max {
		log = log[len(log)-max:]
	}
	return r.db.Model(&model.PaasApp{}).Where("id = ?", id).Update("deploy_log", log).Error
}

// UpdateDeployment 部署完成后回填容器与状态。
func (r *PaasAppRepository) UpdateDeployment(id uint64, containerID string, status int, lastErr string) error {
	return r.db.Model(&model.PaasApp{}).Where("id = ?", id).
		Updates(map[string]any{
			"container_id": containerID,
			"status":       status,
			"last_error":   lastErr,
		}).Error
}

// Create 新增。
func (r *PaasAppRepository) Create(a *model.PaasApp) error { return r.db.Create(a).Error }

// Update 保存全量字段。
func (r *PaasAppRepository) Update(a *model.PaasApp) error { return r.db.Save(a).Error }

// Delete 删除。
func (r *PaasAppRepository) Delete(id uint64) error { return r.db.Delete(&model.PaasApp{}, id).Error }

// ListBySource 按 GitHub 仓库与分支查找应用（webhook 自动部署用）。
func (r *PaasAppRepository) ListBySource(repo, branch string) ([]model.PaasApp, error) {
	var apps []model.PaasApp
	err := r.db.Where("source_repo = ? AND source_branch = ?", repo, branch).Find(&apps).Error
	return apps, err
}
