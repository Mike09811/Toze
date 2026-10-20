package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/Mike09811/Toze/server/internal/pkg/response"
	"github.com/Mike09811/Toze/server/internal/service"
)

// I18nHandler 多语言接口。
type I18nHandler struct{ svc *service.I18nService }

// NewI18nHandler 构造函数。
func NewI18nHandler(svc *service.I18nService) *I18nHandler { return &I18nHandler{svc: svc} }

// List POST /api/system/i18n/list
func (h *I18nHandler) List(c *gin.Context) {
	msgs, err := h.svc.Messages()
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OK(c, msgs)
}
