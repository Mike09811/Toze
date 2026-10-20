package service

import (
	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/pagination"
	"github.com/Mike09811/Toze/server/internal/repository"
)

// LogService 操作日志服务。
type LogService struct {
	repo *repository.LogRepository
}

// NewLogService 构造函数。
func NewLogService(repo *repository.LogRepository) *LogService {
	return &LogService{repo: repo}
}

// Record 记录一条操作日志。
func (s *LogService) Record(l *model.OperationLog) {
	_ = s.repo.Create(l)
}

// List 分页查询。
func (s *LogService) List(username, path string, offset, limit int) ([]model.OperationLog, int64, error) {
	return s.repo.List(username, path, pagination.Parse(offset, limit, "", ""))
}
