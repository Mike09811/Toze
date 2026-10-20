# API 文档

Base URL：`http://127.0.0.1:8088`（前端开发服务器通过 `/api` 代理到此处）。

## 约定

- 请求体统一 `application/json`（列表也用 POST，与前端模板保持一致）。
- 除登录 / 注册 / 健康检查外，均需请求头：

```
Authorization: Bearer <token>
```

- 统一响应：

```json
{ "code": 0, "msg": "成功", "data": {} }
```

| 场景 | HTTP | code |
| --- | --- | --- |
| 成功 | 200 | 0 |
| 参数错误 | 200 | 1001 |
| 未登录 / Token 失效 | 401 | 1002 |
| 无权限 | 403 | 1003 |
| 资源不存在 | 200 | 1004 |
| 账号或密码错误 | 200 | 2001 |
| 账号禁用 | 200 | 2002 |
| 账号已存在 | 200 | 2003 |
| 原密码错误 | 200 | 2004 |
| 角色标识已存在 | 200 | 3001 |
| 角色被占用 | 200 | 3002 |
| 存在子菜单 | 200 | 4002 |
| 菜单被角色引用 | 200 | 4003 |
| 限流 | 429 | 1007 |

## 认证

### POST /api/login

```json
{ "username": "admin", "password": "admin" }
```

```json
{
  "code": 0,
  "msg": "登录成功",
  "data": {
    "username": "admin",
    "uuid": "admin-uuid",
    "info": { "uuid": "admin-uuid", "name": "Aresn", "avatar": "...", "access": ["ROLE_ADMIN"] },
    "token": "eyJhbGciOi...",
    "expires_in": 86400
  }
}
```

限流：每 IP 每分钟 10 次。

### POST /api/register

```json
{ "username": "newuser", "password": "secret123", "name": "新用户" }
```

返回结构同登录（注册后直接登录）。

### GET /api/user/info

返回当前登录用户资料（`uuid/name/avatar/access/account/phone/mail/gender/desc`）。

### POST /api/logout

将当前 Token 的 `jti` 写入黑名单，之后该 Token 立即失效。

### POST /api/user/password

```json
{ "oldPassword": "admin", "newPassword": "newpass1" }
```

## 用户

### POST /api/system/user/list

```json
{
  "name": "", "phone": "", "mail": "", "account": "", "id": "",
  "gender": 0, "status": 0, "date": ["2020-01-01", "2020-12-31"],
  "offset": 0, "limit": 10, "sortKey": "date", "sortType": "desc"
}
```

- `gender`：`0` 全部、`1` 男、`2` 女；`status`：`0` 全部、`1` 正常、`2` 冻结、`3` 关闭
- `sortKey` 白名单：`id`、`name`、`account`、`date`、`status`；`sortType`：`normal/asc/desc`

```json
{ "code": 0, "msg": "成功", "data": { "list": [ { "id": "100000001", "name": "夏洛特", "role": ["ROLE_USER"], "region": ["北京市","市辖区","东城区"] } ], "total": 14 } }
```

### POST /api/system/user/create

```json
{
  "name": "张三", "account": "zhangsan", "password": "123456",
  "phone": "13800000000", "mail": "z@a.com", "avatar": "",
  "gender": 1, "status": 1, "birthday": "1998-01-01",
  "region": ["北京市","市辖区","东城区"], "role": ["ROLE_USER"], "desc": ""
}
```

`password` 省略时默认 `123456`；`role` 省略时默认 `ROLE_USER`。

### POST /api/system/user/update

字段同上，必须带 `id`；`password` 为空表示不修改。

### POST /api/system/user/delete

```json
{ "id": "100000015" }
```

限制：不能删除自己、不能删除内置 `admin`。

### POST /api/system/user/delete/multiple

```json
{ "ids": ["100000015", "100000016"] }
```

> 以上写操作需要 `ROLE_ADMIN`，否则 403。

## 角色

### POST /api/system/role/list

```json
{ "offset": 0, "limit": 10, "sortKey": "id", "sortType": "normal" }
```

```json
{ "code": 0, "msg": "成功", "data": { "list": [ { "id": "2000000001", "name": "ROLE_USER", "desc": "普通用户", "default_role": 1, "menu_list": ["1000001","1000003"] } ], "total": 3 } }
```

### POST /api/system/role/all

返回全部角色（不分页），用于下拉选择。

### POST /api/system/role/create / update

```json
{ "name": "ROLE_TEST", "desc": "测试角色", "default_role": 0, "menu_list": ["1000001","1000003"] }
```

### POST /api/system/role/menu/update

```json
{ "id": "2000000001", "menuIds": ["1000001","1000003","1000015"] }
```

### POST /api/system/role/delete 、 /delete/multiple

内置 `ROLE_ADMIN` 不可删除；被用户占用的角色返回 `3002`。

## 菜单

### POST /api/system/menu/list

```json
{ "code": 0, "msg": "成功", "data": {
  "list": [ { "id": "1000001", "title": "首页", "path": "/", "type": "header", "children": [ ... ] } ],
  "header": [ ... ],
  "sider": [ ... ]
} }
```

`list` 为树形结构，`header/sider` 为扁平视图，兼容前端两种消费方式。

### POST /api/system/menu/header/create

```json
{ "title": "新顶栏", "path": "/new", "name": "new", "icon": "md-home", "auth": "admin", "target": "_blank", "divided": false }
```

### POST /api/system/menu/sider/create

同上，额外支持 `parentId`（父级菜单 ID）与 `hideSider`。

### POST /api/system/menu/update

```json
{ "id": "1000057", "title": "改名", "path": "/new2", "hideSider": true }
```

### POST /api/system/menu/delete 、 /delete/multiple

存在子菜单返回 `4002`；被角色引用返回 `4003`。

## 多语言

### POST /api/system/i18n/list

```json
{ "code": 0, "data": { "zh-CN": { "menu": { "dashboard": "主控台" } }, "en-US": { "menu": { "dashboard": "Dashboard" } } } }
```

## 日志

### POST /api/system/log/list

```json
{ "username": "", "path": "", "offset": 0, "limit": 20 }
```

写操作由中间件自动记录（操作人、动作、方法、路径、IP、UA、状态码、耗时）。

## PaaS 自托管部署

> 单用户自用模块：纳管自有 VPS（SSH 直连，不装常驻 Agent），下发 Docker 命令部署容器，
> 通过 Caddy 自动生成反向代理与 HTTPS 证书。凭证 AES-GCM 加密落库，接口永不回传。

### 服务器管理

| 路径 | 说明 |
| --- | --- |
| POST `/api/paas/server/list` | 分页查询，参数 `{name, ip, status, offset, limit, sortKey, sortType}` |
| POST `/api/paas/server/create` | 新增服务器 `{name, ip, ssh_port, ssh_user, auth_type(1密码/2私钥), auth_content}` |
| POST `/api/paas/server/update` | 编辑；`auth_content` 留空表示保持原凭证 |
| POST `/api/paas/server/delete` | 删除；服务器下仍有应用时返回 `1005` |
| POST `/api/paas/server/test` | 连通性测试，成功标记在线并回填 Docker 版本 |
| POST `/api/paas/server/init` | 后台初始化 Docker + Caddy（Debian/Ubuntu），状态置 2，前端轮询列表 |

### 应用管理

| 路径 | 说明 |
| --- | --- |
| POST `/api/paas/app/list` | 分页查询，参数 `{name, server_id, status, offset, limit, ...}` |
| POST `/api/paas/app/create` | `{server_id, name(小写字母数字中划线), image, container_port, host_port, env:["K=V"], volumes:["/h:/c:ro"], restart}` |
| POST `/api/paas/app/update` | 编辑配置，下次部署生效 |
| POST `/api/paas/app/delete` | 级联删除域名绑定、清理远程容器并同步 Caddy |
| POST `/api/paas/app/deploy` | 后台部署 / 重新部署（`docker rm -f` 旧容器后 `docker run`），状态置 3 |
| POST `/api/paas/app/start` / `stop` / `restart` | 容器启停 |
| POST `/api/paas/app/logs` | `{id, tail}`，返回 `docker logs --tail N` 输出 |
| POST `/api/paas/app/status` | 与远程 `docker inspect` 对账刷新状态 |

应用状态：0 未部署、1 运行中、2 已停止、3 部署中。部署失败原因写入 `last_error`。

### 域名管理

| 路径 | 说明 |
| --- | --- |
| POST `/api/paas/domain/list` | 分页查询，参数 `{domain, app_id, offset, limit, ...}` |
| POST `/api/paas/domain/create` | 绑定 `{app_id, domain}`；同步 Caddy 失败自动回滚 |
| POST `/api/paas/domain/delete` | 解绑并同步 Caddy |

每次域名变更会重新生成目标服务器完整 Caddyfile（base64 传输 → `caddy validate` → `systemctl reload caddy`），
证书由 Caddy 自动申请与续期。域名全局唯一，重复绑定返回 `5004`。

### PaaS 错误码

| code | 说明 |
| --- | --- |
| 5001 | 远程服务器执行失败（SSH / Docker 命令） |
| 5002 | 应用名已存在 |
| 5003 | 宿主机端口冲突（同服务器） |
| 5004 | 域名已被绑定 |
| 5005 | 部署 / 初始化正在进行中 |

## 运维

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/healthz` | 存活探针，返回 `{"status":"ok"}` |
| GET | `/readyz` | 就绪探针，含数据库连通性与连接池状态 |
