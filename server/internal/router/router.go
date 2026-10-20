// Package router 注册所有 HTTP 路由。
package router

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/config"
	"github.com/Mike09811/Toze/server/internal/database"
	"github.com/Mike09811/Toze/server/internal/handler"
	"github.com/Mike09811/Toze/server/internal/middleware"
	"github.com/Mike09811/Toze/server/internal/pkg/token"
	"github.com/Mike09811/Toze/server/internal/repository"
	"github.com/Mike09811/Toze/server/internal/service"
)

// Deps 路由依赖。
type Deps struct {
	Config  *config.Config
	Logger  *slog.Logger
	DB      *gorm.DB
	Handler *handler.Handler
	JWT     *token.Manager
	Tokens  *repository.TokenRepository
	LogSvc  *service.LogService
}

// New 构建 gin.Engine。
func New(d Deps) *gin.Engine {
	if d.Config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(
		middleware.RequestID(),
		middleware.Recovery(d.Logger),
		middleware.AccessLog(d.Logger),
		middleware.CORS(d.Config.HTTP.CORSOrigins),
	)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})
	r.GET("/readyz", func(c *gin.Context) {
		if err := database.Health(d.DB); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db_unavailable", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready", "db": database.Stats(d.DB)})
	})

	api := r.Group("/api")
	api.POST("/login", middleware.RateLimit(10, time.Minute), d.Handler.Auth.Login)
	// GitHub OAuth 浏览器回调（带 code 换 token），无需登录态
	api.GET("/paas/github/callback", d.Handler.Paas.GitHubCallback)
	api.POST("/register", middleware.RateLimit(5, time.Minute), d.Handler.Auth.Register)

	secured := api.Group("")
	secured.Use(middleware.Auth(d.JWT, d.Tokens), middleware.Audit(d.LogSvc))
	{
		secured.GET("/user/info", d.Handler.Auth.Profile)
		secured.POST("/logout", d.Handler.Auth.Logout)
		secured.POST("/user/password", d.Handler.Auth.ChangePassword)

		// 查询类：登录即可
		secured.POST("/system/user/list", d.Handler.User.List)
		secured.POST("/system/role/list", d.Handler.Role.List)
		secured.POST("/system/role/all", d.Handler.Role.All)
		secured.POST("/system/menu/list", d.Handler.Menu.List)
		secured.POST("/system/i18n/list", d.Handler.I18n.List)
		secured.POST("/system/log/list", d.Handler.Log.List)

		// PaaS 查询类：登录即可
		secured.POST("/paas/server/list", d.Handler.Paas.ServerList)
		secured.POST("/paas/github/status", d.Handler.Paas.GitHubStatus)
		secured.POST("/paas/app/list", d.Handler.Paas.AppList)
		secured.POST("/paas/domain/list", d.Handler.Paas.DomainList)

		// 写入类：仅管理员
		admin := secured.Group("", middleware.RequireRoles("ROLE_ADMIN"))
		{
			admin.POST("/system/user/create", d.Handler.User.Create)
			admin.POST("/system/user/update", d.Handler.User.Update)
			admin.POST("/system/user/delete", d.Handler.User.Delete)
			admin.POST("/system/user/delete/multiple", d.Handler.User.DeleteMany)

			admin.POST("/system/role/create", d.Handler.Role.Create)
			admin.POST("/system/role/update", d.Handler.Role.Update)
			admin.POST("/system/role/delete", d.Handler.Role.Delete)
			admin.POST("/system/role/delete/multiple", d.Handler.Role.DeleteMany)
			admin.POST("/system/role/menu/update", d.Handler.Role.UpdateMenus)

			admin.POST("/system/menu/header/create", d.Handler.Menu.CreateHeader)
			admin.POST("/system/menu/sider/create", d.Handler.Menu.CreateSider)
			admin.POST("/system/menu/update", d.Handler.Menu.Update)
			admin.POST("/system/menu/delete", d.Handler.Menu.Delete)
			admin.POST("/system/menu/delete/multiple", d.Handler.Menu.DeleteMany)

			// PaaS 写入类：仅管理员
			admin.POST("/paas/server/create", d.Handler.Paas.ServerCreate)
			admin.POST("/paas/server/update", d.Handler.Paas.ServerUpdate)
			admin.POST("/paas/server/delete", d.Handler.Paas.ServerDelete)
			admin.POST("/paas/server/test", d.Handler.Paas.ServerTest)
			admin.POST("/paas/server/init", d.Handler.Paas.ServerInit)

			admin.POST("/paas/app/create", d.Handler.Paas.AppCreate)
			admin.POST("/paas/app/update", d.Handler.Paas.AppUpdate)
			admin.POST("/paas/app/delete", d.Handler.Paas.AppDelete)
			admin.POST("/paas/app/deploy", d.Handler.Paas.AppDeploy)
			admin.POST("/paas/app/start", d.Handler.Paas.AppStart)
			admin.POST("/paas/app/stop", d.Handler.Paas.AppStop)
			admin.POST("/paas/app/restart", d.Handler.Paas.AppRestart)
			admin.POST("/paas/app/logs", d.Handler.Paas.AppLogs)
			admin.POST("/paas/app/status", d.Handler.Paas.AppStatus)

			admin.POST("/paas/domain/create", d.Handler.Paas.DomainCreate)
			admin.POST("/paas/domain/delete", d.Handler.Paas.DomainDelete)

			admin.POST("/paas/github/auth/url", d.Handler.Paas.GitHubAuthURL)
			admin.POST("/paas/github/repos", d.Handler.Paas.GitHubRepos)
			admin.POST("/paas/github/accounts/default", d.Handler.Paas.GitHubSetDefault)
			admin.POST("/paas/github/accounts/delete", d.Handler.Paas.GitHubDeleteAccount)
		}
	}

	mountStatic(r, d.Config.HTTP.StaticDir)
	return r
}

// mountStatic 托管前端构建产物，支持 history 路由回退。
func mountStatic(r *gin.Engine, dir string) {
	if dir == "" {
		return
	}
	index := filepath.Join(dir, "index.html")
	if _, err := os.Stat(index); err != nil {
		return
	}
	r.Static("/static", filepath.Join(dir, "static"))
	r.Static("/img", filepath.Join(dir, "img"))
	r.Static("/fonts", filepath.Join(dir, "fonts"))

	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "接口不存在", "data": gin.H{}})
			return
		}
		c.File(index)
	})
}
