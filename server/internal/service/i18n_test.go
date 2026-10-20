package service_test

import (
	"testing"

	"github.com/Mike09811/Toze/server/internal/database"
	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/repository"
	"github.com/Mike09811/Toze/server/internal/service"
	"github.com/Mike09811/Toze/server/internal/testutil"
)

func TestI18nMessages(t *testing.T) {
	db := testutil.NewDB(t)
	items := []model.I18nMessage{
		{Locale: "zh-CN", Module: "menu", Key: "dashboard", Value: "主控台"},
		{Locale: "en-US", Module: "menu", Key: "dashboard", Value: "Dashboard"},
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatal(err)
	}
	svc := service.NewI18nService(repository.NewI18nRepository(db))
	msgs, err := svc.Messages()
	if err != nil {
		t.Fatalf("messages: %v", err)
	}
	if msgs["zh-CN"]["menu"]["dashboard"] != "主控台" {
		t.Fatalf("unexpected: %+v", msgs)
	}
	if msgs["en-US"]["menu"]["dashboard"] != "Dashboard" {
		t.Fatalf("unexpected: %+v", msgs)
	}
}

func TestSeedIsIdempotent(t *testing.T) {
	db, err := database.OpenTest("seed_idempotent")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Seed(db); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if err := database.Seed(db); err != nil {
		t.Fatalf("second seed should be skipped: %v", err)
	}
	var users int64
	db.Model(&model.User{}).Count(&users)
	if users < 10 {
		t.Fatalf("expect seed users, got %d", users)
	}
}
