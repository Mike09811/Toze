package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/pkg/response"
	"github.com/Mike09811/Toze/server/internal/service"
)

// UserHandler 用户接口。
type UserHandler struct{ svc *service.UserService }

// NewUserHandler 构造函数。
func NewUserHandler(svc *service.UserService) *UserHandler { return &UserHandler{svc: svc} }

type userListReq struct {
	Name     string   `json:"name"`
	Phone    string   `json:"phone"`
	Mail     string   `json:"mail"`
	Account  string   `json:"account"`
	ID       string   `json:"id"`
	Gender   int      `json:"gender"`
	Status   int      `json:"status"`
	Date     []string `json:"date"`
	Offset   int      `json:"offset"`
	Limit    int      `json:"limit"`
	SortType string   `json:"sortType"`
	SortKey  string   `json:"sortKey"`
}

type userUpsertReq struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Account  string   `json:"account"`
	Password string   `json:"password"`
	Phone    string   `json:"phone"`
	Mail     string   `json:"mail"`
	Avatar   string   `json:"avatar"`
	Gender   int      `json:"gender"`
	Status   int      `json:"status"`
	Birthday string   `json:"birthday"`
	Region   []string `json:"region"`
	Role     []string `json:"role"`
	Desc     string   `json:"desc"`
}

type deleteReq struct {
	ID string `json:"id"`
}

type deleteManyReq struct {
	IDs []string `json:"ids"`
}

// List POST /api/system/user/list
func (h *UserHandler) List(c *gin.Context) {
	var req userListReq
	if !bind(c, &req) {
		return
	}
	list, total, err := h.svc.List(service.UserListQuery{
		Name: req.Name, Phone: req.Phone, Mail: req.Mail, Account: req.Account,
		ID: req.ID, Gender: req.Gender, Status: req.Status, Date: req.Date,
		Offset: req.Offset, Limit: req.Limit, SortType: req.SortType, SortKey: req.SortKey,
	})
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKPage(c, list, total)
}

// Create POST /api/system/user/create
func (h *UserHandler) Create(c *gin.Context) {
	var req userUpsertReq
	if !bind(c, &req) {
		return
	}
	dto, err := h.svc.Create(toUserUpsert(req))
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "创建成功", gin.H{"msg": "创建成功", "id": dto.ID})
}

// Update POST /api/system/user/update
func (h *UserHandler) Update(c *gin.Context) {
	var req userUpsertReq
	if !bind(c, &req) {
		return
	}
	dto, err := h.svc.Update(toUserUpsert(req))
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "保存成功", gin.H{"msg": "保存成功", "id": dto.ID})
}

// Delete POST /api/system/user/delete
func (h *UserHandler) Delete(c *gin.Context) {
	var req deleteReq
	if !bind(c, &req) {
		return
	}
	id, err := service.ParseID(req.ID)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	if err := h.svc.Delete(id, currentUserID(c)); err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "删除成功", gin.H{"msg": "删除成功"})
}

// DeleteMany POST /api/system/user/delete/multiple
func (h *UserHandler) DeleteMany(c *gin.Context) {
	var req deleteManyReq
	if !bind(c, &req) {
		return
	}
	ids := make([]uint64, 0, len(req.IDs))
	for _, raw := range req.IDs {
		id, err := service.ParseID(raw)
		if err != nil {
			response.Fail(c, errcode.InvalidParams, "ID 格式错误")
			return
		}
		ids = append(ids, id)
	}
	if err := h.svc.DeleteMany(ids, currentUserID(c)); err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "删除成功", gin.H{"msg": "删除成功"})
}

func toUserUpsert(req userUpsertReq) service.UserUpsert {
	return service.UserUpsert{
		ID: req.ID, Name: req.Name, Account: req.Account, Password: req.Password,
		Phone: req.Phone, Mail: req.Mail, Avatar: req.Avatar, Gender: req.Gender,
		Status: req.Status, Birthday: req.Birthday, Region: req.Region, Role: req.Role, Desc: req.Desc,
	}
}
