package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/Mike09811/Toze/server/internal/pkg/response"
	"github.com/Mike09811/Toze/server/internal/service"
)

// RoleHandler 角色接口。
type RoleHandler struct{ svc *service.RoleService }

// NewRoleHandler 构造函数。
func NewRoleHandler(svc *service.RoleService) *RoleHandler { return &RoleHandler{svc: svc} }

type roleListReq struct {
	Offset   int    `json:"offset"`
	Limit    int    `json:"limit"`
	SortType string `json:"sortType"`
	SortKey  string `json:"sortKey"`
}

type roleUpsertReq struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Desc        string   `json:"desc"`
	DefaultRole int      `json:"default_role"`
	MenuList    []string `json:"menu_list"`
}

type roleMenuReq struct {
	ID      string   `json:"id"`
	MenuIDs []string `json:"menuIds"`
	Menus   []string `json:"menu_list"`
}

// List POST /api/system/role/list
func (h *RoleHandler) List(c *gin.Context) {
	var req roleListReq
	if !bind(c, &req) {
		return
	}
	list, total, err := h.svc.List(req.Offset, req.Limit, req.SortKey, req.SortType)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKPage(c, list, total)
}

// All POST /api/system/role/all
func (h *RoleHandler) All(c *gin.Context) {
	list, err := h.svc.All()
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OK(c, gin.H{"list": list})
}

// Create POST /api/system/role/create
func (h *RoleHandler) Create(c *gin.Context) {
	var req roleUpsertReq
	if !bind(c, &req) {
		return
	}
	dto, err := h.svc.Create(service.RoleUpsert{
		Name: req.Name, Desc: req.Desc, DefaultRole: req.DefaultRole, MenuList: req.MenuList,
	})
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "创建成功", gin.H{"msg": "创建成功", "id": dto.ID})
}

// Update POST /api/system/role/update
func (h *RoleHandler) Update(c *gin.Context) {
	var req roleUpsertReq
	if !bind(c, &req) {
		return
	}
	dto, err := h.svc.Update(service.RoleUpsert{
		ID: req.ID, Name: req.Name, Desc: req.Desc, DefaultRole: req.DefaultRole, MenuList: req.MenuList,
	})
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "保存成功", gin.H{"msg": "保存成功", "id": dto.ID})
}

// UpdateMenus POST /api/system/role/menu/update
func (h *RoleHandler) UpdateMenus(c *gin.Context) {
	var req roleMenuReq
	if !bind(c, &req) {
		return
	}
	ids := req.MenuIDs
	if len(ids) == 0 {
		ids = req.Menus
	}
	if err := h.svc.UpdateMenus(req.ID, ids); err != nil {
		response.FailErr(c, err)
		return
	}
	response.OKMsg(c, "权限保存成功", gin.H{"msg": "权限保存成功"})
}

// Delete POST /api/system/role/delete
func (h *RoleHandler) Delete(c *gin.Context) {
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

// DeleteMany POST /api/system/role/delete/multiple
func (h *RoleHandler) DeleteMany(c *gin.Context) {
	var req deleteManyReq
	if !bind(c, &req) {
		return
	}
	ids := make([]uint64, 0, len(req.IDs))
	for _, raw := range req.IDs {
		id, err := service.ParseID(raw)
		if err != nil {
			response.Fail(c, 1001, "ID 格式错误")
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
