// Package testutil 提供测试用的内存数据库与路由装配。
package testutil

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/config"
	"github.com/Mike09811/Toze/server/internal/database"
	"github.com/Mike09811/Toze/server/internal/handler"
	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/hash"
	"github.com/Mike09811/Toze/server/internal/pkg/token"
	"github.com/Mike09811/Toze/server/internal/repository"
	"github.com/Mike09811/Toze/server/internal/router"
	"github.com/Mike09811/Toze/server/internal/service"
)

var counter int64

// NewDB 每个用例独立的 SQLite 内存库。
func NewDB(t *testing.T) *gorm.DB {
	t.Helper()
	gin.SetMode(gin.TestMode)
	name := fmt.Sprintf("test_%d_%d", time.Now().UnixNano(), atomic.AddInt64(&counter, 1))
	db, err := database.OpenTest(name)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db
}

// Seed 写入最小可用数据：角色、菜单、管理员。
func Seed(t *testing.T, db *gorm.DB) {
	t.Helper()
	roles := []model.Role{
		{ID: 1, Name: "ROLE_ADMIN", Desc: "管理员"},
		{ID: 2, Name: "ROLE_USER", Desc: "普通用户"},
	}
	if err := db.Create(&roles).Error; err != nil {
		t.Fatalf("seed roles: %v", err)
	}

	menus := []model.Menu{
		{ID: 100, Title: "首页", Path: "/", Type: "header"},
		{ID: 101, ParentID: 100, Title: "Dashboard", Path: "/dashboard", Type: "sider"},
		{ID: 102, ParentID: 101, Title: "主控台", Path: "/dashboard/console", Type: "page"},
		{ID: 103, ParentID: 100, Title: "系统管理", Path: "/system", Type: "sider"},
		{ID: 104, ParentID: 103, Title: "用户管理", Path: "/system/user", Type: "page"},
	}
	if err := db.Create(&menus).Error; err != nil {
		t.Fatalf("seed menus: %v", err)
	}
	if err := db.Model(&roles[0]).Association("Menus").Append(menus); err != nil {
		t.Fatalf("bind menus: %v", err)
	}

	pwd, _ := hash.Make("admin")
	admin := model.User{
		ID: 1, UUID: "admin-uuid", Account: "admin", Password: pwd, Name: "Admin",
		Status: 1, Gender: 1, Region: []string{"北京市"},
	}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	if err := db.Model(&admin).Association("Roles").Append(&roles[0]); err != nil {
		t.Fatalf("bind admin role: %v", err)
	}
}

// Env 测试用配置。
func Env() *config.Config {
	return &config.Config{
		App:  config.AppConfig{Name: "test", Env: "test"},
		HTTP: config.HTTPConfig{Addr: ":0", CORSOrigins: []string{"*"}},
		DB:   config.DBConfig{Name: "test"},
		JWT:  config.JWTConfig{Secret: "test-secret-key-0123456789", Issuer: "test", TTL: time.Hour},
		Log:  config.LogConfig{Level: "error", Format: "text"},
	}
}

// NewRouter 装配完整路由。
func NewRouter(t *testing.T, db *gorm.DB) (*gin.Engine, *token.Manager) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	menuRepo := repository.NewMenuRepository(db)
	logRepo := repository.NewLogRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	i18nRepo := repository.NewI18nRepository(db)
	paasServerRepo := repository.NewPaasServerRepository(db)
	paasAppRepo := repository.NewPaasAppRepository(db)
	paasDomainRepo := repository.NewPaasDomainRepository(db)
	paasGitHubAccountRepo := repository.NewPaasGitHubAccountRepository(db)

	jwtMgr := token.New("test-secret-key-0123456789", "test", time.Hour)

	authSvc := service.NewAuthService(userRepo, roleRepo, tokenRepo, jwtMgr, log)
	userSvc := service.NewUserService(userRepo, roleRepo, log)
	roleSvc := service.NewRoleService(roleRepo, menuRepo, userRepo, log)
	menuSvc := service.NewMenuService(menuRepo, log)
	i18nSvc := service.NewI18nService(i18nRepo)
	logSvc := service.NewLogService(logRepo)
	paasSvc := service.NewPaasService(paasServerRepo, paasAppRepo, paasDomainRepo, paasGitHubAccountRepo, "test-aes-key", "", "", "", log)
	// HTTP 测试不连真实服务器，远程脚本一律返回成功占位
	paasSvc.SetExec(func(ctx context.Context, server *model.PaasServer, secret, script string) (string, error) {
		return "stubbed", nil
	})

	h := handler.New(authSvc, userSvc, roleSvc, menuSvc, i18nSvc, logSvc, paasSvc, log)
	engine := router.New(router.Deps{
		Config: Env(), Logger: log, DB: db, Handler: h, JWT: jwtMgr, Tokens: tokenRepo, LogSvc: logSvc,
	})
	return engine, jwtMgr
}
