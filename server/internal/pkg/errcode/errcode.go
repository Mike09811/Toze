// Package errcode 定义业务错误码。
// 约定：code == 0 表示成功，其余为业务错误；HTTP 层通常仍返回 200，
// 由前端根据 code 判断，鉴权失败等场景才使用 4xx 状态码。
package errcode

import "fmt"

// Code 业务错误码。
type Code int

const (
	OK Code = 0

	// 通用
	InvalidParams Code = 1001
	Unauthorized  Code = 1002
	Forbidden     Code = 1003
	NotFound      Code = 1004
	Conflict      Code = 1005
	Internal      Code = 1006
	TooManyReqs   Code = 1007

	// 账号
	LoginFailed      Code = 2001
	UserDisabled     Code = 2002
	UserExists       Code = 2003
	OldPasswordErr   Code = 2004
	TokenInvalid     Code = 2005
	TokenExpired     Code = 2006
	PermissionDenied Code = 2007

	// 角色 / 菜单
	RoleExists   Code = 3001
	RoleInUse    Code = 3002
	MenuExists   Code = 4001
	MenuHasChild Code = 4002
	MenuHasRole  Code = 4003

	// PaaS
	PaasSSHFailed    Code = 5001
	PaasAppExists    Code = 5002
	PaasPortConflict Code = 5003
	PaasDomainExists Code = 5004
	PaasBusy         Code = 5005
)

var messages = map[Code]string{
	OK:               "成功",
	InvalidParams:    "请求参数错误",
	Unauthorized:     "未登录或登录已过期",
	Forbidden:        "没有访问权限",
	NotFound:         "资源不存在",
	Conflict:         "资源冲突",
	Internal:         "服务器内部错误",
	TooManyReqs:      "请求过于频繁，请稍后再试",
	LoginFailed:      "用户名或密码错误",
	UserDisabled:     "账号已被禁用",
	UserExists:       "账号已存在",
	OldPasswordErr:   "原密码不正确",
	TokenInvalid:     "登录凭证无效",
	TokenExpired:     "登录已过期，请重新登录",
	PermissionDenied: "抱歉，您没有该权限",
	RoleExists:       "角色标识已存在",
	RoleInUse:        "角色正在被用户使用，无法删除",
	MenuExists:       "菜单已存在",
	MenuHasChild:     "存在子菜单，无法删除",
	MenuHasRole:      "菜单已被角色引用，无法删除",
	PaasSSHFailed:    "远程服务器执行失败",
	PaasAppExists:    "应用名已存在",
	PaasPortConflict: "宿主机端口冲突",
	PaasDomainExists: "域名已被绑定",
	PaasBusy:         "操作正在进行中，请稍候",
}

// Message 返回错误码默认文案。
func (c Code) Message() string {
	if m, ok := messages[c]; ok {
		return m
	}
	return fmt.Sprintf("未知错误(%d)", int(c))
}

// Error 实现 error 接口。
func (c Code) Error() string { return c.Message() }

// Error 是带错误码的业务错误。
type Error struct {
	Code Code
	Msg  string
}

// New 构造业务错误。
func New(code Code, msg string) *Error {
	if msg == "" {
		msg = code.Message()
	}
	return &Error{Code: code, Msg: msg}
}

// Newf 构造带格式化文案的业务错误。
func Newf(code Code, format string, args ...any) *Error {
	return &Error{Code: code, Msg: fmt.Sprintf(format, args...)}
}

// Error 实现 error 接口。
func (e *Error) Error() string { return e.Msg }

// As 从 error 中提取业务错误，非业务错误返回 Internal。
func As(err error) *Error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*Error); ok {
		return e
	}
	return New(Internal, err.Error())
}
