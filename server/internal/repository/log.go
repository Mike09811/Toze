package repository

import (
	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/pagination"
)

// LogRepository 操作日志仓储。
type LogRepository struct{ db *gorm.DB }

// NewLogRepository 构造函数。
func NewLogRepository(db *gorm.DB) *LogRepository { return &LogRepository{db: db} }

// Create 写入日志。
func (r *LogRepository) Create(l *model.OperationLog) error { return r.db.Create(l).Error }

// List 分页查询日志。
func (r *LogRepository) List(username, path string, p pagination.Query) ([]model.OperationLog, int64, error) {
	q := r.db.Model(&model.OperationLog{})
	if username != "" {
		q = q.Where("username LIKE ?", "%"+username+"%")
	}
	if path != "" {
		q = q.Where("path LIKE ?", "%"+path+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var logs []model.OperationLog
	err := q.Order("id desc").Offset(p.Offset).Limit(p.Limit).Find(&logs).Error
	return logs, total, err
}
