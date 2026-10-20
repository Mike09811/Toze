package caddyx

import (
	"strings"
	"testing"

	"github.com/Mike09811/Toze/server/internal/pkg/sshx"
)

func TestGenerateSortedAndStable(t *testing.T) {
	routes := []Route{
		{Domain: "b.example.com", Port: 8081},
		{Domain: "a.example.com", Port: 8080},
	}
	got := Generate(routes)
	want := "# 由 Toze 自动生成，请勿手工修改\n\n" +
		"a.example.com {\n    reverse_proxy 127.0.0.1:8080\n}\n\n" +
		"b.example.com {\n    reverse_proxy 127.0.0.1:8081\n}\n\n"
	if got != want {
		t.Fatalf("Generate 输出不符:\ngot:\n%s\nwant:\n%s", got, want)
	}
	// 乱序输入应产出相同内容（幂等）
	shuffled := []Route{{Domain: "a.example.com", Port: 8080}, {Domain: "b.example.com", Port: 8081}}
	if Generate(shuffled) != got {
		t.Fatalf("不同顺序输入应生成相同配置")
	}
}

func TestGenerateEmpty(t *testing.T) {
	got := Generate(nil)
	if !strings.Contains(got, "Toze") {
		t.Fatalf("空配置也应包含管理注释头")
	}
	if strings.Contains(got, "reverse_proxy") {
		t.Fatalf("空配置不应包含反代规则")
	}
}

func TestGenerateQuotesHostileDomain(t *testing.T) {
	// 非法域名正常会在 service 层被拦截，这里防御性验证生成器不做任何二次解释
	got := Generate([]Route{{Domain: sshx.Quote("evil'; rm -rf /"), Port: 80}})
	if !strings.Contains(got, `'evil'\''; rm -rf /' {`) {
		t.Fatalf("域名应原样输出: %s", got)
	}
}
