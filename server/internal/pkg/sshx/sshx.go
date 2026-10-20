// Package sshx 封装对纳管服务器的远程脚本执行。
// 脚本通过 stdin 交给 bash -s 执行，天然规避命令行转义问题；
// 拼接脚本内部的变量仍必须使用 Quote 做单引号包裹。
package sshx

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// 认证方式。
const (
	AuthPassword   = 1
	AuthPrivateKey = 2
)

// Runner SSH 执行器，自用场景每次执行新建连接，不做连接池。
type Runner struct {
	DialTimeout time.Duration // 建立连接超时
}

// NewRunner 默认配置：10 秒连接超时，执行时长由调用方 ctx 控制。
func NewRunner() *Runner { return &Runner{DialTimeout: 10 * time.Second} }

// Exec 在服务器上执行脚本，返回标准输出；失败时错误信息携带 stderr 摘要。
func (r *Runner) Exec(ctx context.Context, host string, port int, user string, authType int, secret, script string) (string, error) {
	auth, err := buildAuth(authType, secret)
	if err != nil {
		return "", err
	}
	dialTimeout := r.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = 10 * time.Second
	}
	dialer := &net.Dialer{Timeout: dialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return "", fmt.Errorf("连接 %s:%d 失败: %w", host, port, err)
	}
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, fmt.Sprintf("%s:%d", host, port), &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 自有服务器纳管，不校验主机指纹
		Timeout:         dialTimeout,
	})
	if err != nil {
		return "", fmt.Errorf("SSH 认证失败（%s@%s）: %w", user, host, err)
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建会话失败: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdin = strings.NewReader(script)
	session.Stdout = &stdout
	session.Stderr = &stderr
	if err := session.Start("bash -s"); err != nil {
		return "", fmt.Errorf("启动远程执行失败: %w", err)
	}
	// ctx 取消时关闭底层连接中断执行
	done := make(chan error, 1)
	go func() { done <- session.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("远程执行失败: %w；stderr: %s", err, tail(stderr.String()))
		}
	case <-ctx.Done():
		conn.Close()
		return "", fmt.Errorf("远程执行超时或被取消: %w", ctx.Err())
	}
	return stdout.String(), nil
}

// Quote 把任意字符串安全地包裹为 POSIX 单引号字面量。
func Quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func buildAuth(authType int, secret string) (ssh.AuthMethod, error) {
	switch authType {
	case AuthPassword:
		return ssh.Password(secret), nil
	case AuthPrivateKey:
		signer, err := ssh.ParsePrivateKey([]byte(secret))
		if err != nil {
			return nil, fmt.Errorf("解析私钥失败: %w", err)
		}
		return ssh.PublicKeys(signer), nil
	default:
		return nil, fmt.Errorf("未知认证方式: %d", authType)
	}
}

func tail(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 500 {
		return s[len(s)-500:]
	}
	return s
}
