package database

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/model"
)

// paasMenuSeed PaaS 模块菜单，固定 ID 便于幂等补插。
// 全量种子（seed/menus.json）已包含同样内容；这里解决的是
// 「数据库已有数据、Seed 整体跳过」的老库不会出现新菜单的问题。
var paasMenuSeed = []model.Menu{
	{ID: 1000061, ParentID: 1000001, Title: "服务器管理", Path: "/paas/server", Header: "home", Type: "sider", Sort: 0},
	{ID: 1000062, ParentID: 1000001, Title: "应用管理", Path: "/paas/app", Header: "home", Type: "sider", Sort: 1},
	{ID: 1000063, ParentID: 1000001, Title: "域名管理", Path: "/paas/domain", Header: "home", Type: "sider", Sort: 2},
}

// EnsurePaasMenus 幂等补插 PaaS 菜单并绑定到管理员角色。
// 每次启动执行：新库由全量种子创建后这里全部跳过；老库只补缺失的行。
func EnsurePaasMenus(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var admin model.Role
		hasAdmin := tx.Where("name = ?", "ROLE_ADMIN").First(&admin).Error == nil

		for _, m := range paasMenuSeed {
			var count int64
			if err := tx.Model(&model.Menu{}).Where("id = ?", m.ID).Count(&count).Error; err != nil {
				return fmt.Errorf("check paas menu %d: %w", m.ID, err)
			}
			if count > 0 {
				continue
			}
			// 父节点不存在时（老库被裁剪过）一并跳过，避免挂孤儿菜单
			if m.ParentID != 1000001 {
				var parent int64
				if err := tx.Model(&model.Menu{}).Where("id = ?", m.ParentID).Count(&parent).Error; err != nil {
					return err
				}
				if parent == 0 {
					continue
				}
			}
			if err := tx.Create(&m).Error; err != nil {
				return fmt.Errorf("create paas menu %d: %w", m.ID, err)
			}
			if hasAdmin {
				if err := tx.Model(&admin).Association("Menus").Append(&m); err != nil {
					return fmt.Errorf("bind paas menu %d to admin: %w", m.ID, err)
				}
			}
		}
		return nil
	})
}
