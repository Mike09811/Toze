package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/pkg/pagination"
	"github.com/Mike09811/Toze/server/internal/pkg/secretbox"
	"github.com/Mike09811/Toze/server/internal/pkg/sshx"
	"github.com/Mike09811/Toze/server/internal/repository"
)

// PaasAppQuery 应用列表查询条件。
type PaasAppQuery struct {
	PaasListQuery
	Name     string
	ServerID uint64
	Status   int
}

// PaasAppUpsert 应用新增 / 编辑入参。
type PaasAppUpsert struct {
	ID              string
	ServerID        string
	Name            string
	Image           string // 直接部署现成镜像
	SourceRepo      string // GitHub 构建：owner/repo，与 Image 二选一
	SourceBranch    string // 构建分支，默认 main
	SourceAccountID uint64 // 构建用 GitHub 账号，0 表示默认/公开仓库
	ContainerPort   int
	HostPort        int
	Env             []string // 每项格式 K=V
	Volumes         []string // 每项格式 /host:/container[:ro|:rw]
	Restart         string
}

// 输入白名单校验：应用名即容器名，镜像 / 环境变量 / 挂载全部走单引号包裹 + 字符集约束。
var (
	paasAppNamePattern    = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
	paasImagePattern      = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9./:_@-]{0,254}$`)
	paasEnvPattern        = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=.+$`)
	paasVolumePattern     = regexp.MustCompile(`^[^\s:]+:[^\s:]+(:ro|:rw)?$`)
	paasContainerIDFormat = regexp.MustCompile(`^[0-9a-f]{12,64}$`)
)

var paasRestartPolicies = map[string]bool{"always": true, "unless-stopped": true, "on-failure": true, "no": true}

var (
	paasRepoPattern   = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	paasBranchPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]{1,100}$`)
)

// ListApps 分页查询应用，附带所属服务器名称。
func (s *PaasService) ListApps(q PaasAppQuery) ([]PaasAppDTO, int64, error) {
	p := pagination.Parse(q.Offset, q.Limit, q.SortKey, q.SortType)
	apps, total, err := s.apps.List(repository.PaasAppFilter{
		Name: q.Name, ServerID: q.ServerID, Status: q.Status,
	}, p)
	if err != nil {
		return nil, 0, err
	}
	serverNames := s.serverNameMap()
	out := make([]PaasAppDTO, 0, len(apps))
	for i := range apps {
		dto := ToPaasAppDTO(&apps[i])
		dto.ServerName = serverNames[apps[i].ServerID]
		dto.DeployLog = "" // 列表不回传构建日志，按需走 deploy-log 接口
		out = append(out, dto)
	}
	return out, total, nil
}

// CreateApp 新增应用（仅落库，不部署）。
func (s *PaasService) CreateApp(req PaasAppUpsert) (*PaasAppDTO, error) {
	app, err := s.buildApp(req, nil)
	if err != nil {
		return nil, err
	}
	if _, err := s.apps.FindByName(app.Name); err == nil {
		return nil, errcode.New(errcode.PaasAppExists, "")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	conflict, err := s.apps.ExistsHostPort(app.ServerID, app.HostPort, 0)
	if err != nil {
		return nil, err
	}
	if conflict {
		return nil, errcode.Newf(errcode.PaasPortConflict, "宿主机端口 %d 已被同服务器其它应用占用", app.HostPort)
	}
	if err := s.apps.Create(app); err != nil {
		return nil, err
	}
	created, err := s.apps.FindByID(app.ID)
	if err != nil {
		return nil, err
	}
	dto := ToPaasAppDTO(created)
	return &dto, nil
}

// UpdateApp 编辑应用配置，下次部署生效。
func (s *PaasService) UpdateApp(req PaasAppUpsert) (*PaasAppDTO, error) {
	id, err := ParseID(req.ID)
	if err != nil {
		return nil, err
	}
	existing, err := s.loadApp(id)
	if err != nil {
		return nil, err
	}
	app, err := s.buildApp(req, existing)
	if err != nil {
		return nil, err
	}
	if other, err := s.apps.FindByName(app.Name); err == nil {
		if other.ID != id {
			return nil, errcode.New(errcode.PaasAppExists, "")
		}
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	conflict, err := s.apps.ExistsHostPort(app.ServerID, app.HostPort, id)
	if err != nil {
		return nil, err
	}
	if conflict {
		return nil, errcode.Newf(errcode.PaasPortConflict, "宿主机端口 %d 已被同服务器其它应用占用", app.HostPort)
	}
	if err := s.apps.Update(app); err != nil {
		return nil, err
	}
	updated, err := s.apps.FindByID(id)
	if err != nil {
		return nil, err
	}
	dto := ToPaasAppDTO(updated)
	return &dto, nil
}

// DeleteApp 删除应用：级联删除域名绑定、清理远程容器并同步 Caddy。
func (s *PaasService) DeleteApp(ctx context.Context, id uint64) error {
	app, err := s.loadApp(id)
	if err != nil {
		return err
	}
	srv, srvErr := s.servers.FindByID(app.ServerID)

	// 容器清理尽力而为：服务器离线时仍允许删除记录
	if srvErr == nil {
		cleanCtx, cancel := context.WithTimeout(ctx, paasQuickTimeout)
		defer cancel()
		if _, err := s.run(cleanCtx, srv, "docker rm -f "+sshx.Quote(app.Name)+" >/dev/null 2>&1 || true"); err != nil {
			s.log.Warn("删除远程容器失败，继续清理数据", "app", app.Name, "error", err)
		}
	}

	// 先删数据再同步配置，保证生成的 Caddyfile 已不含该应用的域名
	domains, _ := s.domains.ListByAppIDs([]uint64{id})
	if err := s.apps.Delete(id); err != nil {
		return err
	}
	for i := range domains {
		if err := s.domains.Delete(domains[i].ID); err != nil {
			s.log.Warn("删除域名绑定失败", "domain", domains[i].Domain, "error", err)
		}
	}
	if srvErr == nil && len(domains) > 0 {
		if err := s.syncCaddy(ctx, srv); err != nil {
			s.log.Warn("删除应用后同步 Caddy 失败", "app", app.Name, "error", err)
		}
	}
	return nil
}

// DeployApp 部署 / 重新部署应用。镜像拉取耗时较长，后台执行，前端轮询状态。
func (s *PaasService) DeployApp(id uint64) error {
	app, err := s.loadApp(id)
	if err != nil {
		return err
	}
	if app.Status == model.PaasAppDeploying {
		return errcode.New(errcode.PaasBusy, "该应用正在部署中，请稍候")
	}
	srv, err := s.loadServer(app.ServerID)
	if err != nil {
		return err
	}
	// GitHub 构建模式：私有仓库需要授权 token，公开仓库未授权也可构建
	githubToken := ""
	if app.SourceRepo != "" {
		if t, _, err := s.resolveGitHubToken(app.SourceAccountID); err == nil {
			githubToken = t
		}
	}
	script := s.buildDeployScript(app, githubToken)
	if err := s.apps.UpdateDeployment(id, app.ContainerID, model.PaasAppDeploying, ""); err != nil {
		return err
	}
	go func(app model.PaasApp, server model.PaasServer) {
		ctx, cancel := context.WithTimeout(context.Background(), paasDeployTimeout)
		defer cancel()

		// 构建输出实时落库，前端轮询 deploy-log 即可看到进度
		live := &paasLiveLog{appID: app.ID, apps: s.apps}
		secret, err := secretbox.Decrypt(s.aesKey, server.AuthContent)
		if err != nil {
			_ = s.apps.UpdateDeployment(app.ID, "", model.PaasAppUndeployed, err.Error())
			return
		}
		out, err := s.execStream(ctx, &server, secret, script, live)
		live.flush() // 终态：把完整输出写库一次
		if err != nil {
			s.log.Error("部署失败", "app", app.Name, "error", err)
			_ = s.apps.UpdateDeployment(app.ID, "", model.PaasAppUndeployed, classifyDeployError(err))
			return
		}
		cid := lastLine(out)
		if !paasContainerIDFormat.MatchString(cid) {
			_ = s.apps.UpdateDeployment(app.ID, "", model.PaasAppUndeployed, "[平台异常] 无法解析容器 ID，远程输出: "+strings.TrimSpace(out))
			return
		}
		if err := s.apps.UpdateDeployment(app.ID, cid, model.PaasAppRunning, ""); err != nil {
			s.log.Error("回填部署结果失败", "app", app.Name, "error", err)
			return
		}
		s.log.Info("部署完成", "app", app.Name, "container", cid)
	}(*app, *srv)
	return nil
}

// paasLiveLog 线程安全的构建日志缓冲：边写入边节流落库。
type paasLiveLog struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	appID  uint64
	apps   *repository.PaasAppRepository
	lastAt time.Time
}

// ansiPattern 终端转义序列（颜色码等），落库前剥离避免日志乱码。
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

func (l *paasLiveLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// 剥离 ANSI 后写入；按 io.Writer 契约必须返回 len(p)，
	// 否则剥离导致 n < len(p) 会被 ssh 通道判定为 short write
	_, _ = l.buf.Write(ansiPattern.ReplaceAll(p, nil))
	// 节流：每 2 秒同步落库一次（避免异步竞态覆盖终态日志）
	if time.Since(l.lastAt) > 2*time.Second {
		l.lastAt = time.Now()
		_ = l.apps.UpdateDeployLog(l.appID, l.buf.String())
	}
	return len(p), nil
}

// flush 结束时同步写全量。
func (l *paasLiveLog) flush() {
	l.mu.Lock()
	defer l.mu.Unlock()
	_ = l.apps.UpdateDeployLog(l.appID, l.buf.String())
}

// buildDeployScript 拼接部署脚本，所有动态值均单引号包裹。
// 镜像模式直接 docker run；GitHub 模式先在服务器上 clone + docker build。
// 无感切换：新版本先在临时端口启动并完成就绪检查（TCP 可连通），
// 确认可用后才移除旧容器并用同镜像秒级拉起正式容器；失败则旧版本不受影响。
func (s *PaasService) buildDeployScript(app *model.PaasApp, githubToken string) string {
	var b strings.Builder
	b.WriteString("set -e\n")
	// 任何一步失败都打上醒目终止标记并注明阶段，方便在日志里一眼定位与归因
	b.WriteString("STAGE=部署准备\n")
	b.WriteString("trap 'echo \"!!!! 部署失败于 [$STAGE] 阶段：上方最近的 error/failed 行即为原因\"' ERR\n")

	image := app.Image
	if app.SourceRepo != "" {
		cloneURL := "https://github.com/" + app.SourceRepo + ".git"
		if githubToken != "" {
			cloneURL = "https://x-access-token:" + githubToken + "@github.com/" + app.SourceRepo + ".git"
		}
		branch := app.SourceBranch
		if branch == "" {
			branch = "main"
		}
		image = "toze/" + app.Name + ":latest"
		// 构建目录放登录用户家目录下，root / 普通用户都有写权限；应用名已过白名单校验
		b.WriteString("STAGE='拉取代码'\n")
		b.WriteString("echo ==== [1/4] 拉取代码（git clone）====\n")
		b.WriteString("WORK=\"$HOME\"/toze/build/" + app.Name + "\n")
		b.WriteString("rm -rf \"$WORK\" && mkdir -p \"$WORK\"\n")
		b.WriteString("git clone --depth 1 -b " + sshx.Quote(branch) + " " + sshx.Quote(cloneURL) + " \"$WORK\"\n")
		// 抹掉 remote 中的 token，避免明文留在服务器的 .git/config
		b.WriteString("git -C \"$WORK\" remote remove origin\n")
		b.WriteString("cd \"$WORK\"\n")
		b.WriteString("STAGE='构建镜像'\n")
		b.WriteString("echo ==== [2/4] 构建镜像（docker build）====\n")
		b.WriteString("docker build --build-arg GOPROXY=https://goproxy.cn,direct -t " + sshx.Quote(image) + " .\n")
	}

	// 公共 run 参数（--restart/-e/-v/镜像），端口与 --name 由调用处单独拼
	runArgsNoPort := func() string {
		var r strings.Builder
		r.WriteString(" --restart " + sshx.Quote(app.Restart))
		for _, e := range decodeJSONStrings(app.Env) {
			r.WriteString(" -e " + sshx.Quote(e))
		}
		for _, v := range decodeJSONStrings(app.Volumes) {
			r.WriteString(" -v " + sshx.Quote(v))
		}
		return r.String()
	}
	imageArg := " " + sshx.Quote(image)
	// -p 表达式；portExpr 为纯数字或 $P 形式的 shell 变量
	portArg := func(portExpr string) string {
		if strings.HasPrefix(portExpr, "$") {
			return " -p \"" + portExpr + ":" + strconv.Itoa(app.ContainerPort) + "\""
		}
		return " -p " + sshx.Quote(portExpr+":"+strconv.Itoa(app.ContainerPort))
	}

	nextName := app.Name + "-next"
	b.WriteString("STAGE='新版本启动检查'\n")
	b.WriteString("echo ==== [3/4] 启动新版本并等待就绪 ====\n")
	b.WriteString("NEXT=" + sshx.Quote(nextName) + "\n")
	b.WriteString("docker rm -f \"$NEXT\" >/dev/null 2>&1 || true\n")
	// 随机挑一个可绑定的临时端口
	b.WriteString("TMP_PORT=\"\"\n")
	b.WriteString("for P in $(shuf -i 20000-45000 -n 5); do\n")
	b.WriteString("    if docker run -d --name \"$NEXT\"" + runArgsNoPort() + portArg("$P") + imageArg + " >/dev/null 2>&1; then TMP_PORT=$P; break; fi\n")
	b.WriteString("    docker rm -f \"$NEXT\" >/dev/null 2>&1 || true\n")
	b.WriteString("done\n")
	b.WriteString("if [ -z \"$TMP_PORT\" ]; then echo 临时端口分配失败; exit 1; fi\n")
	b.WriteString("echo \"--> 新版本已启动（临时端口 $TMP_PORT），等待就绪...\"\n")
	// 就绪检查：容器存活 + TCP 可连通，最多 90 秒；失败保留旧版本
	// 镜像自带 HEALTHCHECK 时以 healthy 为准（数据库类镜像）；
	// 否则 running + 直连容器内网 IP 的容器端口（宿主端口映射有 docker-proxy 兜底，连宿主端口永远通，测不出真实状态）
	b.WriteString("HAS_HC=$(docker inspect -f '{{if .Config.Healthcheck}}1{{end}}' \"$NEXT\" 2>/dev/null || true)\n")
	b.WriteString("CIP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' \"$NEXT\" 2>/dev/null)\n")
	// 就绪等待：每 5 秒把新容器新增日志转储进构建日志（| 前缀），应用报错即时可见
	b.WriteString("READY=0\n")
	b.WriteString("SINCE=0\n")
	b.WriteString("for I in $(seq 1 90); do\n")
	b.WriteString("    STATE=$(docker inspect -f {{.State.Status}} \"$NEXT\" 2>/dev/null)\n")
	b.WriteString("    if [ \"$HAS_HC\" = 1 ]; then\n")
	b.WriteString("        if [ \"$(docker inspect -f '{{.State.Health.Status}}' \"$NEXT\" 2>/dev/null)\" = healthy ]; then READY=1; break; fi\n")
	b.WriteString("    elif [ \"$STATE\" = running ] && [ -n \"$CIP\" ] && (echo > /dev/tcp/\"$CIP\"/" + strconv.Itoa(app.ContainerPort) + " ) 2>/dev/null; then READY=1; break; fi\n")
	b.WriteString("    if [ \"$STATE\" = exited ] || [ \"$STATE\" = dead ]; then break; fi\n")
	b.WriteString("    if [ $((I % 5)) = 0 ]; then docker logs --since \"${SINCE}s\" \"$NEXT\" 2>&1 | sed 's/^/    | /' || true; SINCE=5; fi\n")
	b.WriteString("    sleep 1\n")
	b.WriteString("done\n")
	b.WriteString("if [ \"$READY\" != 1 ]; then\n")
	b.WriteString("    echo !!!! 新容器90秒内未就绪，旧版本保持服务不受影响。新容器日志：\n")
	b.WriteString("    docker logs --tail 50 \"$NEXT\" 2>&1 || true\n")
	b.WriteString("    docker rm -f \"$NEXT\" >/dev/null 2>&1 || true\n")
	b.WriteString("    exit 1\n")
	b.WriteString("fi\n")
	b.WriteString("STAGE='切换流量'\n")
	b.WriteString("echo ==== [4/4] 新版本就绪，切换流量 ====\n")
	// 替换：镜像已验证并预热，正式容器秒级启动，业务中断约 1 秒以内
	b.WriteString("docker rm -f " + sshx.Quote(app.Name) + " >/dev/null 2>&1 || true\n")
	b.WriteString("cid=$(docker run -d --name " + sshx.Quote(app.Name) + runArgsNoPort() + portArg(strconv.Itoa(app.HostPort)) + imageArg + ")\n")
	b.WriteString("docker rm -f \"$NEXT\" >/dev/null 2>&1 || true\n")
	b.WriteString("echo ==== 部署完成，新版本运行中 ====\n")
	b.WriteString("echo \"$cid\"\n")
	return b.String()
}

// StartApp 启动容器。
func (s *PaasService) StartApp(ctx context.Context, id uint64) error {
	return s.containerAction(ctx, id, "start", model.PaasAppRunning)
}

// StopApp 停止容器。
func (s *PaasService) StopApp(ctx context.Context, id uint64) error {
	return s.containerAction(ctx, id, "stop", model.PaasAppStopped)
}

// RestartApp 重启容器。
func (s *PaasService) RestartApp(ctx context.Context, id uint64) error {
	return s.containerAction(ctx, id, "restart", model.PaasAppRunning)
}

func (s *PaasService) containerAction(ctx context.Context, id uint64, action string, status int) error {
	app, err := s.loadApp(id)
	if err != nil {
		return err
	}
	if app.ContainerID == "" {
		return errcode.New(errcode.InvalidParams, "应用尚未部署，请先执行部署")
	}
	if app.Status == model.PaasAppDeploying {
		return errcode.New(errcode.PaasBusy, "该应用正在部署中，请稍候")
	}
	srv, err := s.loadServer(app.ServerID)
	if err != nil {
		return err
	}
	cmd := "docker " + action + " " + sshx.Quote(app.Name)
	if action == "stop" {
		cmd = "docker stop -t 5 " + sshx.Quote(app.Name)
	}
	actCtx, cancel := context.WithTimeout(ctx, paasQuickTimeout)
	defer cancel()
	if _, err := s.run(actCtx, srv, cmd); err != nil {
		return errcode.Newf(errcode.PaasSSHFailed, "执行 docker %s 失败: %v", action, err)
	}
	return s.apps.UpdateStatus(id, status)
}

// AppDeployLog 查看最近一次构建日志。
func (s *PaasService) AppDeployLog(id uint64) (*PaasAppDTO, error) {
	app, err := s.loadApp(id)
	if err != nil {
		return nil, err
	}
	dto := ToPaasAppDTO(app)
	return &dto, nil
}

// AppLogs 查看容器最近日志。
func (s *PaasService) AppLogs(ctx context.Context, id uint64, tail int) (string, error) {
	app, err := s.loadApp(id)
	if err != nil {
		return "", err
	}
	if app.ContainerID == "" {
		return "", errcode.New(errcode.InvalidParams, "应用尚未部署，没有日志")
	}
	srv, err := s.loadServer(app.ServerID)
	if err != nil {
		return "", err
	}
	if tail <= 0 || tail > 5000 {
		tail = 1000
	}
	logCtx, cancel := context.WithTimeout(ctx, paasQuickTimeout)
	defer cancel()
	script := "docker logs --tail " + strconv.Itoa(tail) + " " + sshx.Quote(app.Name) + " 2>&1"
	out, err := s.run(logCtx, srv, script)
	if err != nil {
		return "", errcode.Newf(errcode.PaasSSHFailed, "获取日志失败: %v", err)
	}
	return out, nil
}

// RefreshAppStatus 与服务器上的实际容器状态对账。
func (s *PaasService) RefreshAppStatus(ctx context.Context, id uint64) (*PaasAppDTO, error) {
	app, err := s.loadApp(id)
	if err != nil {
		return nil, err
	}
	srv, err := s.loadServer(app.ServerID)
	if err != nil {
		return nil, err
	}
	statusCtx, cancel := context.WithTimeout(ctx, paasQuickTimeout)
	defer cancel()
	script := "docker inspect -f '{{.State.Status}}' " + sshx.Quote(app.Name) + " 2>/dev/null || echo gone"
	out, err := s.run(statusCtx, srv, script)
	if err != nil {
		return nil, errcode.Newf(errcode.PaasSSHFailed, "查询容器状态失败: %v", err)
	}
	state := strings.TrimSpace(lastLine(out))
	status := model.PaasAppUndeployed
	switch state {
	case "running":
		status = model.PaasAppRunning
	case "exited", "created", "paused", "restarting", "dead":
		status = model.PaasAppStopped
	}
	if err := s.apps.UpdateStatus(id, status); err != nil {
		return nil, err
	}
	updated, err := s.apps.FindByID(id)
	if err != nil {
		return nil, err
	}
	dto := ToPaasAppDTO(updated)
	return &dto, nil
}

// buildApp 校验并组装应用字段。
func (s *PaasService) buildApp(req PaasAppUpsert, existing *model.PaasApp) (*model.PaasApp, error) {
	app := &model.PaasApp{}
	if existing != nil {
		*app = *existing
	}
	serverID, err := ParseID(req.ServerID)
	if err != nil || serverID == 0 {
		return nil, errcode.New(errcode.InvalidParams, "请选择所属服务器")
	}
	if _, err := s.loadServer(serverID); err != nil {
		return nil, err
	}
	app.ServerID = serverID

	name := strings.TrimSpace(req.Name)
	if !paasAppNamePattern.MatchString(name) {
		return nil, errcode.New(errcode.InvalidParams, "应用名必须以小写字母开头，仅含小写字母、数字、中划线，最长 63 位")
	}
	app.Name = name

	image := strings.TrimSpace(req.Image)
	repo := strings.TrimSpace(req.SourceRepo)
	branch := strings.TrimSpace(req.SourceBranch)
	switch {
	case repo != "":
		// GitHub 构建：在目标服务器上 clone + docker build
		if !paasRepoPattern.MatchString(repo) {
			return nil, errcode.New(errcode.InvalidParams, "仓库格式应为 owner/repo，如 docker/getting-started")
		}
		if branch == "" {
			branch = "main"
		}
		if !paasBranchPattern.MatchString(branch) {
			return nil, errcode.New(errcode.InvalidParams, "分支名含有不支持的字符")
		}
		app.SourceRepo = repo
		app.SourceBranch = branch
		app.SourceAccountID = req.SourceAccountID
		app.Image = ""
	case image != "":
		if !paasImagePattern.MatchString(image) {
			return nil, errcode.New(errcode.InvalidParams, "镜像地址格式不正确")
		}
		app.Image = image
		app.SourceRepo = ""
		app.SourceBranch = ""
		app.SourceAccountID = 0
	default:
		return nil, errcode.New(errcode.InvalidParams, "镜像地址与 GitHub 仓库至少填写一项")
	}

	if req.ContainerPort < 1 || req.ContainerPort > 65535 || req.HostPort < 1 || req.HostPort > 65535 {
		return nil, errcode.New(errcode.InvalidParams, "容器端口与宿主机端口必须在 1-65535 之间")
	}
	app.ContainerPort = req.ContainerPort
	app.HostPort = req.HostPort

	env, err := encodeJSONStrings(req.Env, paasEnvPattern, "环境变量格式应为 KEY=VALUE")
	if err != nil {
		return nil, err
	}
	app.Env = env
	volumes, err := encodeJSONStrings(req.Volumes, paasVolumePattern, "挂载格式应为 /宿主机路径:/容器路径[:ro|:rw]")
	if err != nil {
		return nil, err
	}
	app.Volumes = volumes

	restart := strings.TrimSpace(req.Restart)
	if restart == "" {
		restart = "always"
	}
	if !paasRestartPolicies[restart] {
		return nil, errcode.New(errcode.InvalidParams, "不支持的重启策略")
	}
	app.Restart = restart
	return app, nil
}

// encodeJSONStrings 校验每一项后序列化为 JSON 字符串存储。
func encodeJSONStrings(items []string, pattern *regexp.Regexp, errMsg string) (string, error) {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if !pattern.MatchString(item) {
			return "", errcode.Newf(errcode.InvalidParams, "%s，非法项: %s", errMsg, item)
		}
		out = append(out, item)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// decodeJSONStrings 从 JSON 还原字符串数组，损坏数据按空处理。
func decodeJSONStrings(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []string{}
	}
	return out
}

// serverNameMap 全量服务器 id → 名称映射（自用规模下全量查询足够）。
func (s *PaasService) serverNameMap() map[uint64]string {
	servers, _, err := s.servers.List(repository.PaasServerFilter{}, pagination.Query{Limit: pagination.MaxLimit})
	if err != nil {
		return map[uint64]string{}
	}
	m := make(map[uint64]string, len(servers))
	for i := range servers {
		m[servers[i].ID] = servers[i].Name
	}
	return m
}

// ToPaasAppDTO 模型转响应。
func ToPaasAppDTO(a *model.PaasApp) PaasAppDTO {
	return PaasAppDTO{
		ID:              strconv.FormatUint(a.ID, 10),
		ServerID:        strconv.FormatUint(a.ServerID, 10),
		Name:            a.Name,
		Image:           a.Image,
		SourceRepo:      a.SourceRepo,
		SourceBranch:    a.SourceBranch,
		SourceAccountID: a.SourceAccountID,
		ContainerPort:   a.ContainerPort,
		HostPort:        a.HostPort,
		Env:             decodeJSONStrings(a.Env),
		Volumes:         decodeJSONStrings(a.Volumes),
		Restart:         a.Restart,
		ContainerID:     a.ContainerID,
		Status:          a.Status,
		LastError:       a.LastError,
		DeployLog:       a.DeployLog,
		CreatedAt:       formatTime(a.CreatedAt),
	}
}
