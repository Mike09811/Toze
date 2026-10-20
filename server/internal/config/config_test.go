package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Mike09811/Toze/server/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.HTTP.Addr != ":8088" || cfg.DB.Port != 33306 {
		t.Fatalf("unexpected defaults: %+v", cfg.HTTP)
	}
	if cfg.JWT.TTL != 24*time.Hour {
		t.Fatalf("unexpected ttl: %v", cfg.JWT.TTL)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9999")
	t.Setenv("DB_PORT", "3307")
	t.Setenv("JWT_TTL", "30m")
	t.Setenv("HTTP_CORS_ORIGINS", "http://a.com, http://b.com")

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.HTTP.Addr != ":9999" || cfg.DB.Port != 3307 || cfg.JWT.TTL != 30*time.Minute {
		t.Fatalf("env not applied: %+v", cfg)
	}
	if len(cfg.HTTP.CORSOrigins) != 2 {
		t.Fatalf("cors: %v", cfg.HTTP.CORSOrigins)
	}
}

func TestValidate(t *testing.T) {
	cfg, _ := config.Load("")
	cfg.JWT.Secret = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("empty secret should fail")
	}

	cfg.JWT.Secret = "short"
	cfg.App.Env = "production"
	if err := cfg.Validate(); err == nil {
		t.Fatal("production requires long secret")
	}
}

func TestDSN(t *testing.T) {
	cfg, _ := config.Load("")
	dsn := cfg.DB.DSN()
	if !strings.Contains(dsn, "tcp(127.0.0.1:33306)/admin_plus") {
		t.Fatalf("unexpected dsn: %s", dsn)
	}
	if !strings.Contains(cfg.DB.DSNWithoutDatabase(), "tcp(127.0.0.1:33306)/?") {
		t.Fatal("unexpected dsn without database")
	}
}
