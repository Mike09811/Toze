package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/pkg/pagination"
	"github.com/Mike09811/Toze/server/internal/pkg/secretbox"
	"github.com/Mike09811/Toze/server/internal/pkg/sshx"
	"github.com/Mike09811/Toze/server/internal/repository"
)

// SSHExec 在纳管服务器上执行脚本；secret 为解密后的凭证明文，可注入替换以便测试。
type SSHExec func(ctx context.Context, server *model.PaasServer, secret, script string) (string, error)

// SSHExecStream 同 SSHExec，额外把实时输出写入 progress（构建日志直播用）。
type SSHExecStream func(ctx context.Context, server *model.PaasServer, secret, script string, progress io.Writer) (string, error)

// PaasListQuery PaaS 列表通用查询条件。
type PaasListQuery struct {
	Offset   int
	Limit    int
	SortType string
	SortKey  string
}

// PaasServerQuery 服务器列表查询条件。
type PaasServerQuery struct {
	PaasListQuery
	Name   string
	IP     string
	Status int
}

// PaasServerUpsert 服务器新增 / 编辑入参。
type PaasServerUpsert struct {
	ID          string
	Name        string
	IP          string
	SSHPort     int
	SSHUser     string
	AuthType    int
	AuthContent string // 编辑时留空表示保持原凭证
}

// PaasService PaaS 部署平台业务逻辑：服务器纳管、容器部署、域名反代、GitHub 构建。
type PaasService struct {
	servers  *repository.PaasServerRepository
	apps     *repository.PaasAppRepository
	domains  *repository.PaasDomainRepository
	accounts *repository.PaasGitHubAccountRepository
	log      *slog.Logger
	aesKey   string
	// GitHub OAuth 应用凭证，留空表示未配置（仍可构建公开仓库）
	githubClientID     string
	githubClientSecret string
	exec               SSHExec
	execStream         SSHExecStream
}

// NewPaasService 构造函数；aesKey 用于 SSH 凭证落库加密。
func NewPaasService(
	servers *repository.PaasServerRepository,
	apps *repository.PaasAppRepository,
	domains *repository.PaasDomainRepository,
	accounts *repository.PaasGitHubAccountRepository,
	aesKey, githubClientID, githubClientSecret string,
	log *slog.Logger,
) *PaasService {
	s := &PaasService{
		servers: servers, apps: apps, domains: domains, accounts: accounts,
		log: log, aesKey: aesKey,
		githubClientID: githubClientID, githubClientSecret: githubClientSecret,
	}
	s.exec = defaultSSHExec
	s.execStream = defaultSSHExecStream
	return s
}

// SetExec 替换脚本执行器（测试注入用）。
func (s *PaasService) SetExec(fn SSHExec) { s.exec = fn }

// SetExecStream 替换流式脚本执行器（测试注入用）。
func (s *PaasService) SetExecStream(fn SSHExecStream) { s.execStream = fn }

// 默认脚本执行时长：部署/初始化走后台协程，给足镜像拉取时间。
const (
	paasQuickTimeout  = 30 * time.Second
	paasDeployTimeout = 10 * time.Minute
	paasDockerVerNone = "未安装"
)

func defaultSSHExec(ctx context.Context, server *model.PaasServer, secret, script string) (string, error) {
	return sshx.NewRunner().Exec(ctx, server.IP, server.SSHPort, server.SSHUser, server.AuthType, secret, script)
}

func defaultSSHExecStream(ctx context.Context, server *model.PaasServer, secret, script string, progress io.Writer) (string, error) {
	return sshx.NewRunner().ExecStream(ctx, server.IP, server.SSHPort, server.SSHUser, server.AuthType, secret, script, progress)
}

// run 解密凭证后在目标服务器上执行脚本。
func (s *PaasService) run(ctx context.Context, server *model.PaasServer, script string) (string, error) {
	secret, err := secretbox.Decrypt(s.aesKey, server.AuthContent)
	if err != nil {
		return "", errcode.Newf(errcode.Internal, "解密 SSH 凭证失败，请检查 PAAS_AES_KEY 配置: %v", err)
	}
	return s.exec(ctx, server, secret, script)
}

// ---------- 服务器管理 ----------

var paasIPPattern = regexp.MustCompile(`^[0-9a-zA-Z.:-]{3,64}$`)

// ListServers 分页查询服务器。
func (s *PaasService) ListServers(q PaasServerQuery) ([]PaasServerDTO, int64, error) {
	p := pagination.Parse(q.Offset, q.Limit, q.SortKey, q.SortType)
	servers, total, err := s.servers.List(repository.PaasServerFilter{
		Name: q.Name, IP: q.IP, Status: q.Status,
	}, p)
	if err != nil {
		return nil, 0, err
	}
	out := make([]PaasServerDTO, 0, len(servers))
	for i := range servers {
		dto := ToPaasServerDTO(&servers[i])
		if dto.DockerVer == "" {
			dto.DockerVer = paasDockerVerNone
		}
		dto.AppCount, _ = s.servers.CountApps(servers[i].ID)
		out = append(out, dto)
	}
	return out, total, nil
}

// CreateServer 新增纳管服务器。
func (s *PaasService) CreateServer(req PaasServerUpsert) (*PaasServerDTO, error) {
	srv, err := s.buildServer(req, nil)
	if err != nil {
		return nil, err
	}
	if req.AuthContent == "" {
		return nil, errcode.New(errcode.InvalidParams, "SSH 密码或私钥不能为空")
	}
	encrypted, err := secretbox.Encrypt(s.aesKey, req.AuthContent)
	if err != nil {
		return nil, errcode.Newf(errcode.Internal, "加密 SSH 凭证失败: %v", err)
	}
	srv.AuthContent = encrypted
	if err := s.servers.Create(srv); err != nil {
		return nil, err
	}
	created, err := s.servers.FindByID(srv.ID)
	if err != nil {
		return nil, err
	}
	dto := ToPaasServerDTO(created)
	return &dto, nil
}

// UpdateServer 编辑服务器，凭证留空表示保持不变。
func (s *PaasService) UpdateServer(req PaasServerUpsert) (*PaasServerDTO, error) {
	id, err := ParseID(req.ID)
	if err != nil {
		return nil, err
	}
	existing, err := s.servers.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errcode.New(errcode.NotFound, "服务器不存在")
		}
		return nil, err
	}
	srv, err := s.buildServer(req, existing)
	if err != nil {
		return nil, err
	}
	if req.AuthContent != "" {
		encrypted, err := secretbox.Encrypt(s.aesKey, req.AuthContent)
		if err != nil {
			return nil, errcode.Newf(errcode.Internal, "加密 SSH 凭证失败: %v", err)
		}
		srv.AuthContent = encrypted
	} else {
		srv.AuthContent = existing.AuthContent
	}
	if err := s.servers.Update(srv); err != nil {
		return nil, err
	}
	updated, err := s.servers.FindByID(id)
	if err != nil {
		return nil, err
	}
	dto := ToPaasServerDTO(updated)
	return &dto, nil
}

// DeleteServer 删除服务器；仍有应用时拒绝，避免孤儿容器。
func (s *PaasService) DeleteServer(id uint64) error {
	if _, err := s.servers.FindByID(id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errcode.New(errcode.NotFound, "服务器不存在")
		}
		return err
	}
	count, err := s.servers.CountApps(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errcode.Newf(errcode.Conflict, "该服务器下仍有 %d 个应用，请先删除应用", count)
	}
	return s.servers.Delete(id)
}

// TestServer 连通性测试：SSH 执行 docker 版本探测，成功则标记在线。
func (s *PaasService) TestServer(ctx context.Context, id uint64) (*PaasServerDTO, error) {
	srv, err := s.loadServer(id)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, paasQuickTimeout)
	defer cancel()

	out, err := s.run(ctx, srv, `if command -v docker >/dev/null 2>&1 && docker version >/dev/null 2>&1; then
    docker version --format '{{.Server.Version}}'
elif command -v docker >/dev/null 2>&1 && sudo -n docker version >/dev/null 2>&1; then
    echo no-perm
else
    echo no-docker
fi`)
	if err != nil {
		_ = s.servers.UpdateError(id, model.PaasServerOffline, err.Error())
		return nil, errcode.Newf(errcode.PaasSSHFailed, "连接失败: %v", err)
	}
	ver := strings.TrimSpace(lastLine(out))
	if ver == "no-docker" || ver == "no-perm" || ver == "" {
		ver = paasDockerVerNone
	}
	if err := s.servers.UpdateStatus(id, model.PaasServerOnline, ver); err != nil {
		return nil, err
	}
	updated, err := s.servers.FindByID(id)
	if err != nil {
		return nil, err
	}
	dto := ToPaasServerDTO(updated)
	return &dto, nil
}

// InitServer 一键初始化环境（Docker + Caddy）。耗时较长，后台执行，前端轮询状态。
func (s *PaasService) InitServer(id uint64) error {
	srv, err := s.loadServer(id)
	if err != nil {
		return err
	}
	if srv.Status == model.PaasServerIniting {
		return errcode.New(errcode.PaasBusy, "环境初始化正在进行中，请稍候")
	}
	if err := s.servers.UpdateError(id, model.PaasServerIniting, ""); err != nil {
		return err
	}
	go func(server model.PaasServer) {
		ctx, cancel := context.WithTimeout(context.Background(), paasDeployTimeout)
		defer cancel()
		out, err := s.run(ctx, &server, paasInitScript)
		if err != nil {
			s.log.Error("服务器环境初始化失败", "server", server.Name, "error", err)
			_ = s.servers.UpdateError(server.ID, model.PaasServerOffline, err.Error())
			return
		}
		ver := strings.TrimSpace(lastLine(out))
		if ver == "" {
			ver = paasDockerVerNone
		}
		if err := s.servers.UpdateStatus(server.ID, model.PaasServerOnline, ver); err != nil {
			s.log.Error("回填服务器状态失败", "server", server.Name, "error", err)
			return
		}
		s.log.Info("服务器环境初始化完成", "server", server.Name, "docker", ver)
	}(*srv)
	return nil
}

// paasInitScript 安装 Docker 与 Caddy（Debian / Ubuntu），幂等；
// 同时把当前 SSH 用户加入 docker 组，使后续容器操作无需 sudo。
// Caddy 优先走官方 apt 源，拉取失败（国内网络常见）自动回退 GitHub 静态二进制 + 自建 systemd 单元。
const paasInitScript = `set -e
SUDO=""
if [ "$(id -u)" != 0 ]; then
    SUDO="sudo -n"
fi
export DEBIAN_FRONTEND=noninteractive
if ! command -v docker >/dev/null 2>&1; then
    curl -fsSL https://get.docker.com | $SUDO sh
fi
$SUDO systemctl enable --now docker
if ! command -v git >/dev/null 2>&1; then
    $SUDO apt-get update -y >/dev/null 2>&1 || true
    $SUDO apt-get install -y git
fi
LOGIN_USER="$(id -un)"
if [ "$LOGIN_USER" != root ] && ! id -nG "$LOGIN_USER" | tr ' ' '\n' | grep -qx docker; then
    $SUDO usermod -aG docker "$LOGIN_USER"
fi
if ! command -v caddy >/dev/null 2>&1; then
    install_caddy() {
        $SUDO apt-get update -y >/dev/null 2>&1 || true
        if $SUDO apt-get install -y debian-keyring debian-archive-keyring apt-transport-https curl gnupg >/dev/null 2>&1 \
            && curl -1sLf 'https://dl.cloudflare.com/caddy/gpg' | $SUDO gpg --batch --yes --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg 2>/dev/null \
            && curl -1sLf 'https://dl.cloudflare.com/caddy/stable/debian.deb.txt' | $SUDO tee /etc/apt/sources.list.d/caddy-stable.list >/dev/null \
            && $SUDO apt-get update -y >/dev/null 2>&1 \
            && $SUDO apt-get install -y caddy >/dev/null 2>&1; then
            return 0
        fi
        echo "官方 apt 源不可达，改用 GitHub Release 安装 Caddy"
        ARCH="$(uname -m)"
        case "$ARCH" in
            x86_64) ARCH=amd64 ;;
            aarch64) ARCH=arm64 ;;
            *) echo "不支持的架构: $ARCH"; return 1 ;;
        esac
        VER="$(curl -fsSL -o /dev/null -w '%{url_effective}' https://github.com/caddyserver/caddy/releases/latest | sed 's|.*/tag/||')"
        VER="${VER#v}"
        if [ -z "$VER" ]; then
            echo "无法解析 Caddy 最新版本号"
            return 1
        fi
        URL="https://github.com/caddyserver/caddy/releases/download/v${VER}/caddy_${VER}_linux_${ARCH}.deb"
        curl -fsSL "$URL" -o /tmp/caddy.deb || curl -fsSL "https://gh-proxy.com/${URL}" -o /tmp/caddy.deb
        $SUDO dpkg -i /tmp/caddy.deb || { $SUDO apt-get update -y >/dev/null 2>&1; $SUDO apt-get install -f -y; }
    }
    install_caddy
fi
$SUDO systemctl enable --now caddy
$SUDO mkdir -p /etc/caddy
$SUDO docker version --format '{{.Server.Version}}'
`

// buildServer 校验并组装服务器字段。
func (s *PaasService) buildServer(req PaasServerUpsert, existing *model.PaasServer) (*model.PaasServer, error) {
	srv := &model.PaasServer{}
	if existing != nil {
		*srv = *existing
	}
	srv.Name = strings.TrimSpace(req.Name)
	if srv.Name == "" {
		return nil, errcode.New(errcode.InvalidParams, "服务器名称不能为空")
	}
	ip := strings.TrimSpace(req.IP)
	if !paasIPPattern.MatchString(ip) {
		return nil, errcode.New(errcode.InvalidParams, "服务器 IP 格式不正确")
	}
	srv.IP = ip
	if req.SSHPort == 0 {
		req.SSHPort = 22
	}
	if req.SSHPort < 1 || req.SSHPort > 65535 {
		return nil, errcode.New(errcode.InvalidParams, "SSH 端口必须在 1-65535 之间")
	}
	srv.SSHPort = req.SSHPort
	srv.SSHUser = strings.TrimSpace(req.SSHUser)
	if srv.SSHUser == "" {
		srv.SSHUser = "root"
	}
	if req.AuthType == 0 {
		req.AuthType = model.PaasAuthPassword
	}
	if req.AuthType != model.PaasAuthPassword && req.AuthType != model.PaasAuthPrivateKey {
		return nil, errcode.New(errcode.InvalidParams, "认证方式只支持密码或私钥")
	}
	srv.AuthType = req.AuthType
	if existing == nil {
		srv.Status = model.PaasServerOffline
	}
	return srv, nil
}

// loadServer 加载服务器，不存在时返回业务错误。
func (s *PaasService) loadServer(id uint64) (*model.PaasServer, error) {
	srv, err := s.servers.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errcode.New(errcode.NotFound, "服务器不存在")
		}
		return nil, err
	}
	return srv, nil
}

// loadApp 加载应用，不存在时返回业务错误。
func (s *PaasService) loadApp(id uint64) (*model.PaasApp, error) {
	app, err := s.apps.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errcode.New(errcode.NotFound, "应用不存在")
		}
		return nil, err
	}
	return app, nil
}

// ToPaasServerDTO 模型转响应。
func ToPaasServerDTO(s *model.PaasServer) PaasServerDTO {
	return PaasServerDTO{
		ID:        strconv.FormatUint(s.ID, 10),
		Name:      s.Name,
		IP:        s.IP,
		SSHPort:   s.SSHPort,
		SSHUser:   s.SSHUser,
		AuthType:  s.AuthType,
		Status:    s.Status,
		DockerVer: s.DockerVer,
		LastError: s.LastError,
		CreatedAt: formatTime(s.CreatedAt),
	}
}

// lastLine 取输出中最后一个非空行。
func lastLine(out string) string {
	lines := strings.Split(out, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return ""
}
