// Package response 统一 HTTP 响应格式。
// 成功：{"code":0,"msg":"成功","data":{...}}
// 失败：{"code":<非0>,"msg":"原因","data":{}}
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
)

// Body 是统一响应体。
type Body struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

// Page 是分页数据结构。
type Page struct {
	List  any   `json:"list"`
	Total int64 `json:"total"`
}

// OK 返回成功数据。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{Code: 0, Msg: "成功", Data: data})
}

// OKMsg 返回带自定义提示的成功数据。
func OKMsg(c *gin.Context, msg string, data any) {
	if msg == "" {
		msg = "成功"
	}
	c.JSON(http.StatusOK, Body{Code: 0, Msg: msg, Data: data})
}

// OKPage 返回分页数据。
func OKPage(c *gin.Context, list any, total int64) {
	c.JSON(http.StatusOK, Body{Code: 0, Msg: "成功", Data: Page{List: list, Total: total}})
}

// Fail 业务失败：HTTP 200 + 非 0 code，前端拦截器可统一提示。
func Fail(c *gin.Context, code errcode.Code, msg string) {
	if msg == "" {
		msg = code.Message()
	}
	c.JSON(http.StatusOK, Body{Code: int(code), Msg: msg, Data: gin.H{}})
}

// FailErr 业务失败，自动解包 errcode.Error。
func FailErr(c *gin.Context, err error) {
	e := errcode.As(err)
	Fail(c, e.Code, e.Msg)
}

// Abort 鉴权/路由级失败：写入 HTTP 状态码并终止链路。
func Abort(c *gin.Context, status int, code errcode.Code, msg string) {
	if msg == "" {
		msg = code.Message()
	}
	c.AbortWithStatusJSON(status, Body{Code: int(code), Msg: msg, Data: gin.H{}})
}
