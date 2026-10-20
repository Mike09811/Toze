# Toze

> 自带服务器的极简 PaaS —— 一台 VPS 跑全套：纳管服务器、部署 Docker 容器、绑定域名自动 HTTPS。

Toze 是给个人开发者的自托管部署平台（替代 Zeabur / Coolify 核心能力的极简版）：

- **控制面**：Go（Gin + GORM + MySQL）+ Vue 3 后台，Docker Compose 一键部署
- **数据面**：你的任意 VPS，通过 SSH 下发指令，**不安装常驻 Agent**
- **核心闭环**：添加服务器 → 一键初始化（Docker + Caddy）→ 部署镜像 → 绑定域名 → 自动 HTTPS

```text
浏览器 → Toze 控制面 → SSH → 你的 VPS
                              ├─ Docker 运行业务容器
                              └─ Caddy 域名 + 自动证书
```

## 快速安装（一台全新 Ubuntu/Debian 服务器）

```bash
curl -fsSL https://raw.githubusercontent.com/Mike09811/Toze/main/install.sh | bash
```

或手动：

```bash
git clone https://github.com/Mike09811/Toze.git && cd Toze
bash install.sh
```

脚本会自动：安装 Docker（如缺失）→ 生成随机密钥写入 `.env` → `docker compose up -d --build` → 等待健康检查通过。

完成后访问 `http://服务器IP:8080`，默认账号 **admin / admin**（请立即在「系统管理 → 用户管理」修改密码）。

## 更新升级

部署目录本身就是 git 仓库，更新只需：

```bash
cd ~/Toze && git pull
sudo docker compose up -d --build
```

`.env` 不在 git 里，更新不会覆盖密钥。

## GitHub 构建

应用来源支持两种：**镜像**（直接填现成镜像）与 **GitHub 构建**（在目标服务器上 `git clone + docker build`，仓库根目录需有 `Dockerfile`）。

- **公开仓库**：无需任何配置，直接填 `owner/repo` 即可构建
- **私有仓库**：需要一键授权 GitHub。到 GitHub → Settings → Developer settings → OAuth Apps 创建一个应用：
  - Homepage URL：`http://你的服务器IP:8080`
  - Authorization callback URL：`http://你的服务器IP:8080/api/paas/github/callback`
  - 把生成的 Client ID / Client Secret 填到 `.env`（`GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET`）后 `docker compose up -d` 重启
  - 在「应用管理 → 新建应用 → GitHub 构建」里点「一键授权 GitHub」完成绑定

授权 token 使用 AES-GCM 加密存储，构建时仅在目标服务器上瞬时使用并随即抹除。

## 使用流程

1. **服务器管理** → 添加服务器（IP / SSH 端口 / 账号 / 密码或私钥，凭证 AES-GCM 加密落库、永不回传）
2. 点「初始化」自动安装 Docker + Caddy（Debian/Ubuntu，支持非 root 用户）
3. **应用管理** → 新建应用（镜像、端口映射、环境变量、数据卷）→ 部署
4. **域名管理** → 域名 A 记录解析到服务器 IP 后绑定，Caddy 自动签发 / 续期 HTTPS 证书

## 功能一览

| 模块 | 能力 |
| --- | --- |
| 服务器 | 增删改查、连通性测试、一键初始化 Docker + Caddy、状态与错误回显 |
| 应用 | 部署 / 重新部署、启停 / 重启、日志查看（tail 200）、状态对账、端口冲突校验 |
| 域名 | 绑定 / 解绑，自动生成完整 Caddyfile（base64 传输 → validate → 热重载） |
| 安全 | SSH 凭证 AES-256-GCM 加密、脚本全部单引号转义 + 白名单校验、写操作仅管理员 |
| 平台底座 | 用户 / 角色 / 菜单 RBAC、JWT、操作审计、多语言、健康探针 |

所有远程脚本通过 stdin 交给 `bash -s` 执行，动态值一律单引号包裹；长任务（部署、初始化）后台执行，前端轮询状态。

## 技术栈

**后端**：Go 1.26+ / Gin / GORM / MySQL 8 / golang.org/x/crypto/ssh / slog
**前端**：Vue 3 / view-ui-plus / vue-cli 4（webpack 4）
**测试**：Go 标准测试 + 纯 Go SQLite 内存库（`go test ./...` 无需外部依赖）

## 本地开发

```bash
make setup    # 安装前后端依赖
make run      # 后端 :8088（需 MySQL，见 server/.env.example）
make web-dev  # 前端 :7802，/api 代理到 8088
make test     # 全量测试
```

前端构建需 Node 17+ 且带 `NODE_OPTIONS=--openssl-legacy-provider`（webpack 4 兼容），首次复制 `web/.env.example` 为 `web/.env`。

## 配置项（.env）

| 变量 | 说明 |
| --- | --- |
| `DB_PASSWORD` / `DB_NAME` | MySQL 密码与库名 |
| `JWT_SECRET` | 登录令牌签名密钥（至少 32 位） |
| `PAAS_AES_KEY` | SSH 凭证加密密钥，**一旦设置不可更换**（否则已存凭证无法解密） |

完整变量见 [server/.env.example](server/.env.example)，接口文档见 [docs/API.md](docs/API.md)。

## 目录结构

```text
server/  Go 后端（cmd/server 入口，internal/ 分层：model/repository/service/handler）
web/     Vue 3 前端（Admin Plus 演示版底座 + PaaS 页面）
docs/    接口文档与截图
```

## License

[MIT](LICENSE)
