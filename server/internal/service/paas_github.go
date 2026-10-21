package service

import (
	"context"
	"errors"
	"strconv"

	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/pkg/githubx"
	"github.com/Mike09811/Toze/server/internal/pkg/secretbox"
	"github.com/Mike09811/Toze/server/internal/repository"
)

// GitHubAccountDTO 授权账号信息（不含 token）。
type GitHubAccountDTO struct {
	ID        string `json:"id"`
	Login     string `json:"login"`
	IsDefault bool   `json:"is_default"`
	// TokenValid 由状态接口实时探测：token 是否仍被 GitHub 接受。
	TokenValid bool   `json:"token_valid"`
	InvalidMsg string `json:"invalid_msg,omitempty"`
	CreatedAt  string `json:"created_at"`
}

// GitHubStatusDTO 授权总览。
type GitHubStatusDTO struct {
	Accounts        []GitHubAccountDTO `json:"accounts"`
	OAuthConfigured bool               `json:"oauth_configured"`
}

// RepoDTO 仓库信息。
type RepoDTO struct {
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
	Private       bool   `json:"private"`
}

// GitHubAuthURL 生成一键授权跳转地址。
func (s *PaasService) GitHubAuthURL() (string, error) {
	if s.githubClientID == "" || s.githubClientSecret == "" {
		return "", errcode.New(errcode.InvalidParams,
			"GitHub OAuth 未配置：请在 .env 设置 GITHUB_CLIENT_ID / GITHUB_CLIENT_SECRET（公开仓库构建无需授权）")
	}
	return githubx.AuthorizeURL(s.githubClientID, "repo"), nil
}

// GitHubCallback OAuth 回调：code 换 token，按登录名合并入账号表。
func (s *PaasService) GitHubCallback(ctx context.Context, code string) error {
	if s.githubClientID == "" || s.githubClientSecret == "" {
		return errcode.New(errcode.InvalidParams, "GitHub OAuth 未配置")
	}
	if code == "" {
		return errcode.New(errcode.InvalidParams, "缺少授权码")
	}
	token, err := githubx.Exchange(ctx, s.githubClientID, s.githubClientSecret, code)
	if err != nil {
		return errcode.Newf(errcode.Internal, "%v", err)
	}
	login, err := githubx.User(ctx, token)
	if err != nil {
		return errcode.Newf(errcode.Internal, "获取 GitHub 用户信息失败: %v", err)
	}
	encrypted, err := secretbox.Encrypt(s.aesKey, token)
	if err != nil {
		return errcode.Newf(errcode.Internal, "加密 GitHub 令牌失败: %v", err)
	}

	account, err := s.accounts.FindByLogin(login)
	isNew := false
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		account = &model.PaasGitHubAccount{}
		isNew = true
	}
	account.Login = login
	account.Token = encrypted
	if count, _ := s.accounts.Count(); count == 0 {
		account.IsDefault = true
	}
	if err := s.accounts.Upsert(account); err != nil {
		return err
	}
	s.log.Info("GitHub 授权成功", "login", login, "new", isNew)
	return nil
}

// GitHubStatus 授权总览：账号列表（实时校验 token 有效性）+ OAuth 配置状态。
func (s *PaasService) GitHubStatus(ctx context.Context) (*GitHubStatusDTO, error) {
	accounts, err := s.accounts.List()
	if err != nil {
		return nil, err
	}
	out := make([]GitHubAccountDTO, 0, len(accounts))
	for i := range accounts {
		dto := ToGitHubAccountDTO(&accounts[i])
		dto.TokenValid, dto.InvalidMsg = s.probeGitHubToken(ctx, &accounts[i])
		out = append(out, dto)
	}
	return &GitHubStatusDTO{
		Accounts:        out,
		OAuthConfigured: s.githubClientID != "" && s.githubClientSecret != "",
	}, nil
}

// probeGitHubToken 实时探测账号 token 是否仍有效（一次 GET /user）。
func (s *PaasService) probeGitHubToken(ctx context.Context, account *model.PaasGitHubAccount) (bool, string) {
	plain, err := secretbox.Decrypt(s.aesKey, account.Token)
	if err != nil {
		return false, "令牌解密失败（PAAS_AES_KEY 变更过？），请删除账号后重新授权"
	}
	if _, err := githubx.User(ctx, plain); err != nil {
		var se *githubx.StatusError
		if errors.As(err, &se) && se.Status == 401 {
			return false, "授权已失效（token 被 GitHub 吊销或撤销），请重新一键授权"
		}
		if errors.As(err, &se) && se.Status == 403 {
			return false, "访问被 GitHub 拒绝（403），可能是限流，稍后重试"
		}
		return false, "GitHub 访问异常: " + err.Error()
	}
	return true, ""
}

// mapGitHubAPIError 把 GitHub API 错误翻译成用户可执行的提示。
func mapGitHubAPIError(err error) error {
	var se *githubx.StatusError
	if errors.As(err, &se) {
		switch se.Status {
		case 401:
			return errcode.New(errcode.Unauthorized,
				"GitHub 授权已失效（token 被吊销或撤销），请到「系统设置 → GitHub 授权」重新一键授权")
		case 403:
			return errcode.New(errcode.Forbidden,
				"GitHub 拒绝访问（403）：token 权限不足或触发限流，稍后重试")
		case 404:
			return errcode.New(errcode.NotFound,
				"仓库不存在，或当前 GitHub 账号无权访问")
		}
	}
	return err
}

// GitHubRepos 列出指定账号的仓库；accountID 为空时用默认账号。
func (s *PaasService) GitHubRepos(ctx context.Context, accountID uint64) ([]RepoDTO, error) {
	token, _, err := s.resolveGitHubToken(accountID)
	if err != nil {
		return nil, err
	}
	repos, err := githubx.ListRepos(ctx, token)
	if err != nil {
		return nil, mapGitHubAPIError(err)
	}
	out := make([]RepoDTO, 0, len(repos))
	for _, r := range repos {
		out = append(out, RepoDTO{
			FullName:      r.FullName,
			DefaultBranch: r.DefaultBranch,
			Private:       r.Private,
		})
	}
	return out, nil
}

// GitHubSetDefault 设置默认账号。
func (s *PaasService) GitHubSetDefault(id uint64) error {
	if _, err := s.accounts.FindByID(id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errcode.New(errcode.NotFound, "账号不存在")
		}
		return err
	}
	return s.accounts.SetDefault(id)
}

// GitHubDeleteAccount 删除授权账号。
func (s *PaasService) GitHubDeleteAccount(id uint64) error {
	if err := s.accounts.Delete(id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errcode.New(errcode.NotFound, "账号不存在")
		}
		return err
	}
	return nil
}

// resolveGitHubToken 解析可用 token：指定账号优先，其次默认账号。
func (s *PaasService) resolveGitHubToken(accountID uint64) (token, login string, err error) {
	var account *model.PaasGitHubAccount
	if accountID > 0 {
		account, err = s.accounts.FindByID(accountID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return "", "", errcode.New(errcode.NotFound, "GitHub 账号不存在，请到系统设置重新授权")
			}
			return "", "", err
		}
	} else {
		accounts, lerr := s.accounts.List()
		if lerr != nil {
			return "", "", lerr
		}
		if len(accounts) == 0 {
			return "", "", errcode.New(errcode.Unauthorized, "尚未连接 GitHub，请到系统设置完成一键授权")
		}
		account = &accounts[0] // List 排序：默认账号在前
	}
	plain, derr := secretbox.Decrypt(s.aesKey, account.Token)
	if derr != nil {
		return "", "", errcode.Newf(errcode.Internal, "解密 GitHub 令牌失败: %v", derr)
	}
	return plain, account.Login, nil
}

// GitHubWebhookDTO webhook 处理结果。
type GitHubWebhookDTO struct {
	Triggered []string `json:"triggered"` // 触发部署的应用名
	Skipped   string   `json:"skipped"`   // 跳过原因（如正在部署中）
}

// HandleGitHubWebhook 处理 push 事件：匹配 repo + branch 的应用自动重新部署。
func (s *PaasService) HandleGitHubWebhook(repo, branch string) (*GitHubWebhookDTO, error) {
	if repo == "" || branch == "" {
		return nil, errcode.New(errcode.InvalidParams, "webhook 载荷缺少仓库或分支信息")
	}
	apps, err := s.apps.ListBySource(repo, branch)
	if err != nil {
		return nil, err
	}
	out := &GitHubWebhookDTO{Triggered: []string{}, Skipped: ""}
	for i := range apps {
		if apps[i].Status == model.PaasAppDeploying {
			out.Skipped += apps[i].Name + " 正在部署中，已跳过；"
			continue
		}
		if err := s.DeployApp(apps[i].ID); err != nil {
			out.Skipped += apps[i].Name + ": " + err.Error() + "；"
			continue
		}
		out.Triggered = append(out.Triggered, apps[i].Name)
	}
	s.log.Info("GitHub webhook", "repo", repo, "branch", branch, "triggered", out.Triggered, "skipped", out.Skipped)
	return out, nil
}

// ToGitHubAccountDTO 模型转响应。
func ToGitHubAccountDTO(a *model.PaasGitHubAccount) GitHubAccountDTO {
	return GitHubAccountDTO{
		ID:        strconv.FormatUint(a.ID, 10),
		Login:     a.Login,
		IsDefault: a.IsDefault,
		CreatedAt: formatTime(a.CreatedAt),
	}
}
