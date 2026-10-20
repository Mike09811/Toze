package service

import (
	"context"

	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/pkg/githubx"
	"github.com/Mike09811/Toze/server/internal/pkg/secretbox"
)

// GitHubDTO 授权状态。
type GitHubDTO struct {
	Connected bool   `json:"connected"`
	Login     string `json:"login"`
	// OAuth 未配置时无法发起一键授权，只能构建公开仓库
	OAuthConfigured bool `json:"oauth_configured"`
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

// GitHubCallback OAuth 回调：code 换 token，加密落库。
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
	setting, err := s.settings.Get()
	if err != nil {
		return err
	}
	setting.GitHubLogin = login
	setting.GitHubToken = encrypted
	if err := s.settings.Save(setting); err != nil {
		return err
	}
	s.log.Info("GitHub 授权成功", "login", login)
	return nil
}

// GitHubStatus 当前授权状态。
func (s *PaasService) GitHubStatus() (*GitHubDTO, error) {
	setting, err := s.settings.Get()
	if err != nil {
		return nil, err
	}
	dto := &GitHubDTO{
		Connected:       setting.GitHubToken != "",
		Login:           setting.GitHubLogin,
		OAuthConfigured: s.githubClientID != "" && s.githubClientSecret != "",
	}
	return dto, nil
}

// GitHubRepos 列出已授权用户的仓库。
func (s *PaasService) GitHubRepos(ctx context.Context) ([]RepoDTO, error) {
	token, err := s.githubToken()
	if err != nil {
		return nil, err
	}
	repos, err := githubx.ListRepos(ctx, token)
	if err != nil {
		return nil, errcode.Newf(errcode.Internal, "%v", err)
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

// GitHubDisconnect 断开授权。
func (s *PaasService) GitHubDisconnect() error {
	setting, err := s.settings.Get()
	if err != nil {
		return err
	}
	setting.GitHubLogin = ""
	setting.GitHubToken = ""
	return s.settings.Save(setting)
}

// githubToken 解密 GitHub token；未授权时报错。
func (s *PaasService) githubToken() (string, error) {
	setting, err := s.settings.Get()
	if err != nil {
		return "", err
	}
	if setting.GitHubToken == "" {
		return "", errcode.New(errcode.Unauthorized, "尚未连接 GitHub，请先在应用管理页完成一键授权")
	}
	token, err := secretbox.Decrypt(s.aesKey, setting.GitHubToken)
	if err != nil {
		return "", errcode.Newf(errcode.Internal, "解密 GitHub 令牌失败: %v", err)
	}
	return token, nil
}
