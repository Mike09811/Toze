package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/pkg/response"
	"github.com/Mike09811/Toze/server/internal/service"
)

// AuthHandler 认证相关接口。
type AuthHandler struct{ svc *service.AuthService }

// NewAuthHandler 构造函数。
func NewAuthHandler(svc *service.AuthService) *AuthHandler { return &AuthHandler{svc: svc} }

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type registerReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name"`
}

type changePwdReq struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword" binding:"required"`
}

// Login POST /api/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if !bind(c, &req) {
		return
	}
	res, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "登录成功", res)
}

// Register POST /api/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerReq
	if !bind(c, &req) {
		return
	}
	res, err := h.svc.Register(req.Username, req.Password, req.Name)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "注册成功", res)
}

// Logout POST /api/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	tc := claims(c)
	if tc == nil {
		response.Abort(c, 401, errcode.Unauthorized, "")
		return
	}
	exp := tc.ExpiresAt.Time
	if err := h.svc.Logout(tc.ID, exp); err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "退出成功", gin.H{})
}

// Profile GET /api/user/info
func (h *AuthHandler) Profile(c *gin.Context) {
	p, err := h.svc.Profile(currentUserID(c))
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OK(c, p)
}

// ChangePassword POST /api/user/password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePwdReq
	if !bind(c, &req) {
		return
	}
	if err := h.svc.ChangePassword(currentUserID(c), req.OldPassword, req.NewPassword); err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "密码修改成功", gin.H{})
}
