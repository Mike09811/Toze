// Package service 业务逻辑层。
package service

import (
	"strconv"
	"time"

	"github.com/Mike09811/Toze/server/internal/model"
)

// UserDTO 用户响应结构，字段与前端表格一致。
type UserDTO struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Avatar   string   `json:"avatar"`
	Account  string   `json:"account"`
	Phone    string   `json:"phone"`
	Mail     string   `json:"mail"`
	Gender   int      `json:"gender"`
	Status   int      `json:"status"`
	Date     string   `json:"date"`
	Birthday string   `json:"birthday"`
	Region   []string `json:"region"`
	Role     []string `json:"role"`
	Desc     string   `json:"desc"`
}

// RoleDTO 角色响应结构。
type RoleDTO struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Desc        string   `json:"desc"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	DefaultRole int      `json:"default_role"`
	MenuList    []string `json:"menu_list"`
}

// MenuDTO 菜单响应结构。
type MenuDTO struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Subtitle  string    `json:"subtitle,omitempty"`
	Path      string    `json:"path"`
	Name      string    `json:"name,omitempty"`
	Header    string    `json:"header,omitempty"`
	Icon      string    `json:"icon,omitempty"`
	Auth      string    `json:"auth,omitempty"`
	Target    string    `json:"target,omitempty"`
	Type      string    `json:"type,omitempty"`
	HideSider bool      `json:"hideSider,omitempty"`
	Divided   bool      `json:"divided,omitempty"`
	Sort      int       `json:"sort"`
	Children  []MenuDTO `json:"children,omitempty"`
}

// ProfileDTO 登录用户信息。
type ProfileDTO struct {
	UUID    string   `json:"uuid"`
	Name    string   `json:"name"`
	Avatar  string   `json:"avatar"`
	Access  []string `json:"access"`
	Account string   `json:"account"`
	Phone   string   `json:"phone"`
	Mail    string   `json:"mail"`
	Gender  int      `json:"gender"`
	Desc    string   `json:"desc"`
}

// LoginResult 登录响应。
type LoginResult struct {
	Username  string     `json:"username"`
	UUID      string     `json:"uuid"`
	Info      ProfileDTO `json:"info"`
	Token     string     `json:"token"`
	ExpiresIn int64      `json:"expires_in"`
}

const timeLayout = "2006-01-02 15:04:05"

// ToUserDTO 模型转响应。
func ToUserDTO(u *model.User) UserDTO {
	region := u.Region
	if region == nil {
		region = []string{}
	}
	roles := u.RoleCodes()
	if roles == nil {
		roles = []string{}
	}
	return UserDTO{
		ID:       strconv.FormatUint(u.ID, 10),
		Name:     u.Name,
		Avatar:   u.Avatar,
		Account:  u.Account,
		Phone:    u.Phone,
		Mail:     u.Mail,
		Gender:   u.Gender,
		Status:   u.Status,
		Date:     u.CreatedAt.Format(timeLayout),
		Birthday: u.Birthday,
		Region:   region,
		Role:     roles,
		Desc:     u.Desc,
	}
}

// ToUserDTOs 批量转换。
func ToUserDTOs(users []model.User) []UserDTO {
	out := make([]UserDTO, 0, len(users))
	for i := range users {
		out = append(out, ToUserDTO(&users[i]))
	}
	return out
}

// ToRoleDTO 模型转响应。
func ToRoleDTO(r *model.Role) RoleDTO {
	menus := make([]string, 0, len(r.Menus))
	for _, m := range r.Menus {
		menus = append(menus, strconv.FormatUint(m.ID, 10))
	}
	return RoleDTO{
		ID:          strconv.FormatUint(r.ID, 10),
		Name:        r.Name,
		Desc:        r.Desc,
		CreatedAt:   r.CreatedAt.Format(timeLayout),
		UpdatedAt:   r.UpdatedAt.Format(timeLayout),
		DefaultRole: r.DefaultRole,
		MenuList:    menus,
	}
}

// ToRoleDTOs 批量转换。
func ToRoleDTOs(roles []model.Role) []RoleDTO {
	out := make([]RoleDTO, 0, len(roles))
	for i := range roles {
		out = append(out, ToRoleDTO(&roles[i]))
	}
	return out
}

// ToMenuDTOs 树形转换。
func ToMenuDTOs(menus []*model.Menu) []MenuDTO {
	out := make([]MenuDTO, 0, len(menus))
	for _, m := range menus {
		dto := MenuDTO{
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
		if len(m.Children) > 0 {
			dto.Children = ToMenuDTOs(m.Children)
		}
		out = append(out, dto)
	}
	return out
}

func formatTime(t time.Time) string { return t.Format(timeLayout) }

// PaasServerDTO 服务器响应结构，凭证字段永不下发。
type PaasServerDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
	SSHPort   int    `json:"ssh_port"`
	SSHUser   string `json:"ssh_user"`
	AuthType  int    `json:"auth_type"`
	Status    int    `json:"status"`
	DockerVer string `json:"docker_ver"`
	LastError string `json:"last_error"`
	CreatedAt string `json:"created_at"`
	AppCount  int64  `json:"app_count"`
}

// PaasAppDTO 应用响应结构。
type PaasAppDTO struct {
	ID              string   `json:"id"`
	ServerID        string   `json:"server_id"`
	ServerName      string   `json:"server_name"`
	Name            string   `json:"name"`
	Image           string   `json:"image"`
	SourceRepo      string   `json:"source_repo"`
	SourceBranch    string   `json:"source_branch"`
	SourceAccountID uint64   `json:"source_account_id"`
	ContainerPort   int      `json:"container_port"`
	HostPort        int      `json:"host_port"`
	Env             []string `json:"env"`
	Volumes         []string `json:"volumes"`
	Restart         string   `json:"restart"`
	ContainerID     string   `json:"container_id"`
	Status          int      `json:"status"`
	LastError       string   `json:"last_error"`
	CreatedAt       string   `json:"created_at"`
}

// PaasDomainDTO 域名响应结构。
type PaasDomainDTO struct {
	ID         string `json:"id"`
	AppID      string `json:"app_id"`
	AppName    string `json:"app_name"`
	ServerID   string `json:"server_id"`
	ServerName string `json:"server_name"`
	Domain     string `json:"domain"`
	HostPort   int    `json:"host_port"`
	CreatedAt  string `json:"created_at"`
}
