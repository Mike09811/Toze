// Package config 负责加载并校验服务配置。
// 配置优先级：环境变量 > .env 文件 > 内置默认值。
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config 是应用的完整配置。
type Config struct {
	App  AppConfig
	HTTP HTTPConfig
	DB   DBConfig
	JWT  JWTConfig
	Log  LogConfig
	PaaS PaaSConfig
}

// AppConfig 应用基础信息。
type AppConfig struct {
	Name string
	Env  string
}

// HTTPConfig HTTP 服务配置。
type HTTPConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	CORSOrigins     []string
	StaticDir       string
}

// DBConfig 数据库配置。
type DBConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	Charset         string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	AutoMigrate     bool
	Seed            bool
}

// JWTConfig 令牌配置。
type JWTConfig struct {
	Secret     string
	Issuer     string
	TTL        time.Duration
	RefreshTTL time.Duration
}

// LogConfig 日志配置。
type LogConfig struct {
	Level  string
	Format string // json | text
}

// PaaSConfig 自托管 PaaS 模块配置。
type PaaSConfig struct {
	// AESKey 用于 SSH 凭证落库加密（PAAS_AES_KEY），更换后已存凭证将无法解密
	AESKey string
	// GitHub OAuth 应用（GitHub Settings → Developer settings → OAuth Apps 创建）
	// 留空则无法一键授权，但仍可通过仓库全名构建公开仓库
	GitHubClientID     string
	GitHubClientSecret string
}

// Load 读取配置，envFile 不存在时忽略。
func Load(envFile string) (*Config, error) {
	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil && !os.IsNotExist(err) {
			// .env 缺失属于正常情况，其它错误（如格式错误）需要暴露
			if !strings.Contains(err.Error(), "no such file") {
				return nil, fmt.Errorf("load env file: %w", err)
			}
		}
	}

	cfg := &Config{
		App: AppConfig{
			Name: getString("APP_NAME", "toze"),
			Env:  getString("APP_ENV", "development"),
		},
		HTTP: HTTPConfig{
			Addr:            getString("HTTP_ADDR", ":8088"),
			ReadTimeout:     getDuration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
			ShutdownTimeout: getDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
			CORSOrigins:     getSlice("HTTP_CORS_ORIGINS", []string{"http://localhost:7802", "http://127.0.0.1:7802"}),
			StaticDir:       getString("HTTP_STATIC_DIR", "../web/dist"),
		},
		DB: DBConfig{
			Host:            getString("DB_HOST", "127.0.0.1"),
			Port:            getInt("DB_PORT", 33306),
			User:            getString("DB_USER", "root"),
			Password:        getString("DB_PASSWORD", "admin123"),
			Name:            getString("DB_NAME", "admin_plus"),
			Charset:         getString("DB_CHARSET", "utf8mb4"),
			MaxIdleConns:    getInt("DB_MAX_IDLE_CONNS", 10),
			MaxOpenConns:    getInt("DB_MAX_OPEN_CONNS", 100),
			ConnMaxLifetime: getDuration("DB_CONN_MAX_LIFETIME", time.Hour),
			AutoMigrate:     getBool("DB_AUTO_MIGRATE", true),
			Seed:            getBool("DB_SEED", true),
		},
		JWT: JWTConfig{
			Secret:     getString("JWT_SECRET", "change-me-in-production-please-32bytes"),
			Issuer:     getString("JWT_ISSUER", "toze"),
			TTL:        getDuration("JWT_TTL", 24*time.Hour),
			RefreshTTL: getDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		Log: LogConfig{
			Level:  getString("LOG_LEVEL", "info"),
			Format: getString("LOG_FORMAT", "text"),
		},
		PaaS: PaaSConfig{
			AESKey:             getString("PAAS_AES_KEY", ""),
			GitHubClientID:     getString("GITHUB_CLIENT_ID", ""),
			GitHubClientSecret: getString("GITHUB_CLIENT_SECRET", ""),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate 校验必填项。
func (c *Config) Validate() error {
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET 不能为空")
	}
	if !c.IsProduction() && len(c.JWT.Secret) < 16 {
		return fmt.Errorf("JWT_SECRET 至少 16 位")
	}
	if c.IsProduction() && len(c.JWT.Secret) < 32 {
		return fmt.Errorf("生产环境 JWT_SECRET 至少 32 位")
	}
	if c.DB.Name == "" {
		return fmt.Errorf("DB_NAME 不能为空")
	}
	if c.HTTP.Addr == "" {
		return fmt.Errorf("HTTP_ADDR 不能为空")
	}
	return nil
}

// IsProduction 是否生产环境。
func (c *Config) IsProduction() bool { return c.App.Env == "production" }

// DSN 生成 MySQL 连接串。
func (c DBConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local&multiStatements=true",
		c.User, c.Password, c.Host, c.Port, c.Name, c.Charset)
}

// DSNWithoutDatabase 用于首次创建数据库。
func (c DBConfig) DSNWithoutDatabase() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=%s&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.Charset)
}

func getString(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func getSlice(key string, def []string) []string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return def
}
