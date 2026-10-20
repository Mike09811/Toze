// Package handler HTTP 处理器层。
package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/pkg/response"
	"github.com/Mike09811/Toze/server/internal/pkg/token"
	"github.com/Mike09811/Toze/server/internal/service"
)

// Handler 聚合所有依赖。
type Handler struct {
	Auth *AuthHandler
	User *UserHandler
	Role *RoleHandler
	Menu *MenuHandler
	I18n *I18nHandler
	Log  *LogHandler
	Paas *PaasHandler
}

// New 构建处理器集合。
func New(auth *service.AuthService, user *service.UserService, role *service.RoleService,
	menu *service.MenuService, i18n *service.I18nService, logSvc *service.LogService,
	paas *service.PaasService, logger *slog.Logger) *Handler {
	return &Handler{
		Auth: NewAuthHandler(auth),
		User: NewUserHandler(user),
		Role: NewRoleHandler(role),
		Menu: NewMenuHandler(menu),
		I18n: NewI18nHandler(i18n),
		Log:  NewLogHandler(logSvc),
		Paas: NewPaasHandler(paas),
	}
}

// bind 统一 JSON 绑定与错误响应。
func bind(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		response.Fail(c, errcode.InvalidParams, "请求参数格式错误: "+err.Error())
		return false
	}
	return true
}

// claims 获取当前登录用户声明。
func claims(c *gin.Context) *token.Claims {
	v, ok := c.Get("claims")
	if !ok {
		return nil
	}
	tc, _ := v.(*token.Claims)
	return tc
}

// currentUserID 当前用户 ID。
func currentUserID(c *gin.Context) uint64 {
	if tc := claims(c); tc != nil {
		return tc.UserID
	}
	return 0
}

// Health 健康检查。
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
