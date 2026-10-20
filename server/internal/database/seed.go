package database

import (
	"embed"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/hash"
)

//go:embed seed/*.json
var seedFS embed.FS

// DefaultPassword 种子用户统一密码。
const DefaultPassword = "admin123"

type seedMenu struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	Subtitle string     `json:"subtitle"`
	Path     string     `json:"path"`
	Name     string     `json:"name"`
	Header   string     `json:"header"`
	Icon     string     `json:"icon"`
	Auth     string     `json:"auth"`
	Target   string     `json:"target"`
	Children []seedMenu `json:"children"`
}

type seedRole struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Desc        string   `json:"desc"`
	DefaultRole int      `json:"default_role"`
	CreatedAt   string   `json:"created_at"`
	MenuList    []string `json:"menu_list"`
}

type seedUser struct {
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

// Seed 幂等写入演示数据；force=true 时重复执行会跳过已存在记录。
func Seed(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := seedMenus(tx); err != nil {
			return fmt.Errorf("seed menus: %w", err)
		}
		if err := seedRoles(tx); err != nil {
			return fmt.Errorf("seed roles: %w", err)
		}
		if err := seedUsers(tx); err != nil {
			return fmt.Errorf("seed users: %w", err)
		}
		if err := seedI18n(tx); err != nil {
			return fmt.Errorf("seed i18n: %w", err)
		}
		return nil
	})
}

func seedMenus(tx *gorm.DB) error {
	raw, err := seedFS.ReadFile("seed/menus.json")
	if err != nil {
		return err
	}
	var roots []seedMenu
	if err := json.Unmarshal(raw, &roots); err != nil {
		return err
	}

	var walk func(nodes []seedMenu, parentID uint64, depth int) error
	walk = func(nodes []seedMenu, parentID uint64, depth int) error {
		for i, n := range nodes {
			id, err := strconv.ParseUint(n.ID, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid menu id %q: %w", n.ID, err)
			}
			menu := model.Menu{
				ID:       id,
				ParentID: parentID,
				Title:    n.Title,
				Subtitle: n.Subtitle,
				Path:     n.Path,
				Name:     n.Name,
				Header:   n.Header,
				Icon:     n.Icon,
				Auth:     n.Auth,
				Target:   n.Target,
				Sort:     i,
				Type:     menuType(depth),
			}
			if err := tx.Create(&menu).Error; err != nil {
				return err
			}
			if len(n.Children) > 0 {
				if err := walk(n.Children, id, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(roots, 0, 0)
}

func menuType(depth int) string {
	switch depth {
	case 0:
		return "header"
	case 1:
		return "sider"
	default:
		return "page"
	}
}

func seedRoles(tx *gorm.DB) error {
	raw, err := seedFS.ReadFile("seed/roles.json")
	if err != nil {
		return err
	}
	var roles []seedRole
	if err := json.Unmarshal(raw, &roles); err != nil {
		return err
	}

	var menus []model.Menu
	if err := tx.Find(&menus).Error; err != nil {
		return err
	}
	menuByID := make(map[uint64]model.Menu, len(menus))
	for _, m := range menus {
		menuByID[m.ID] = m
	}

	for _, r := range roles {
		id, err := strconv.ParseUint(r.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid role id %q: %w", r.ID, err)
		}
		created := parseTime(r.CreatedAt)
		role := model.Role{
			ID:          id,
			Name:        r.Name,
			Desc:        r.Desc,
			DefaultRole: r.DefaultRole,
			CreatedAt:   created,
			UpdatedAt:   created,
		}
		if err := tx.Create(&role).Error; err != nil {
			return err
		}

		assign := make([]model.Menu, 0, len(r.MenuList))
		if r.Name == "ROLE_ADMIN" {
			assign = menus
		} else {
			for _, mid := range r.MenuList {
				if id, err := strconv.ParseUint(mid, 10, 64); err == nil {
					if m, ok := menuByID[id]; ok {
						assign = append(assign, m)
					}
				}
			}
		}
		if len(assign) > 0 {
			if err := tx.Model(&role).Association("Menus").Append(assign); err != nil {
				return err
			}
		}
	}
	return nil
}

func seedUsers(tx *gorm.DB) error {
	raw, err := seedFS.ReadFile("seed/users.json")
	if err != nil {
		return err
	}
	var users []seedUser
	if err := json.Unmarshal(raw, &users); err != nil {
		return err
	}

	var roles []model.Role
	if err := tx.Find(&roles).Error; err != nil {
		return err
	}
	roleByName := make(map[string]model.Role, len(roles))
	for _, r := range roles {
		roleByName[r.Name] = r
	}

	pwd, err := hash.Make(DefaultPassword)
	if err != nil {
		return err
	}
	adminPwd, err := hash.Make("admin")
	if err != nil {
		return err
	}

	usedAccounts := map[string]bool{"admin": true}
	for _, u := range users {
		id, err := strconv.ParseUint(u.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid user id %q: %w", u.ID, err)
		}
		created := parseTime(u.Date)
		account := u.Account
		if usedAccounts[account] {
			// 演示数据中存在同名账号，追加 ID 后缀保证唯一
			account = account + "-" + u.ID[len(u.ID)-3:]
		}
		usedAccounts[account] = true
		user := model.User{
			ID:        id,
			UUID:      "user-" + u.ID,
			Account:   account,
			Password:  pwd,
			Name:      u.Name,
			Avatar:    u.Avatar,
			Phone:     u.Phone,
			Mail:      u.Mail,
			Gender:    u.Gender,
			Status:    u.Status,
			Birthday:  u.Birthday,
			Region:    u.Region,
			Desc:      u.Desc,
			CreatedAt: created,
			UpdatedAt: created,
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		for _, code := range u.Role {
			if r, ok := roleByName[code]; ok {
				if err := tx.Model(&user).Association("Roles").Append(&r); err != nil {
					return err
				}
			}
		}
	}

	// 内置管理员：admin / admin
	admin := model.User{
		UUID:     "admin-uuid",
		Account:  "admin",
		Password: adminPwd,
		Name:     "Admin",
		Avatar:   "",
		Phone:    "18800008888",
		Mail:     "admin@admin-plus.dev",
		Gender:   1,
		Status:   1,
		Region:   []string{"北京市", "市辖区", "东城区"},
		Desc:     "内置超级管理员",
	}
	if err := tx.Create(&admin).Error; err != nil {
		return err
	}
	if r, ok := roleByName["ROLE_ADMIN"]; ok {
		if err := tx.Model(&admin).Association("Roles").Append(&r); err != nil {
			return err
		}
	}
	if r, ok := roleByName["ROLE_OPERATOR"]; ok {
		if err := tx.Model(&admin).Association("Roles").Append(&r); err != nil {
			return err
		}
	}
	return nil
}

func seedI18n(tx *gorm.DB) error {
	items := []model.I18nMessage{
		{Locale: "zh-CN", Module: "menu", Key: "dashboard", Value: "主控台"},
		{Locale: "zh-CN", Module: "menu", Key: "system", Value: "系统管理"},
		{Locale: "zh-CN", Module: "menu", Key: "form", Value: "表单页面"},
		{Locale: "zh-CN", Module: "menu", Key: "list", Value: "列表页面"},
		{Locale: "zh-CN", Module: "common", Key: "submit", Value: "提交"},
		{Locale: "zh-CN", Module: "common", Key: "cancel", Value: "取消"},
		{Locale: "en-US", Module: "menu", Key: "dashboard", Value: "Dashboard"},
		{Locale: "en-US", Module: "menu", Key: "system", Value: "System"},
		{Locale: "en-US", Module: "menu", Key: "form", Value: "Forms"},
		{Locale: "en-US", Module: "menu", Key: "list", Value: "Lists"},
		{Locale: "en-US", Module: "common", Key: "submit", Value: "Submit"},
		{Locale: "en-US", Module: "common", Key: "cancel", Value: "Cancel"},
	}
	return tx.Create(&items).Error
}

func parseTime(s string) time.Time {
	for _, layout := range []string{"2006-01-02 15:04:05", time.RFC3339, "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t
		}
	}
	return time.Now()
}
