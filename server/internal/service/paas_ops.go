package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/pkg/githubx"
)

// ServerResourcesDTO 服务器实时资源。
type ServerResourcesDTO struct {
	LoadAvg     string `json:"load_avg"`    // 1 分钟负载
	Cores       int    `json:"cores"`       // CPU 核数
	MemUsedMB   int64  `json:"mem_used_mb"` // 已用内存
	MemTotalMB  int64  `json:"mem_total_mb"`
	DiskUsedPct int    `json:"disk_used_pct"` // 根分区使用率
	UptimeDays  int    `json:"uptime_days"`
}

// ServerResources 采集服务器实时资源（负载/内存/磁盘/在线时长）。
func (s *PaasService) ServerResources(ctx context.Context, id uint64) (*ServerResourcesDTO, error) {
	srv, err := s.loadServer(id)
	if err != nil {
		return nil, err
	}
	script := `LA=$(cut -d' ' -f1 /proc/loadavg)
CN=$(nproc)
MT=$(awk '/MemTotal/{print $2}' /proc/meminfo)
MF=$(awk '/MemAvailable/{print $2}' /proc/meminfo)
DU=$(df -P / | tail -1 | awk '{print $5}' | tr -d '%')
UP=$(cut -d. -f1 /proc/uptime)
echo "LA=$LA CN=$CN MT=$MT MF=$MF DU=$DU UP=$UP"`
	resCtx, cancel := context.WithTimeout(ctx, paasQuickTimeout)
	defer cancel()
	out, err := s.run(resCtx, srv, script)
	if err != nil {
		return nil, errcode.Newf(errcode.PaasSSHFailed, "采集资源失败: %v", err)
	}
	dto := &ServerResourcesDTO{}
	for _, kv := range strings.Fields(strings.TrimSpace(out)) {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "LA":
			dto.LoadAvg = parts[1]
		case "CN":
			dto.Cores, _ = strconv.Atoi(parts[1])
		case "MT":
			dto.MemTotalMB = toInt64(parts[1]) / 1024
		case "MF":
			// MF 为 MemAvailable（KB），已用 = 总量 - 可用
			dto.MemUsedMB = dto.MemTotalMB - toInt64(parts[1])/1024
		case "DU":
			dto.DiskUsedPct, _ = strconv.Atoi(parts[1])
		case "UP":
			dto.UptimeDays = int(toInt64(parts[1]) / 86400)
		}
	}
	return dto, nil
}

func toInt64(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

// GitHubSetupWebhook 用默认账号为仓库一键配置 push webhook（自动部署）。
func (s *PaasService) GitHubSetupWebhook(ctx context.Context, repo, publicBaseURL string) error {
	if !paasRepoPattern.MatchString(repo) {
		return errcode.New(errcode.InvalidParams, "仓库格式应为 owner/repo")
	}
	token, login, err := s.resolveGitHubToken(0)
	if err != nil {
		return err
	}
	if publicBaseURL == "" {
		return errcode.New(errcode.InvalidParams, "缺少回调地址，请从页面发起")
	}
	callback := strings.TrimRight(publicBaseURL, "/") + "/api/paas/github/webhook"
	if err := githubx.CreateRepoWebhook(ctx, token, repo, callback); err != nil {
		if mapped := mapGitHubAPIError(err); mapped != err {
			be := mapped.(*errcode.Error)
			return errcode.Newf(be.Code, "为 %s 配置 webhook 失败：%s", repo, be.Msg)
		}
		return errcode.Newf(errcode.Internal, "为 %s 配置 webhook 失败: %v", repo, err)
	}
	s.log.Info("仓库 webhook 已配置", "repo", repo, "account", login, "callback", callback)
	return nil
}

// DomainCertDTO 域名证书状态。
type DomainCertDTO struct {
	Domain string `json:"domain"`
	// ok: 证书文件已生成；pending: Caddy 尚未签发（DNS 未指向本机或刚绑定）
	Status string `json:"status"`
}

// ServerDomainCerts 检查某服务器上全部域名的 HTTPS 证书签发状态。
func (s *PaasService) ServerDomainCerts(ctx context.Context, serverID uint64) ([]DomainCertDTO, error) {
	srv, err := s.loadServer(serverID)
	if err != nil {
		return nil, err
	}
	apps, err := s.apps.ListByServer(serverID)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(apps))
	for i := range apps {
		ids = append(ids, apps[i].ID)
	}
	domains, err := s.domains.ListByAppIDs(ids)
	if err != nil {
		return nil, err
	}
	if len(domains) == 0 {
		return []DomainCertDTO{}, nil
	}
	var b strings.Builder
	b.WriteString("set -e\nSUDO=\"\"\n[ \"$(id -u)\" = 0 ] || SUDO=\"sudo -n\"\n")
	for i := range domains {
		d := domains[i].Domain
		b.WriteString(fmt.Sprintf("if $SUDO test -f \"/var/lib/caddy/certificates/acme-v02.api.letsencrypt.org-directory/%s/%s.crt\" 2>/dev/null; then echo \"%s ok\"; else echo \"%s pending\"; fi\n", d, d, d, d))
	}
	certCtx, cancel := context.WithTimeout(ctx, paasQuickTimeout)
	defer cancel()
	out, err := s.run(certCtx, srv, b.String())
	if err != nil {
		return nil, errcode.Newf(errcode.PaasSSHFailed, "查询证书状态失败: %v", err)
	}
	list := []DomainCertDTO{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 2 {
			list = append(list, DomainCertDTO{Domain: fields[0], Status: fields[1]})
		}
	}
	return list, nil
}
