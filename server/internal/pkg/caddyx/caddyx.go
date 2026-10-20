// Package caddyx 生成 Caddy 反向代理配置文件。
// 纯字符串拼接，无第三方依赖；证书申请与续期全部交给 Caddy 原生 on-demand 自动 HTTPS。
package caddyx

import (
	"fmt"
	"sort"
	"strings"
)

// Route 一条「域名 → 本机端口」的反代规则。
type Route struct {
	Domain string
	Port   int
}

// Generate 生成完整 Caddyfile 内容，按域名排序保证多次生成结果幂等。
func Generate(routes []Route) string {
	sorted := make([]Route, len(routes))
	copy(sorted, routes)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Domain < sorted[j].Domain })

	var b strings.Builder
	b.WriteString("# 由 Toze 自动生成，请勿手工修改\n\n")
	for _, r := range sorted {
		fmt.Fprintf(&b, "%s {\n    reverse_proxy 127.0.0.1:%d\n}\n\n", r.Domain, r.Port)
	}
	return b.String()
}
