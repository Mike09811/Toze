package service

import (
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/pkg/pagination"
	"github.com/Mike09811/Toze/server/internal/repository"
)

// RoleUpsert 角色入参。
type RoleUpsert struct {
	ID          string
	Name        string
	Desc        string
	DefaultRole int
	MenuList    []string
}

// RoleService 角色业务逻辑。
type RoleService struct {
	roles *repository.RoleRepository
	menus *repository.MenuRepository
	users *repository.UserRepository
	log   *slog.Logger
}

// NewRoleService 构造函数。
func NewRoleService(roles *repository.RoleRepository, menus *repository.MenuRepository,
	users *repository.UserRepository, log *slog.Logger) *RoleService {
	return &RoleService{roles: roles, menus: menus, users: users, log: log}
}

// List 分页查询。
func (s *RoleService) List(offset, limit int, sortKey, sortType string) ([]RoleDTO, int64, error) {
	p := pagination.Parse(offset, limit, sortKey, sortType)
	roles, total, err := s.roles.List(p)
	if err != nil {
		return nil, 0, err
	}
	return ToRoleDTOs(roles), total, nil
}

// All 全部角色。
func (s *RoleService) All() ([]RoleDTO, error) {
	roles, err := s.roles.All()
	if err != nil {
		return nil, err
	}
	out := make([]RoleDTO, 0, len(roles))
	for i := range roles {
		out = append(out, RoleDTO{ID: idString(roles[i].ID), Name: roles[i].Name, Desc: roles[i].Desc, DefaultRole: roles[i].DefaultRole})
	}
	return out, nil
}

// Create 新增角色。
func (s *RoleService) Create(req RoleUpsert) (*RoleDTO, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errcode.New(errcode.InvalidParams, "角色标识不能为空")
	}
	exists, err := s.roles.ExistsByName(name, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errcode.New(errcode.RoleExists, "")
	}
	menus, err := s.menusByIDs(req.MenuList)
	if err != nil {
		return nil, err
	}
	role := &model.Role{Name: name, Desc: req.Desc, DefaultRole: req.DefaultRole}
	if err := s.roles.Create(role, menus); err != nil {
		return nil, err
	}
	created, err := s.roles.FindByID(role.ID)
	if err != nil {
		return nil, err
	}
	dto := ToRoleDTO(created)
	return &dto, nil
}

// Update 编辑角色。
func (s *RoleService) Update(req RoleUpsert) (*RoleDTO, error) {
	id, err := ParseID(req.ID)
	if err != nil {
		return nil, err
	}
	role, err := s.roles.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errcode.New(errcode.NotFound, "角色不存在")
		}
		return nil, err
	}
	if name := strings.TrimSpace(req.Name); name != "" && name != role.Name {
		if role.Name == "ROLE_ADMIN" {
			return nil, errcode.New(errcode.Forbidden, "内置超级管理员角色不可重命名")
		}
		exists, err := s.roles.ExistsByName(name, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errcode.New(errcode.RoleExists, "")
		}
		role.Name = name
	}
	role.Desc = req.Desc
	role.DefaultRole = req.DefaultRole

	var menus []model.Menu
	if req.MenuList != nil {
		menus, err = s.menusByIDs(req.MenuList)
		if err != nil {
			return nil, err
		}
	}
	if err := s.roles.Update(role, menus); err != nil {
		return nil, err
	}
	updated, err := s.roles.FindByID(id)
	if err != nil {
		return nil, err
	}
	dto := ToRoleDTO(updated)
	return &dto, nil
}

// UpdateMenus 分配角色菜单。
func (s *RoleService) UpdateMenus(id string, menuIDs []string) error {
	roleID, err := ParseID(id)
	if err != nil {
		return err
	}
	role, err := s.roles.FindByID(roleID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errcode.New(errcode.NotFound, "角色不存在")
		}
		return err
	}
	menus, err := s.menusByIDs(menuIDs)
	if err != nil {
		return err
	}
	return s.roles.UpdateMenus(role, menus)
}

// Delete 删除角色。
func (s *RoleService) Delete(id uint64) error {
	role, err := s.roles.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errcode.New(errcode.NotFound, "角色不存在")
		}
		return err
	}
	if role.Name == "ROLE_ADMIN" {
		return errcode.New(errcode.Forbidden, "内置超级管理员角色不可删除")
	}
	count, err := s.users.CountByRole(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errcode.New(errcode.RoleInUse, "")
	}
	return s.roles.Delete(id)
}

// DeleteMany 批量删除。
func (s *RoleService) DeleteMany(ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	roles, err := s.roles.FindByIDs(ids)
	if err != nil {
		return err
	}
	for _, r := range roles {
		if r.Name == "ROLE_ADMIN" {
			return errcode.New(errcode.Forbidden, "内置超级管理员角色不可删除")
		}
		count, err := s.users.CountByRole(r.ID)
		if err != nil {
			return err
		}
		if count > 0 {
			return errcode.New(errcode.RoleInUse, "角色 "+r.Name+" 正在被用户使用")
		}
	}
	return s.roles.DeleteMany(ids)
}

func (s *RoleService) menusByIDs(ids []string) ([]model.Menu, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	parsed := make([]uint64, 0, len(ids))
	for _, raw := range ids {
		id, err := ParseID(raw)
		if err != nil {
			continue
		}
		parsed = append(parsed, id)
	}
	return s.menus.FindByIDs(parsed)
}

func idString(id uint64) string { return strconv.FormatUint(id, 10) }
