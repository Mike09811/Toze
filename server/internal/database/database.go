// Package database 负责数据库连接、迁移与种子数据。
package database

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/Mike09811/Toze/server/internal/config"
)

// Open 连接 MySQL；数据库不存在时自动创建。
func Open(cfg config.DBConfig, debug bool) (*gorm.DB, error) {
	level := gormlogger.Warn
	if debug {
		level = gormlogger.Info
	}
	gcfg := &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(level),
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN()), gcfg)
	if err != nil {
		if !isUnknownDatabase(err) {
			return nil, fmt.Errorf("connect mysql: %w", err)
		}
		if e := ensureDatabase(cfg); e != nil {
			return nil, fmt.Errorf("create database: %w", e)
		}
		db, err = gorm.Open(mysql.Open(cfg.DSN()), gcfg)
		if err != nil {
			return nil, fmt.Errorf("connect mysql after create: %w", err)
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	return db, nil
}

// OpenTest 使用内存 SQLite，便于单元测试，不依赖外部服务。
func OpenTest(name string) (*gorm.DB, error) {
	if name == "" {
		name = "default"
	}
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, err
	}
	if err := Migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

func ensureDatabase(cfg config.DBConfig) error {
	db, err := gorm.Open(mysql.Open(cfg.DSNWithoutDatabase()), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	stmt := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", cfg.Name)
	return db.Exec(stmt).Error
}

func isUnknownDatabase(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unknown database") || strings.Contains(msg, "error 1049")
}

// Health 健康检查。
func Health(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// Stats 返回连接池状态，便于运维接口输出。
func Stats(db *gorm.DB) map[string]any {
	sqlDB, err := db.DB()
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	s := sqlDB.Stats()
	return map[string]any{
		"open":     s.OpenConnections,
		"in_use":   s.InUse,
		"idle":     s.Idle,
		"max_open": s.MaxOpenConnections,
		"wait":     s.WaitCount,
		"time":     time.Now().Format(time.RFC3339),
	}
}

// ErrNotFound 统一未找到错误。
var ErrNotFound = errors.New("record not found")
