package model

import "time"

// Menu 菜单节点，ParentID=0 为根节点。
type Menu struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	ParentID  uint64    `gorm:"index;default:0"`
	Title     string    `gorm:"size:64;not null"`
	Subtitle  string    `gorm:"size:64"`
	Path      string    `gorm:"size:255;index"`
	Name      string    `gorm:"size:64;index"`
	Header    string    `gorm:"size:64"`
	Icon      string    `gorm:"size:64"`
	Auth      string    `gorm:"size:255"`
	Target    string    `gorm:"size:16"`
	Type      string    `gorm:"size:16;default:page;comment:header/sider/page"`
	HideSider bool      `gorm:"default:false"`
	Divided   bool      `gorm:"default:false"`
	Sort      int       `gorm:"default:0"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time

	// Children 仅用于响应组装，不落库。
	Children []*Menu `gorm:"-" json:"children,omitempty"`
}

// TableName 表名。
func (Menu) TableName() string { return "sys_menus" }

// BuildTree 把扁平菜单组装成树，parentID=0 视为根。
func BuildTree(items []*Menu) []*Menu {
	byID := make(map[uint64]*Menu, len(items))
	for _, m := range items {
		m.Children = nil
		byID[m.ID] = m
	}
	roots := make([]*Menu, 0, 8)
	for _, m := range items {
		if m.ParentID == 0 || byID[m.ParentID] == nil {
			roots = append(roots, m)
			continue
		}
		p := byID[m.ParentID]
		p.Children = append(p.Children, m)
	}
	return roots
}
