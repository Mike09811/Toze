// Admin Plus Gin 服务入口。
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Mike09811/Toze/server/internal/config"
	"github.com/Mike09811/Toze/server/internal/database"
	"github.com/Mike09811/Toze/server/internal/handler"
	"github.com/Mike09811/Toze/server/internal/logger"
	"github.com/Mike09811/Toze/server/internal/pkg/token"
	"github.com/Mike09811/Toze/server/internal/repository"
	"github.com/Mike09811/Toze/server/internal/router"
	"github.com/Mike09811/Toze/server/internal/service"
)

func main() {
	envFile := flag.String("env", ".env", "env 文件路径")
	flag.Parse()

	cfg, err := config.Load(*envFile)
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Log)
	slog.SetDefault(log)
	log.Info("启动 toze", "env", cfg.App.Env, "addr", cfg.HTTP.Addr)

	db, err := database.Open(cfg.DB, cfg.Log.Level == "debug")
	if err != nil {
		log.Error("连接数据库失败", "error", err)
		os.Exit(1)
	}
	if cfg.DB.AutoMigrate {
		if err := database.Migrate(db); err != nil {
			log.Error("数据库迁移失败", "error", err)
			os.Exit(1)
		}
		log.Info("数据库迁移完成")
	}
	if cfg.DB.Seed {
		if err := database.Seed(db); err != nil {
			log.Error("种子数据写入失败", "error", err)
			os.Exit(1)
		}
		log.Info("种子数据就绪")
	}
	if err := database.EnsurePaasMenus(db); err != nil {
		log.Error("PaaS 菜单补插失败", "error", err)
		os.Exit(1)
	}

	jwtMgr := token.New(cfg.JWT.Secret, cfg.JWT.Issuer, cfg.JWT.TTL)

	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	menuRepo := repository.NewMenuRepository(db)
	logRepo := repository.NewLogRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	i18nRepo := repository.NewI18nRepository(db)
	paasServerRepo := repository.NewPaasServerRepository(db)
	paasAppRepo := repository.NewPaasAppRepository(db)
	paasDomainRepo := repository.NewPaasDomainRepository(db)
	paasSettingRepo := repository.NewPaasSettingRepository(db)

	authSvc := service.NewAuthService(userRepo, roleRepo, tokenRepo, jwtMgr, log)
	userSvc := service.NewUserService(userRepo, roleRepo, log)
	roleSvc := service.NewRoleService(roleRepo, menuRepo, userRepo, log)
	menuSvc := service.NewMenuService(menuRepo, log)
	i18nSvc := service.NewI18nService(i18nRepo)
	logSvc := service.NewLogService(logRepo)
	paasSvc := service.NewPaasService(paasServerRepo, paasAppRepo, paasDomainRepo, paasSettingRepo,
		cfg.PaaS.AESKey, cfg.PaaS.GitHubClientID, cfg.PaaS.GitHubClientSecret, log)
	if cfg.PaaS.AESKey == "" {
		log.Warn("PAAS_AES_KEY 未配置，PaaS 服务器凭证将无法加密保存，请尽快在 .env 中设置")
	}

	h := handler.New(authSvc, userSvc, roleSvc, menuSvc, i18nSvc, logSvc, paasSvc, log)

	engine := router.New(router.Deps{
		Config: cfg, Logger: log, DB: db, Handler: h, JWT: jwtMgr, Tokens: tokenRepo, LogSvc: logSvc,
	})

	srv := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      engine,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	go func() {
		log.Info("HTTP 服务已启动", "addr", cfg.HTTP.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP 服务异常退出", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("收到退出信号，开始优雅关闭")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("优雅关闭失败", "error", err)
	}
	if err := tokenRepo.Cleanup(time.Now()); err != nil {
		log.Warn("清理过期令牌失败", "error", err)
	}
	log.Info("服务已退出")
}
