package service

import (
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/repository"
)

// MenuUpsert 菜单入参。
type MenuUpsert struct {
	ID        string
	ParentID  string
	Title     string
	Subtitle  string
	Path      string
	Name      string
	Header    string
	Icon      string
	Auth      string
	Target    string
	Type      string
	HideSider bool
	Divided   bool
	Sort      int
}

// MenuListResult 菜单列表响应，同时提供 list / header / sider 三种视图。
type MenuListResult struct {
	List   []MenuDTO `json:"list"`
	Header []MenuDTO `json:"header"`
	Sider  []MenuDTO `json:"sider"`
}

// MenuService 菜单业务逻辑。
type MenuService struct {
	menus *repository.MenuRepository
	log   *slog.Logger
}

// NewMenuService 构造函数。
func NewMenuService(menus *repository.MenuRepository, log *slog.Logger) *MenuService {
	return &MenuService{menus: menus, log: log}
}

// Tree 返回菜单树及各视图。
func (s *MenuService) Tree() (*MenuListResult, error) {
	all, err := s.menus.All()
	if err != nil {
		return nil, err
	}
	roots := model.BuildTree(all)
	list := ToMenuDTOs(roots)

	header := make([]MenuDTO, 0, len(list))
	sider := make([]MenuDTO, 0, len(list))
	for _, root := range list {
		if root.Type == "header" || root.Type == "" {
			header = append(header, MenuDTO{ID: root.ID, Title: root.Title, Path: root.Path, Name: root.Name, Type: root.Type, Sort: root.Sort})
		}
		if len(root.Children) > 0 {
			sider = append(sider, root.Children...)
		}
	}
	return &MenuListResult{List: list, Header: header, Sider: sider}, nil
}

// Create 新增菜单。
func (s *MenuService) Create(req MenuUpsert, kind string) (*MenuDTO, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, errcode.New(errcode.InvalidParams, "菜单标题不能为空")
	}
	if strings.TrimSpace(req.Path) == "" {
		return nil, errcode.New(errcode.InvalidParams, "菜单路径不能为空")
	}
	menuType := req.Type
	if menuType == "" {
		if kind == "header" {
			menuType = "header"
		} else {
			menuType = "sider"
		}
	}
	parentID := uint64(0)
	if req.ParentID != "" {
		id, err := ParseID(req.ParentID)
		if err != nil {
			return nil, err
		}
		if _, err := s.menus.FindByID(id); err != nil {
			return nil, errcode.New(errcode.InvalidParams, "父级菜单不存在")
		}
		parentID = id
	}
	menu := &model.Menu{
		ParentID:  parentID,
		Title:     title,
		Subtitle:  req.Subtitle,
		Path:      req.Path,
		Name:      req.Name,
		Header:    req.Header,
		Icon:      req.Icon,
		Auth:      req.Auth,
		Target:    req.Target,
		Type:      menuType,
		HideSider: req.HideSider,
		Divided:   req.Divided,
		Sort:      req.Sort,
	}
	if err := s.menus.Create(menu); err != nil {
		return nil, err
	}
	dto := toMenuDTO(menu)
	return &dto, nil
}

// Update 编辑菜单。
func (s *MenuService) Update(req MenuUpsert) (*MenuDTO, error) {
	id, err := ParseID(req.ID)
	if err != nil {
		return nil, err
	}
	menu, err := s.menus.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errcode.New(errcode.NotFound, "菜单不存在")
		}
		return nil, err
	}
	if t := strings.TrimSpace(req.Title); t != "" {
		menu.Title = t
	}
	if req.Path != "" {
		menu.Path = req.Path
	}
	menu.Subtitle = req.Subtitle
	menu.Name = req.Name
	menu.Header = req.Header
	menu.Icon = req.Icon
	menu.Auth = req.Auth
	menu.Target = req.Target
	menu.HideSider = req.HideSider
	menu.Divided = req.Divided
	menu.Sort = req.Sort
	if req.Type != "" {
		menu.Type = req.Type
	}
	if req.ParentID != "" {
		pid, err := ParseID(req.ParentID)
		if err != nil {
			return nil, err
		}
		menu.ParentID = pid
	}
	if err := s.menus.Update(menu); err != nil {
		return nil, err
	}
	dto := toMenuDTO(menu)
	return &dto, nil
}

// Delete 删除菜单。
func (s *MenuService) Delete(id uint64) error {
	if _, err := s.menus.FindByID(id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errcode.New(errcode.NotFound, "菜单不存在")
		}
		return err
	}
	hasChild, err := s.menus.HasChildren(id)
	if err != nil {
		return err
	}
	if hasChild {
		return errcode.New(errcode.MenuHasChild, "")
	}
	count, err := s.menus.CountRoles(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errcode.New(errcode.MenuHasRole, "")
	}
	return s.menus.Delete(id)
}

// DeleteMany 批量删除。
func (s *MenuService) DeleteMany(ids []uint64) error {
	for _, id := range ids {
		hasChild, err := s.menus.HasChildren(id)
		if err != nil {
			return err
		}
		if hasChild {
			return errcode.New(errcode.MenuHasChild, "选中的菜单包含子菜单")
		}
	}
	return s.menus.DeleteMany(ids)
}

func toMenuDTO(m *model.Menu) MenuDTO {
	return MenuDTO{
		ID:        strconv.FormatUint(m.ID, 10),
		Title:     m.Title,
		Subtitle:  m.Subtitle,
		Path:      m.Path,
		Name:      m.Name,
		Header:    m.Header,
		Icon:      m.Icon,
		Auth:      m.Auth,
		Target:    m.Target,
		Type:      m.Type,
		HideSider: m.HideSider,
		Divided:   m.Divided,
		Sort:      m.Sort,
	}
}
