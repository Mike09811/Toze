# AGENTS.md

给在本仓库（Toze）工作的 AI / 新同学的最小必要上下文。读完这一页再动手，能避开 90% 的坑。

## 1. 这是什么

Toze = 自带服务器的极简 PaaS：Vue 3 中后台（`web/`，源自 Admin Plus 完整演示版）+ Go/Gin/MySQL 后端（`server/`）+ SSH 纳管数据面（`internal/pkg/sshx`、`internal/service/paas*.go`）。
接口契约是从前端原有 Mock 1:1 反推的，所以**前端业务代码几乎是原版**，改动集中在集成点。

跑起来最快的方式：`docker compose up -d --build` → 前端 :8080，后端 :8088。
本地开发：`make run`（:8088）+ `make web-dev`（:7802，`/api` 代理到 8088）。

## 2. 不要踩的坑（按踩坑概率排序）

1. **前端必须带 `NODE_OPTIONS=--openssl-legacy-provider`**。Node 17+ 默认禁用 md4，vue-cli 4 / webpack 4 会直接崩。Node 20（CI）也一样。
2. **不要升级 `vue-router`**。锁在 `4.0.14`：4.5+ 的 dist 用了 `?.` 可选链，webpack 4 解析不了，报 `Module parse failed`。
3. **不要删 `codemirror@5.65.16` 这个显式依赖**。simplemde 依赖 `codemirror: "*"`，npm 会装成 v6，导致 `addon/display/*.js` 全找不到。
4. **不要打开 Mock**。`web/src/setting.env.js` 的 `isMock` 必须保持 `false`；`src/mock/` 只是历史残留，不会被引用。
5. **`go test` 不需要 MySQL**。测试用纯 Go 的 SQLite 内存库（`glebarez/sqlite`），每个用例独立 DSN。
6. **种子数据只在表为空时写入**，重启不会重复插；演示数据里有两个同名 `yao` 账号，第二個会自动加 `-011` 后缀，别当成 bug。
7. **前端一律用现成组件，禁止自己造**（原条目正文见下）。UI 元素直接用 `view-ui-plus` 的组件（`Table`、`Form`、`Modal`、`Card`、`Page` 等），组件 API 查 `web/node_modules/view-ui-plus/types` 或官方文档；做页面先去 `web/src/pages/` 找同类现成页面照抄结构（列表、表单、搜索、树形、system 管理页全有模板），只改数据源和字段；布局/框架复用 `web/src/components/frame`，别另起炉灶。手写组件 = 风格漂移 + webpack 4 生态踩坑，一律不做。

## 3. 接口约定（改动前必读）

- 统一响应：`{"code":0,"msg":"成功","data":{}}`，前端拦截器在 `code===0` 时**只取 `data`**。
- **业务失败返回 HTTP 200 + 非 0 code**；只有未登录/Token 失效用 401、无权限用 403。这是前端 `plugins/request` 的既有逻辑，不要"修正"成 REST 风格。
- 所有列表接口都是 **POST**，参数 `{offset, limit, sortKey, sortType}`，`sortType ∈ normal|asc|desc`，`sortKey` 走白名单。
- 字段名必须和前端一致（大小写、下划线都要对）：`hideSider`、`menu_list`、`default_role`、`created_at`、`date`、`region`、`role`。
- `POST /api/system/menu/list` 同时返回 `list`（树）、`header`、`sider` 三种视图，因为前端两处消费方式不同，别只留一个。
- 认证：`Authorization: Bearer <token>`；前端把 token 存在 cookie `token` 里，由 `web/src/setting.js` 的 `requestConfig` 注入。

## 4. 加一个接口的标准流程

```
model → repository → service(+DTO) → handler → router 注册 → 测试 → docs/API.md
```

1. `internal/model/`：新表或加字段，然后确认 `database/migrate.go` 的 AutoMigrate 列表。
2. `internal/repository/`：只做数据访问，返回 `gorm` 错误；查不到统一 `repository.ErrNotFound`。
3. `internal/service/`：业务校验 + 错误码，返回 `errcode.Error`；响应 DTO 放 `dto.go`。
4. `internal/handler/`：只做绑定、调用、`response.OK/OKPage/Fail/Abort`，不写业务。
5. `internal/router/router.go`：读接口挂 `secured`，写接口挂 `admin`（`RequireRoles("ROLE_ADMIN")`）。
6. 测试：service 用例放 `internal/service/*_test.go`，HTTP 用例放 `tests/api_test.go`。
7. 同步 `docs/API.md`，前端若要用，改 `web/src/api/*.js` 只能加函数、不要改已有契约。

## 5. 完成标准（Definition of Done）

```bash
cd server
gofmt -l ./cmd ./internal ./tests   # 必须无输出
go vet ./...                        # 必须无告警
go test ./... -count=1              # 必须全绿（当前 53 个用例）
```

前端改动则额外跑：

```bash
cd web && NODE_OPTIONS=--openssl-legacy-provider npm run build
```

涉及页面/交互的改动，起 `make web-dev` + `make run` 点一遍：登录 → 主控台 → 用户/角色/菜单三个管理页，浏览器控制台不能有红色报错。

## 6. 代码风格

- Go：标准库风格，错误用 `fmt.Errorf("...: %w", err)` 包装；日志用 `slog`，禁止 `fmt.Println`。
- 前端：沿用模板的 4 空格缩进、单引号、分号；不要引入新的状态管理库或 UI 库；组件一律用现成的（见第 2 节第 7 条），不要新写 UI 组件。
- 提交信息：`feat|fix|docs|refactor|test(scope): 一句话`，中文描述可以。
- 不要提交 `.env`、`dist/`、`node_modules/`、`server/bin/`（已在 `.gitignore`）。

## 7. 目录速查

| 路径 | 说明 |
| --- | --- |
| `server/cmd/server/main.go` | 启动顺序：配置 → 日志 → DB → 迁移 → 种子 → 路由 → 优雅停机 |
| `server/internal/pkg/response` | 统一响应，改响应格式只改这里 |
| `server/internal/pkg/errcode` | 错误码表，新增错误先加常量 |
| `server/internal/middleware` | RequestID / 日志 / Recovery / CORS / JWT / RBAC / 限流 / 审计 |
| `server/internal/testutil` | 测试装配：内存库 + 最小种子 + 完整路由 |
| `web/src/setting.js` | 集成点：token 注入、布局配置 |
| `web/src/setting.env.js` | 集成点：`isMock`、publicPath |
| `web/vue.config.js` | 集成点：`/api` 代理、webpack 配置 |

## 8. PaaS / 部署相关新增踩坑

1. **`npm ci` 必须带 `--legacy-peer-deps`**（eslint peer 冲突），`web/Dockerfile` 已带，本地安装同样。
2. **本地构建前端需要 `web/.env`**（至少 `VUE_APP_TITLE=Toze`），已被 gitignore，从 `web/.env.example` 复制。
3. **远程初始化脚本必须兼容非 root SSH 用户**：脚本内统一 `SUDO` 前缀（`sudo -n`）+ `DEBIAN_FRONTEND=noninteractive`，Docker 装完把登录用户加进 docker 组。
4. **国内 VPS 装不上 `dl.cloudflare.com` 的 Caddy apt 源**：`paasInitScript` 已做三级回退（apt → GitHub Release deb → gh-proxy 镜像），别删。
5. **SSH 远程脚本一律走 stdin（`bash -s`）+ `sshx.Quote` 单引号包裹动态值**，新增脚本必须保持该约定。
6. `PAAS_AES_KEY` 一旦产生数据就不可更换（已存凭证无法解密）。
