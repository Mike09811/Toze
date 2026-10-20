package service

import (
	"context"
	"encoding/base64"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/caddyx"
	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/pkg/pagination"
	"github.com/Mike09811/Toze/server/internal/pkg/sshx"
	"github.com/Mike09811/Toze/server/internal/repository"
)

// PaasDomainQuery 域名列表查询条件。
type PaasDomainQuery struct {
	PaasListQuery
	Domain string
	AppID  uint64
}

// PaasDomainUpsert 域名绑定入参。
type PaasDomainUpsert struct {
	AppID  string
	Domain string
}

// 域名校验：常规主机名，必须至少包含一个点，不允许通配符与端口。
var paasDomainPattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`)

// ListDomains 分页查询域名，附带应用与服务器信息。
func (s *PaasService) ListDomains(q PaasDomainQuery) ([]PaasDomainDTO, int64, error) {
	p := pagination.Parse(q.Offset, q.Limit, q.SortKey, q.SortType)
	domains, total, err := s.domains.List(repository.PaasDomainFilter{
		Domain: q.Domain, AppID: q.AppID,
	}, p)
	if err != nil {
		return nil, 0, err
	}
	appByID := s.appMap()
	serverNames := s.serverNameMap()
	out := make([]PaasDomainDTO, 0, len(domains))
	for i := range domains {
		dto := PaasDomainDTO{
			ID:        strconv.FormatUint(domains[i].ID, 10),
			AppID:     strconv.FormatUint(domains[i].AppID, 10),
			Domain:    domains[i].Domain,
			CreatedAt: formatTime(domains[i].CreatedAt),
		}
		if app, ok := appByID[domains[i].AppID]; ok {
			dto.AppName = app.Name
			dto.HostPort = app.HostPort
			dto.ServerID = strconv.FormatUint(app.ServerID, 10)
			dto.ServerName = serverNames[app.ServerID]
		}
		out = append(out, dto)
	}
	return out, total, nil
}

// CreateDomain 绑定域名：落库后同步 Caddy，同步失败则回滚。
func (s *PaasService) CreateDomain(req PaasDomainUpsert) (*PaasDomainDTO, error) {
	appID, err := ParseID(req.AppID)
	if err != nil || appID == 0 {
		return nil, errcode.New(errcode.InvalidParams, "请选择要绑定的应用")
	}
	app, err := s.loadApp(appID)
	if err != nil {
		return nil, err
	}
	if _, err := s.loadServer(app.ServerID); err != nil {
		return nil, err
	}
	domain := strings.ToLower(strings.TrimSpace(req.Domain))
	if len(domain) > 253 || !paasDomainPattern.MatchString(domain) {
		return nil, errcode.New(errcode.InvalidParams, "域名格式不正确，示例：app.example.com")
	}
	if _, err := s.domains.FindByDomain(domain); err == nil {
		return nil, errcode.Newf(errcode.PaasDomainExists, "域名 %s 已被绑定", domain)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	record := &model.PaasDomain{AppID: appID, Domain: domain}
	if err := s.domains.Create(record); err != nil {
		return nil, err
	}
	srv, err := s.servers.FindByID(app.ServerID)
	if err != nil {
		return nil, err
	}
	if err := s.syncCaddy(context.Background(), srv); err != nil {
		// 同步失败回滚，避免库里有域名但线上没生效
		if delErr := s.domains.Delete(record.ID); delErr != nil {
			s.log.Warn("回滚域名绑定失败", "domain", domain, "error", delErr)
		}
		return nil, errcode.Newf(errcode.Internal, "域名已保存但 Caddy 同步失败，已回滚: %v", err)
	}
	dto, err := s.domainDTO(record.ID)
	if err != nil {
		return nil, err
	}
	return &dto, nil
}

// DeleteDomain 解绑域名并同步 Caddy。
func (s *PaasService) DeleteDomain(ctx context.Context, id uint64) error {
	domain, err := s.domains.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errcode.New(errcode.NotFound, "域名不存在")
		}
		return err
	}
	app, err := s.loadApp(domain.AppID)
	if err != nil {
		return err
	}
	if err := s.domains.Delete(id); err != nil {
		return err
	}
	if srv, err := s.servers.FindByID(app.ServerID); err == nil {
		if err := s.syncCaddy(ctx, srv); err != nil {
			return errcode.Newf(errcode.Internal, "域名已解绑，但 Caddy 同步失败: %v", err)
		}
	}
	return nil
}

// syncCaddy 重新生成目标服务器的完整 Caddyfile：写入 → 校验 → 热重载。
func (s *PaasService) syncCaddy(ctx context.Context, srv *model.PaasServer) error {
	apps, err := s.apps.ListByServer(srv.ID)
	if err != nil {
		return err
	}
	ids := make([]uint64, 0, len(apps))
	portByApp := make(map[uint64]int, len(apps))
	for i := range apps {
		ids = append(ids, apps[i].ID)
		portByApp[apps[i].ID] = apps[i].HostPort
	}
	domains, err := s.domains.ListByAppIDs(ids)
	if err != nil {
		return err
	}
	routes := make([]caddyx.Route, 0, len(domains))
	for i := range domains {
		if port, ok := portByApp[domains[i].AppID]; ok {
			routes = append(routes, caddyx.Route{Domain: domains[i].Domain, Port: port})
		}
	}
	content := caddyx.Generate(routes)
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	script := `set -e
SUDO=""
[ "$(id -u)" = 0 ] || SUDO="sudo -n"
echo ` + sshx.Quote(encoded) + ` | base64 -d | $SUDO tee /etc/caddy/Caddyfile >/dev/null
$SUDO caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile
$SUDO systemctl reload caddy
`

	syncCtx, cancel := context.WithTimeout(ctx, paasQuickTimeout)
	defer cancel()
	if _, err := s.run(syncCtx, srv, script); err != nil {
		return err
	}
	s.log.Info("Caddy 配置已同步", "server", srv.Name, "sites", len(routes))
	return nil
}

// appMap 全量应用 id → 模型映射（自用规模下全量查询足够）。
func (s *PaasService) appMap() map[uint64]model.PaasApp {
	apps, _, err := s.apps.List(repository.PaasAppFilter{}, pagination.Query{Limit: pagination.MaxLimit})
	if err != nil {
		return map[uint64]model.PaasApp{}
	}
	m := make(map[uint64]model.PaasApp, len(apps))
	for i := range apps {
		m[apps[i].ID] = apps[i]
	}
	return m
}

// domainDTO 组装单条域名响应。
func (s *PaasService) domainDTO(id uint64) (PaasDomainDTO, error) {
	dto := PaasDomainDTO{}
	d, err := s.domains.FindByID(id)
	if err != nil {
		return dto, err
	}
	dto.ID = strconv.FormatUint(d.ID, 10)
	dto.AppID = strconv.FormatUint(d.AppID, 10)
	dto.Domain = d.Domain
	dto.CreatedAt = formatTime(d.CreatedAt)
	if app, err := s.apps.FindByID(d.AppID); err == nil {
		dto.AppName = app.Name
		dto.HostPort = app.HostPort
		dto.ServerID = strconv.FormatUint(app.ServerID, 10)
		if srv, err := s.servers.FindByID(app.ServerID); err == nil {
			dto.ServerName = srv.Name
		}
	}
	return dto, nil
}
