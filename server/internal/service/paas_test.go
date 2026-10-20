package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/database"
	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/secretbox"
	"github.com/Mike09811/Toze/server/internal/repository"
)

// 本文件为内部测试（package service），需要访问 buildDeployScript 等私有方法，
// 因此不走 testutil（其反向依赖 service，会造成导入环），直接用 database.OpenTest。
var paasTestSeq int64

func newPaasDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := fmt.Sprintf("paas_%d_%d", time.Now().UnixNano(), atomic.AddInt64(&paasTestSeq, 1))
	db, err := database.OpenTest(name)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db
}

// fakeExec 记录脚本并按规则返回结果，替代真实 SSH。
type fakeExec struct {
	mu      sync.Mutex
	scripts []string
	fail    bool
}

func (f *fakeExec) exec(_ context.Context, _ *model.PaasServer, _, script string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scripts = append(f.scripts, script)
	if f.fail {
		return "", errors.New("ssh unavailable")
	}
	if strings.Contains(script, "docker inspect") {
		return "running\n", nil
	}
	if strings.Contains(script, "docker version --format") {
		return "27.1.1\n", nil
	}
	if strings.Contains(script, "docker run") {
		return "abc123def4567890abc123def4567890abc123def4567890abc123def4567890\n", nil
	}
	return "", nil
}

func (f *fakeExec) lastScript() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.scripts) == 0 {
		return ""
	}
	return f.scripts[len(f.scripts)-1]
}

func newPaasTestService(t *testing.T) (*PaasService, *fakeExec, *gorm.DB) {
	t.Helper()
	db := newPaasDB(t)
	fake := &fakeExec{}
	svc := NewPaasService(
		repository.NewPaasServerRepository(db),
		repository.NewPaasAppRepository(db),
		repository.NewPaasDomainRepository(db),
		repository.NewPaasSettingRepository(db),
		"test-aes-key", "", "",
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	svc.SetExec(fake.exec)
	return svc, fake, db
}

func seedServer(t *testing.T, svc *PaasService) *model.PaasServer {
	t.Helper()
	dto, err := svc.CreateServer(PaasServerUpsert{
		Name: "vps-1", IP: "203.0.113.10", SSHPort: 22, SSHUser: "root",
		AuthType: model.PaasAuthPassword, AuthContent: "p@ss 'quoted'",
	})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	srv, err := svc.servers.FindByID(mustUint(dto.ID))
	if err != nil {
		t.Fatalf("reload server: %v", err)
	}
	return srv
}

func seedApp(t *testing.T, svc *PaasService, serverID uint64, name string, hostPort int) *model.PaasApp {
	t.Helper()
	dto, err := svc.CreateApp(PaasAppUpsert{
		ServerID: u64str(serverID), Name: name, Image: "nginx:1.27",
		ContainerPort: 80, HostPort: hostPort,
		Env:     []string{"TOKEN=abc123", "MSG=it's fine"},
		Volumes: []string{"/srv/data:/data", "/srv/conf:/etc/nginx:ro"},
	})
	if err != nil {
		t.Fatalf("create app %s: %v", name, err)
	}
	app, err := svc.apps.FindByID(mustUint(dto.ID))
	if err != nil {
		t.Fatalf("reload app: %v", err)
	}
	return app
}

func TestPaasServerCredentialEncryptedAtRest(t *testing.T) {
	svc, _, db := newPaasTestService(t)
	srv := seedServer(t, svc)

	var raw model.PaasServer
	if err := db.First(&raw, srv.ID).Error; err != nil {
		t.Fatalf("load raw: %v", err)
	}
	if raw.AuthContent == "p@ss 'quoted'" {
		t.Fatalf("凭证不能明文落库")
	}
	plain, err := secretbox.Decrypt("test-aes-key", raw.AuthContent)
	if err != nil || plain != "p@ss 'quoted'" {
		t.Fatalf("凭证应可解密还原: %v %q", err, plain)
	}

	// 更新时凭证留空 = 保持原凭证
	if _, err := svc.UpdateServer(PaasServerUpsert{
		ID: u64str(srv.ID), Name: "vps-1b", IP: "203.0.113.10",
		SSHPort: 22, SSHUser: "root", AuthType: model.PaasAuthPassword,
	}); err != nil {
		t.Fatalf("update server: %v", err)
	}
	kept, _ := svc.servers.FindByID(srv.ID)
	if kept.AuthContent != raw.AuthContent {
		t.Fatalf("留空更新不应改动凭证")
	}
	if kept.Name != "vps-1b" {
		t.Fatalf("名称应更新")
	}
}

func TestPaasServerDeleteBlockedByApps(t *testing.T) {
	svc, _, _ := newPaasTestService(t)
	srv := seedServer(t, svc)
	seedApp(t, svc, srv.ID, "web", 8080)

	if err := svc.DeleteServer(srv.ID); err == nil {
		t.Fatalf("有应用时删除服务器应被拒绝")
	}
}

func TestPaasServerTestConnectivity(t *testing.T) {
	svc, _, _ := newPaasTestService(t)
	srv := seedServer(t, svc)
	dto, err := svc.TestServer(context.Background(), srv.ID)
	if err != nil {
		t.Fatalf("test: %v", err)
	}
	if dto.Status != model.PaasServerOnline || dto.DockerVer != "27.1.1" {
		t.Fatalf("应标记在线并回填版本: %+v", dto)
	}
}

func TestPaasAppValidation(t *testing.T) {
	svc, _, _ := newPaasTestService(t)
	srv := seedServer(t, svc)

	cases := []PaasAppUpsert{
		{ServerID: u64str(srv.ID), Name: "Bad_Name", Image: "nginx", ContainerPort: 80, HostPort: 8080},
		{ServerID: u64str(srv.ID), Name: "1abc", Image: "nginx", ContainerPort: 80, HostPort: 8080},
		{ServerID: u64str(srv.ID), Name: "ok", Image: "nginx; rm -rf /", ContainerPort: 80, HostPort: 8080},
		{ServerID: u64str(srv.ID), Name: "ok", Image: "nginx", ContainerPort: 0, HostPort: 8080},
		{ServerID: u64str(srv.ID), Name: "ok", Image: "nginx", ContainerPort: 80, HostPort: 70000},
		{ServerID: u64str(srv.ID), Name: "ok", Image: "nginx", ContainerPort: 80, HostPort: 8080, Env: []string{"IN VALID"}},
		{ServerID: u64str(srv.ID), Name: "ok", Image: "nginx", ContainerPort: 80, HostPort: 8080, Volumes: []string{"/a /b"}},
		{ServerID: u64str(srv.ID), Name: "ok", Image: "nginx", ContainerPort: 80, HostPort: 8080, Restart: "always;reboot"},
	}
	for i, req := range cases {
		if _, err := svc.CreateApp(req); err == nil {
			t.Fatalf("第 %d 组非法参数应被拒绝: %+v", i, req)
		}
	}
}

func TestPaasAppPortConflict(t *testing.T) {
	svc, _, _ := newPaasTestService(t)
	srv := seedServer(t, svc)
	seedApp(t, svc, srv.ID, "web", 8080)
	_, err := svc.CreateApp(PaasAppUpsert{
		ServerID: u64str(srv.ID), Name: "api", Image: "nginx", ContainerPort: 80, HostPort: 8080,
	})
	if err == nil || !strings.Contains(err.Error(), "端口") {
		t.Fatalf("同服务器端口冲突应被拒绝: %v", err)
	}
	// 不同服务器同端口允许
	srv2, err := svc.CreateServer(PaasServerUpsert{
		Name: "vps-2", IP: "203.0.113.11", AuthType: model.PaasAuthPassword, AuthContent: "x",
	})
	if err != nil {
		t.Fatalf("create server 2: %v", err)
	}
	if _, err := svc.CreateApp(PaasAppUpsert{
		ServerID: srv2.ID, Name: "api", Image: "nginx", ContainerPort: 80, HostPort: 8080,
	}); err != nil {
		t.Fatalf("不同服务器同端口应允许: %v", err)
	}
}

func TestPaasDeployScriptQuotesEverything(t *testing.T) {
	svc, _, _ := newPaasTestService(t)
	srv := seedServer(t, svc)
	app := seedApp(t, svc, srv.ID, "web", 8080)
	app.Image = "my repo/app:v1'evil" // 构造极端值直接验证脚本拼接
	script := svc.buildDeployScript(app, "")

	for _, want := range []string{
		"--name 'web'",
		"--restart 'always'",
		"-p '8080:80'",
		"-e 'TOKEN=abc123'",
		`-e 'MSG=it'\''s fine'`,
		"-v '/srv/data:/data'",
		"-v '/srv/conf:/etc/nginx:ro'",
		`'my repo/app:v1'\''evil'`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("部署脚本缺少 %q:\n%s", want, script)
		}
	}
	if strings.Contains(script, "rm -rf") {
		t.Fatalf("脚本不应包含未转义的危险命令:\n%s", script)
	}
}

func TestPaasDomainBindAndCaddySync(t *testing.T) {
	svc, fake, _ := newPaasTestService(t)
	srv := seedServer(t, svc)
	app := seedApp(t, svc, srv.ID, "web", 8080)

	if _, err := svc.CreateDomain(PaasDomainUpsert{AppID: u64str(app.ID), Domain: "Web.Example.COM"}); err != nil {
		t.Fatalf("bind: %v", err)
	}
	script := fake.lastScript()
	if !strings.Contains(script, "caddy validate") || !strings.Contains(script, "systemctl reload caddy") {
		t.Fatalf("同步脚本应先校验再重载:\n%s", script)
	}
	// 域名应小写化并出现在解码后的 Caddyfile 中
	var encoded string
	for _, l := range strings.Split(script, "\n") {
		if !strings.HasPrefix(l, "echo '") {
			continue
		}
		rest := l[len("echo '"):]
		if idx := strings.Index(rest, "'"); idx >= 0 {
			encoded = rest[:idx]
		}
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("脚本内应是 base64 配置: %v", err)
	}
	if !strings.Contains(string(decoded), "web.example.com {\n    reverse_proxy 127.0.0.1:8080") {
		t.Fatalf("Caddyfile 内容不符:\n%s", decoded)
	}

	// 重复绑定拒绝
	if _, err := svc.CreateDomain(PaasDomainUpsert{AppID: u64str(app.ID), Domain: "web.example.com"}); err == nil {
		t.Fatalf("重复域名应被拒绝")
	}
	// 非法域名拒绝
	for _, bad := range []string{"no-dot", "-bad.example.com", "a..b", "web.example.com:80", "*.example.com"} {
		if _, err := svc.CreateDomain(PaasDomainUpsert{AppID: u64str(app.ID), Domain: bad}); err == nil {
			t.Fatalf("非法域名 %q 应被拒绝", bad)
		}
	}
}

func TestPaasDomainSyncFailureRollsBack(t *testing.T) {
	svc, fake, _ := newPaasTestService(t)
	srv := seedServer(t, svc)
	app := seedApp(t, svc, srv.ID, "web", 8080)

	fake.fail = true
	if _, err := svc.CreateDomain(PaasDomainUpsert{AppID: u64str(app.ID), Domain: "fail.example.com"}); err == nil {
		t.Fatalf("同步失败应报错")
	}
	fake.fail = false
	if _, err := svc.domains.FindByDomain("fail.example.com"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("同步失败应回滚域名记录: %v", err)
	}
}

func TestPaasDeleteAppCascadesDomains(t *testing.T) {
	svc, _, _ := newPaasTestService(t)
	srv := seedServer(t, svc)
	app := seedApp(t, svc, srv.ID, "web", 8080)
	if _, err := svc.CreateDomain(PaasDomainUpsert{AppID: u64str(app.ID), Domain: "a.example.com"}); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if _, err := svc.CreateDomain(PaasDomainUpsert{AppID: u64str(app.ID), Domain: "b.example.com"}); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if err := svc.DeleteApp(context.Background(), app.ID); err != nil {
		t.Fatalf("delete app: %v", err)
	}
	domains, err := svc.domains.ListByAppIDs([]uint64{app.ID})
	if err != nil {
		t.Fatalf("list domains: %v", err)
	}
	if len(domains) != 0 {
		t.Fatalf("应用删除后域名应级联删除，剩余 %d 条", len(domains))
	}
}

func TestPaasRefreshAppStatus(t *testing.T) {
	svc, _, _ := newPaasTestService(t)
	srv := seedServer(t, svc)
	app := seedApp(t, svc, srv.ID, "web", 8080)
	if err := svc.apps.UpdateDeployment(app.ID, "abc123", model.PaasAppStopped, ""); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	dto, err := svc.RefreshAppStatus(context.Background(), app.ID)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if dto.Status != model.PaasAppRunning {
		t.Fatalf("状态应与远程对账为运行中: %+v", dto)
	}
}

func u64str(v uint64) string { return strconv.FormatUint(v, 10) }

func mustUint(s string) uint64 {
	id, err := ParseID(s)
	if err != nil {
		panic(err)
	}
	return id
}

func TestPaasGitHubBuildScript(t *testing.T) {
	svc, _, _ := newPaasTestService(t)
	srv := seedServer(t, svc)
	dto, err := svc.CreateApp(PaasAppUpsert{
		ServerID: u64str(srv.ID), Name: "site", SourceRepo: "docker/getting-started", SourceBranch: "main",
		ContainerPort: 3000, HostPort: 18090,
	})
	if err != nil {
		t.Fatalf("create github app: %v", err)
	}
	app, _ := svc.apps.FindByID(mustUint(dto.ID))
	if app.Image != "" || app.SourceRepo != "docker/getting-started" {
		t.Fatalf("GitHub 应用不应存镜像字段: %+v", app)
	}
	script := svc.buildDeployScript(app, "tok'en")
	for _, want := range []string{
		"WORK=\"$HOME\"/toze/build/site",
		"git clone --depth 1 -b 'main' 'https://x-access-token:tok'\\''en@github.com/docker/getting-started.git'",
		"docker build -t 'toze/site:latest' .",
		"git -C \"$WORK\" remote remove origin",
		"--name 'site'",
		"-p '18090:3000'",
		"'toze/site:latest'",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("构建脚本缺少 %q:\n%s", want, script)
		}
	}
	// 二选一校验
	if _, err := svc.CreateApp(PaasAppUpsert{
		ServerID: u64str(srv.ID), Name: "both", Image: "nginx", SourceRepo: "a/b", ContainerPort: 80, HostPort: 1,
	}); err != nil {
		t.Fatalf("同时填写时以仓库为准，不应报错")
	}
	if _, err := svc.CreateApp(PaasAppUpsert{
		ServerID: u64str(srv.ID), Name: "none", ContainerPort: 80, HostPort: 2,
	}); err == nil {
		t.Fatalf("镜像与仓库都为空应被拒绝")
	}
}
