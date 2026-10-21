// Package githubx 封装 GitHub OAuth 与 REST API 访问。
package githubx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Repo 仓库摘要。
type Repo struct {
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
	Private       bool   `json:"private"`
	UpdatedAt     string `json:"updated_at"`
}

// StatusError GitHub API 非 2xx 响应，携带状态码供调用方区分失败原因。
type StatusError struct {
	Status int
	Body   string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("GitHub 返回 %d: %s", e.Status, truncate(e.Body, 200))
}

// AuthorizeURL 生成一键授权跳转地址。
func AuthorizeURL(clientID string, scopes string) string {
	return "https://github.com/login/oauth/authorize?client_id=" +
		url.QueryEscape(clientID) + "&scope=" + url.QueryEscape(scopes)
}

// Exchange 用回调 code 换 access token。
func Exchange(ctx context.Context, clientID, clientSecret, code string) (string, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("code", code)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 GitHub OAuth 失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var out struct {
		AccessToken      string `json:"access_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("解析 OAuth 响应失败: %w", err)
	}
	if out.AccessToken == "" {
		msg := out.ErrorDescription
		if msg == "" {
			msg = out.Error
		}
		return "", fmt.Errorf("GitHub 授权失败: %s", msg)
	}
	return out.AccessToken, nil
}

// User 获取 token 对应的 GitHub 登录名。
func User(ctx context.Context, token string) (string, error) {
	var out struct {
		Login string `json:"login"`
	}
	if err := api(ctx, token, "/user", &out); err != nil {
		return "", err
	}
	return out.Login, nil
}

// ListRepos 列出 token 拥有的仓库（按更新时间倒序，最多 100 个）。
func ListRepos(ctx context.Context, token string) ([]Repo, error) {
	var out []Repo
	if err := api(ctx, token, "/user/repos?per_page=100&sort=updated&type=all", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func api(ctx context.Context, token, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com"+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	ctx2, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req = req.WithContext(ctx2)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求 GitHub API 失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return &StatusError{Status: resp.StatusCode, Body: string(body)}
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("解析 GitHub API 响应失败: %w", err)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// CreateRepoWebhook 为仓库配置 push webhook（幂等：已存在同 URL 的 hook 会报 422，调用方忽略）。
func CreateRepoWebhook(ctx context.Context, token, repo, callbackURL string) error {
	payload := map[string]any{
		"config": map[string]any{
			"url":          callbackURL,
			"content_type": "json",
			"insecure_ssl": "1",
		},
		"events": []string{"push"},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.github.com/repos/"+repo+"/hooks", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求 GitHub API 失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusUnprocessableEntity {
		return &StatusError{Status: resp.StatusCode, Body: string(respBody)}
	}
	return nil
}
