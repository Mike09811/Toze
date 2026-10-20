package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/Mike09811/Toze/server/internal/pkg/response"
	"github.com/Mike09811/Toze/server/internal/service"
)

// LogHandler 日志接口。
type LogHandler struct{ svc *service.LogService }

// NewLogHandler 构造函数。
func NewLogHandler(svc *service.LogService) *LogHandler { return &LogHandler{svc: svc} }

type logListReq struct {
	Username string `json:"username"`
	Path     string `json:"path"`
	Offset   int    `json:"offset"`
	Limit    int    `json:"limit"`
}

// List POST /api/system/log/list
func (h *LogHandler) List(c *gin.Context) {
	var req logListReq
	if !bind(c, &req) {
		return
	}
	list, total, err := h.svc.List(req.Username, req.Path, req.Offset, req.Limit)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKPage(c, list, total)
}
