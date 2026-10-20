package database

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/model"
)

// Migrate 自动建表 / 升级表结构。
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.User{},
		&model.Role{},
		&model.Menu{},
		&model.OperationLog{},
		&model.I18nMessage{},
		&model.RevokedToken{},
		&model.PaasServer{},
		&model.PaasApp{},
		&model.PaasDomain{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}
