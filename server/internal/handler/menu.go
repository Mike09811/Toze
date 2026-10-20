package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/pkg/response"
	"github.com/Mike09811/Toze/server/internal/service"
)

// MenuHandler 菜单接口。
type MenuHandler struct{ svc *service.MenuService }

// NewMenuHandler 构造函数。
func NewMenuHandler(svc *service.MenuService) *MenuHandler { return &MenuHandler{svc: svc} }

type menuUpsertReq struct {
	ID        string `json:"id"`
	ParentID  string `json:"parentId"`
	Title     string `json:"title"`
	Subtitle  string `json:"subtitle"`
	Path      string `json:"path"`
	Name      string `json:"name"`
	Header    string `json:"header"`
	Icon      string `json:"icon"`
	Auth      string `json:"auth"`
	Target    string `json:"target"`
	Type      string `json:"type"`
	HideSider bool   `json:"hideSider"`
	Divided   bool   `json:"divided"`
	Sort      int    `json:"sort"`
}

// List POST /api/system/menu/list
func (h *MenuHandler) List(c *gin.Context) {
	res, err := h.svc.Tree()
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OK(c, res)
}

// CreateHeader POST /api/system/menu/header/create
func (h *MenuHandler) CreateHeader(c *gin.Context) {
	h.create(c, "header")
}

// CreateSider POST /api/system/menu/sider/create
func (h *MenuHandler) CreateSider(c *gin.Context) {
	h.create(c, "sider")
}

func (h *MenuHandler) create(c *gin.Context, kind string) {
	var req menuUpsertReq
	if !bind(c, &req) {
		return
	}
	dto, err := h.svc.Create(toMenuUpsert(req), kind)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "创建成功", gin.H{"msg": "创建成功", "id": dto.ID})
}

// Update POST /api/system/menu/update
func (h *MenuHandler) Update(c *gin.Context) {
	var req menuUpsertReq
	if !bind(c, &req) {
		return
	}
	dto, err := h.svc.Update(toMenuUpsert(req))
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "保存成功", gin.H{"msg": "保存成功", "id": dto.ID})
}

// Delete POST /api/system/menu/delete
func (h *MenuHandler) Delete(c *gin.Context) {
	var req deleteReq
	if !bind(c, &req) {
		return
	}
	id, err := service.ParseID(req.ID)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	if err := h.svc.Delete(id); err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "删除成功", gin.H{"msg": "删除成功"})
}

// DeleteMany POST /api/system/menu/delete/multiple
func (h *MenuHandler) DeleteMany(c *gin.Context) {
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
	if err := h.svc.DeleteMany(ids); err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "删除成功", gin.H{"msg": "删除成功"})
}

func toMenuUpsert(req menuUpsertReq) service.MenuUpsert {
	return service.MenuUpsert{
		ID: req.ID, ParentID: req.ParentID, Title: req.Title, Subtitle: req.Subtitle,
		Path: req.Path, Name: req.Name, Header: req.Header, Icon: req.Icon, Auth: req.Auth,
		Target: req.Target, Type: req.Type, HideSider: req.HideSider, Divided: req.Divided, Sort: req.Sort,
	}
}
