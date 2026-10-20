// Package middleware 提供通用 Gin 中间件。
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/pkg/response"
	"github.com/Mike09811/Toze/server/internal/pkg/token"
	"github.com/Mike09811/Toze/server/internal/service"
)

const (
	// CtxClaims 令牌声明在上下文中的键。
	CtxClaims = "claims"
	// CtxRequestID 请求 ID 上下文键。
	CtxRequestID = "request_id"
)

// RequestID 为每个请求生成唯一 ID。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		c.Set(CtxRequestID, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// AccessLog 输出结构化访问日志。
func AccessLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if c.Request.URL.Path == "/healthz" {
			return
		}
		log.Info("http_request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
			"request_id", c.GetString(CtxRequestID),
		)
	}
}

// Recovery 捕获 panic 并返回统一错误。
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("panic_recovered", "error", err, "path", c.Request.URL.Path)
				response.Abort(c, http.StatusInternalServerError, errcode.Internal, "")
			}
		}()
		c.Next()
	}
}

// CORS 跨域支持，仅放行配置的来源。
func CORS(origins []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(origins))
	for _, o := range origins {
		allowed[o] = true
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && (allowed[origin] || contains(origins, "*")) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// TokenChecker 判断令牌是否已被注销。
type TokenChecker interface {
	IsRevoked(jti string) (bool, error)
}

// Auth 校验 JWT 并注入用户信息。
func Auth(jwt *token.Manager, tokens TokenChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := extractToken(c)
		if raw == "" {
			response.Abort(c, http.StatusUnauthorized, errcode.Unauthorized, "未提供登录凭证")
			return
		}
		claims, err := jwt.Parse(raw)
		if err != nil {
			msg := "登录凭证无效"
			if err == token.ErrExpired {
				msg = "登录已过期，请重新登录"
			}
			response.Abort(c, http.StatusUnauthorized, errcode.Unauthorized, msg)
			return
		}
		if tokens != nil && claims.ID != "" {
			if revoked, _ := tokens.IsRevoked(claims.ID); revoked {
				response.Abort(c, http.StatusUnauthorized, errcode.Unauthorized, "登录已失效，请重新登录")
				return
			}
		}
		c.Set(CtxClaims, claims)
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	if auth != "" && !strings.Contains(auth, " ") {
		return auth
	}
	if v, err := c.Cookie("token"); err == nil && v != "" {
		return v
	}
	return ""
}

// RequireRoles 校验角色。
func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get(CtxClaims)
		if !ok {
			response.Abort(c, http.StatusUnauthorized, errcode.Unauthorized, "")
			return
		}
		claims, ok := v.(*token.Claims)
		if !ok || !service.HasRole(claims.Roles, roles...) {
			response.Abort(c, http.StatusForbidden, errcode.Forbidden, "")
			return
		}
		c.Next()
	}
}

// Audit 对写操作记录审计日志。
func Audit(logSvc *service.LogService) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.Request.URL.Path
		if strings.Contains(path, "/list") || c.Request.Method != http.MethodPost {
			return
		}
		entry := &model.OperationLog{
			Action:    actionOf(path),
			Method:    c.Request.Method,
			Path:      path,
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Status:    c.Writer.Status(),
			LatencyMS: time.Since(start).Milliseconds(),
		}
		if v, ok := c.Get(CtxClaims); ok {
			if claims, ok := v.(*token.Claims); ok {
				entry.UserID = claims.UserID
				entry.Username = claims.Subject
			}
		}
		logSvc.Record(entry)
	}
}

func actionOf(path string) string {
	switch {
	case strings.Contains(path, "/login"):
		return "登录"
	case strings.Contains(path, "/register"):
		return "注册"
	case strings.Contains(path, "/logout"):
		return "退出"
	case strings.Contains(path, "create"):
		return "新增"
	case strings.Contains(path, "update"):
		return "修改"
	case strings.Contains(path, "delete"):
		return "删除"
	default:
		return "其它"
	}
}
