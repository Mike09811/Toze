package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Mike09811/Toze/server/internal/pkg/response"
	"github.com/Mike09811/Toze/server/internal/service"
)

// PaasHandler PaaS 接口。
type PaasHandler struct{ svc *service.PaasService }

// NewPaasHandler 构造函数。
func NewPaasHandler(svc *service.PaasService) *PaasHandler { return &PaasHandler{svc: svc} }

type paasListReq struct {
	Offset   int    `json:"offset"`
	Limit    int    `json:"limit"`
	SortType string `json:"sortType"`
	SortKey  string `json:"sortKey"`
}

type paasServerListReq struct {
	paasListReq
	Name   string `json:"name"`
	IP     string `json:"ip"`
	Status int    `json:"status"`
}

type paasAppListReq struct {
	paasListReq
	Name     string `json:"name"`
	ServerID string `json:"server_id"`
	Status   int    `json:"status"`
}

type paasDomainListReq struct {
	paasListReq
	Domain string `json:"domain"`
	AppID  string `json:"app_id"`
}

type paasServerUpsertReq struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	IP          string `json:"ip"`
	SSHPort     int    `json:"ssh_port"`
	SSHUser     string `json:"ssh_user"`
	AuthType    int    `json:"auth_type"`
	AuthContent string `json:"auth_content"`
}

type paasAppUpsertReq struct {
	ID              string   `json:"id"`
	ServerID        string   `json:"server_id"`
	Name            string   `json:"name"`
	Image           string   `json:"image"`
	SourceRepo      string   `json:"source_repo"`
	SourceBranch    string   `json:"source_branch"`
	SourceAccountID string   `json:"source_account_id"`
	ContainerPort   int      `json:"container_port"`
	HostPort        int      `json:"host_port"`
	Env             []string `json:"env"`
	Volumes         []string `json:"volumes"`
	Restart         string   `json:"restart"`
}

type paasDomainUpsertReq struct {
	AppID  string `json:"app_id"`
	Domain string `json:"domain"`
}

type paasAppLogsReq struct {
	ID   string `json:"id"`
	Tail int    `json:"tail"`
}

// ServerList POST /api/paas/server/list
func (h *PaasHandler) ServerList(c *gin.Context) {
	var req paasServerListReq
	if !bind(c, &req) {
		return
	}
	list, total, err := h.svc.ListServers(service.PaasServerQuery{
		PaasListQuery: toPaasListQuery(req.paasListReq),
		Name:          req.Name, IP: req.IP, Status: req.Status,
	})
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKPage(c, list, total)
}

// ServerCreate POST /api/paas/server/create
func (h *PaasHandler) ServerCreate(c *gin.Context) {
	var req paasServerUpsertReq
	if !bind(c, &req) {
		return
	}
	dto, err := h.svc.CreateServer(toPaasServerUpsert(req))
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "添加成功", dto)
}

// ServerUpdate POST /api/paas/server/update
func (h *PaasHandler) ServerUpdate(c *gin.Context) {
	var req paasServerUpsertReq
	if !bind(c, &req) {
		return
	}
	dto, err := h.svc.UpdateServer(toPaasServerUpsert(req))
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "保存成功", dto)
}

// ServerDelete POST /api/paas/server/delete
func (h *PaasHandler) ServerDelete(c *gin.Context) {
	var req deleteReq
	if !bind(c, &req) {
		return
	}
	id, err := service.ParseID(req.ID)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	if err := h.svc.DeleteServer(id); err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "删除成功", gin.H{"msg": "删除成功"})
}

// ServerTest POST /api/paas/server/test 连通性测试
func (h *PaasHandler) ServerTest(c *gin.Context) {
	var req deleteReq
	if !bind(c, &req) {
		return
	}
	id, err := service.ParseID(req.ID)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	dto, err := h.svc.TestServer(c.Request.Context(), id)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "连接成功", dto)
}

// ServerInit POST /api/paas/server/init 环境初始化（后台执行）
func (h *PaasHandler) ServerInit(c *gin.Context) {
	var req deleteReq
	if !bind(c, &req) {
		return
	}
	id, err := service.ParseID(req.ID)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	if err := h.svc.InitServer(id); err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "初始化已开始，请稍后刷新查看结果", gin.H{"msg": "初始化已开始"})
}

// AppList POST /api/paas/app/list
func (h *PaasHandler) AppList(c *gin.Context) {
	var req paasAppListReq
	if !bind(c, &req) {
		return
	}
	serverID, _ := service.ParseID(req.ServerID)
	list, total, err := h.svc.ListApps(service.PaasAppQuery{
		PaasListQuery: toPaasListQuery(req.paasListReq),
		Name:          req.Name, ServerID: serverID, Status: req.Status,
	})
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKPage(c, list, total)
}

// AppCreate POST /api/paas/app/create
func (h *PaasHandler) AppCreate(c *gin.Context) {
	var req paasAppUpsertReq
	if !bind(c, &req) {
		return
	}
	dto, err := h.svc.CreateApp(toPaasAppUpsert(req))
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "创建成功", dto)
}

// AppUpdate POST /api/paas/app/update
func (h *PaasHandler) AppUpdate(c *gin.Context) {
	var req paasAppUpsertReq
	if !bind(c, &req) {
		return
	}
	dto, err := h.svc.UpdateApp(toPaasAppUpsert(req))
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "保存成功", dto)
}

// AppDelete POST /api/paas/app/delete
func (h *PaasHandler) AppDelete(c *gin.Context) {
	var req deleteReq
	if !bind(c, &req) {
		return
	}
	id, err := service.ParseID(req.ID)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	if err := h.svc.DeleteApp(c.Request.Context(), id); err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "删除成功", gin.H{"msg": "删除成功"})
}

// AppDeploy POST /api/paas/app/deploy（后台执行）
func (h *PaasHandler) AppDeploy(c *gin.Context) {
	var req deleteReq
	if !bind(c, &req) {
		return
	}
	id, err := service.ParseID(req.ID)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	if err := h.svc.DeployApp(id); err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "部署已开始，请稍后刷新查看状态", gin.H{"msg": "部署已开始"})
}

// AppStart POST /api/paas/app/start
func (h *PaasHandler) AppStart(c *gin.Context) {
	h.appContainerAction(c, h.svc.StartApp, "启动成功")
}

// AppStop POST /api/paas/app/stop
func (h *PaasHandler) AppStop(c *gin.Context) {
	h.appContainerAction(c, h.svc.StopApp, "已停止")
}

// AppRestart POST /api/paas/app/restart
func (h *PaasHandler) AppRestart(c *gin.Context) {
	h.appContainerAction(c, h.svc.RestartApp, "已重启")
}

func (h *PaasHandler) appContainerAction(c *gin.Context, fn func(ctx context.Context, id uint64) error, msg string) {
	var req deleteReq
	if !bind(c, &req) {
		return
	}
	id, err := service.ParseID(req.ID)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	if err := fn(c.Request.Context(), id); err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, msg, gin.H{"msg": msg})
}

// AppDeployLog POST /api/paas/app/deploy-log 构建日志（部署中实时轮询）
func (h *PaasHandler) AppDeployLog(c *gin.Context) {
	var req deleteReq
	if !bind(c, &req) {
		return
	}
	id, err := service.ParseID(req.ID)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	dto, err := h.svc.AppDeployLog(id)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OK(c, gin.H{"status": dto.Status, "log": dto.DeployLog})
}

// AppLogs POST /api/paas/app/logs
func (h *PaasHandler) AppLogs(c *gin.Context) {
	var req paasAppLogsReq
	if !bind(c, &req) {
		return
	}
	id, err := service.ParseID(req.ID)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	logs, err := h.svc.AppLogs(c.Request.Context(), id, req.Tail)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OK(c, gin.H{"logs": logs})
}

// AppStatus POST /api/paas/app/status 状态刷新
func (h *PaasHandler) AppStatus(c *gin.Context) {
	var req deleteReq
	if !bind(c, &req) {
		return
	}
	id, err := service.ParseID(req.ID)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	dto, err := h.svc.RefreshAppStatus(c.Request.Context(), id)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OK(c, dto)
}

// DomainList POST /api/paas/domain/list
func (h *PaasHandler) DomainList(c *gin.Context) {
	var req paasDomainListReq
	if !bind(c, &req) {
		return
	}
	appID, _ := service.ParseID(req.AppID)
	list, total, err := h.svc.ListDomains(service.PaasDomainQuery{
		PaasListQuery: toPaasListQuery(req.paasListReq),
		Domain:        req.Domain, AppID: appID,
	})
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKPage(c, list, total)
}

// DomainCreate POST /api/paas/domain/create
func (h *PaasHandler) DomainCreate(c *gin.Context) {
	var req paasDomainUpsertReq
	if !bind(c, &req) {
		return
	}
	dto, err := h.svc.CreateDomain(toPaasDomainUpsert(req))
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "绑定成功", dto)
}

// DomainDelete POST /api/paas/domain/delete
func (h *PaasHandler) DomainDelete(c *gin.Context) {
	var req deleteReq
	if !bind(c, &req) {
		return
	}
	id, err := service.ParseID(req.ID)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	if err := h.svc.DeleteDomain(c.Request.Context(), id); err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "解绑成功", gin.H{"msg": "解绑成功"})
}

func toPaasListQuery(req paasListReq) service.PaasListQuery {
	return service.PaasListQuery{
		Offset: req.Offset, Limit: req.Limit, SortType: req.SortType, SortKey: req.SortKey,
	}
}

func toPaasServerUpsert(req paasServerUpsertReq) service.PaasServerUpsert {
	return service.PaasServerUpsert{
		ID: req.ID, Name: req.Name, IP: req.IP, SSHPort: req.SSHPort,
		SSHUser: req.SSHUser, AuthType: req.AuthType, AuthContent: req.AuthContent,
	}
}

func toPaasAppUpsert(req paasAppUpsertReq) service.PaasAppUpsert {
	accountID, _ := service.ParseID(req.SourceAccountID)
	return service.PaasAppUpsert{
		ID: req.ID, ServerID: req.ServerID, Name: req.Name, Image: req.Image,
		SourceRepo: req.SourceRepo, SourceBranch: req.SourceBranch, SourceAccountID: accountID,
		ContainerPort: req.ContainerPort, HostPort: req.HostPort,
		Env: req.Env, Volumes: req.Volumes, Restart: req.Restart,
	}
}

// GitHubStatus POST /api/paas/github/status（账号列表 + 配置状态）
func (h *PaasHandler) GitHubStatus(c *gin.Context) {
	dto, err := h.svc.GitHubStatus()
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OK(c, dto)
}

// GitHubSetDefault POST /api/paas/github/accounts/default
func (h *PaasHandler) GitHubSetDefault(c *gin.Context) {
	var req struct {
		ID string `json:"id"`
	}
	if !bind(c, &req) {
		return
	}
	id, err := service.ParseID(req.ID)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	if err := h.svc.GitHubSetDefault(id); err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "已设为默认账号", gin.H{"msg": "ok"})
}

// GitHubDeleteAccount POST /api/paas/github/accounts/delete
func (h *PaasHandler) GitHubDeleteAccount(c *gin.Context) {
	var req deleteReq
	if !bind(c, &req) {
		return
	}
	id, err := service.ParseID(req.ID)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	if err := h.svc.GitHubDeleteAccount(id); err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "已删除授权账号", gin.H{"msg": "ok"})
}

// GitHubAuthURL POST /api/paas/github/auth/url
func (h *PaasHandler) GitHubAuthURL(c *gin.Context) {
	url, err := h.svc.GitHubAuthURL()
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OK(c, gin.H{"url": url})
}

// GitHubRepos POST /api/paas/github/repos
func (h *PaasHandler) GitHubRepos(c *gin.Context) {
	var req struct {
		AccountID string `json:"account_id"`
	}
	if !bind(c, &req) {
		return
	}
	accountID, _ := service.ParseID(req.AccountID)
	repos, err := h.svc.GitHubRepos(c.Request.Context(), accountID)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OK(c, repos)
}

// GitHubCallback GET /api/paas/github/callback（浏览器 OAuth 回调，公开）
func (h *PaasHandler) GitHubCallback(c *gin.Context) {
	if err := h.svc.GitHubCallback(c.Request.Context(), c.Query("code")); err != nil {
		c.String(http.StatusInternalServerError, "GitHub 授权失败: %v", err)
		return
	}
	c.Redirect(http.StatusFound, "/paas/app?github=ok")
}

func toPaasDomainUpsert(req paasDomainUpsertReq) service.PaasDomainUpsert {
	return service.PaasDomainUpsert{AppID: req.AppID, Domain: req.Domain}
}
