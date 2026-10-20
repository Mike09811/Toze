package service_test

import (
	"io"
	"log/slog"
	"testing"

	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/repository"
	"github.com/Mike09811/Toze/server/internal/service"
	"github.com/Mike09811/Toze/server/internal/testutil"
)

func newMenuSvc(db *gorm.DB) *service.MenuService {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return service.NewMenuService(repository.NewMenuRepository(db), log)
}

func TestMenuTree(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	res, err := newMenuSvc(db).Tree()
	if err != nil {
		t.Fatalf("tree: %v", err)
	}
	if len(res.List) != 1 {
		t.Fatalf("expect 1 root, got %d", len(res.List))
	}
	root := res.List[0]
	if len(root.Children) != 2 {
		t.Fatalf("expect 2 children, got %d", len(root.Children))
	}
	if len(res.Header) != 1 || len(res.Sider) != 2 {
		t.Fatalf("unexpected views: header=%d sider=%d", len(res.Header), len(res.Sider))
	}
}

func TestMenuCreateAndUpdate(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	svc := newMenuSvc(db)

	dto, err := svc.Create(service.MenuUpsert{Title: "新页面", Path: "/new", ParentID: "100"}, "sider")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if dto.Type != "sider" {
		t.Fatalf("unexpected type: %s", dto.Type)
	}

	updated, err := svc.Update(service.MenuUpsert{ID: dto.ID, Title: "改名", Path: "/new2"})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Title != "改名" || updated.Path != "/new2" {
		t.Fatalf("unexpected: %+v", updated)
	}

	_, err = svc.Create(service.MenuUpsert{Path: "/x"}, "header")
	assertCode(t, err, errcode.InvalidParams)
	_, err = svc.Create(service.MenuUpsert{Title: "x"}, "header")
	assertCode(t, err, errcode.InvalidParams)
}

func TestMenuDeleteGuards(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	svc := newMenuSvc(db)

	// 有子菜单
	assertCode(t, svc.Delete(100), errcode.MenuHasChild)
	// 被角色引用
	assertCode(t, svc.Delete(104), errcode.MenuHasRole)
	// 不存在
	assertCode(t, svc.Delete(999999), errcode.NotFound)
}

func TestMenuDeleteUnreferenced(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	svc := newMenuSvc(db)

	dto, err := svc.Create(service.MenuUpsert{Title: "临时", Path: "/tmp-menu"}, "header")
	if err != nil {
		t.Fatal(err)
	}
	id, _ := service.ParseID(dto.ID)
	if err := svc.Delete(id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.Tree(); err != nil {
		t.Fatal(err)
	}
}
