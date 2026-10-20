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

func newRoleSvc(db *gorm.DB) *service.RoleService {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return service.NewRoleService(repository.NewRoleRepository(db), repository.NewMenuRepository(db), repository.NewUserRepository(db), log)
}

func TestRoleCreateAndList(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	svc := newRoleSvc(db)

	dto, err := svc.Create(service.RoleUpsert{Name: "ROLE_TEST", Desc: "测试角色", MenuList: []string{"100", "101"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(dto.MenuList) != 2 {
		t.Fatalf("menu should be bound: %+v", dto)
	}

	list, total, err := svc.List(0, 10, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(list) != 3 {
		t.Fatalf("unexpected total %d", total)
	}
}

func TestRoleCreateDuplicate(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	_, err := newRoleSvc(db).Create(service.RoleUpsert{Name: "ROLE_ADMIN"})
	assertCode(t, err, errcode.RoleExists)
}

func TestRoleDeleteGuards(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	svc := newRoleSvc(db)

	// ROLE_ADMIN 不可删除
	assertCode(t, svc.Delete(1), errcode.Forbidden)

	// 正在被使用的角色不可删除：admin(1) 绑定了 ROLE_ADMIN
	dto, err := svc.Create(service.RoleUpsert{Name: "ROLE_INUSE"})
	if err != nil {
		t.Fatal(err)
	}
	id, _ := service.ParseID(dto.ID)
	if _, err := newUserSvc(db).Create(service.UserUpsert{
		Name: "绑定用户", Account: "binduser", Password: "123456", Role: []string{"ROLE_INUSE"},
	}); err != nil {
		t.Fatal(err)
	}
	assertCode(t, svc.Delete(id), errcode.RoleInUse)

	// 不存在的角色
	assertCode(t, svc.Delete(888888), errcode.NotFound)
}

func TestRoleUpdateMenus(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	svc := newRoleSvc(db)

	dto, err := svc.Create(service.RoleUpsert{Name: "ROLE_MENU", Desc: "菜单测试"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.UpdateMenus(dto.ID, []string{"100", "103", "104"}); err != nil {
		t.Fatalf("update menus: %v", err)
	}

	list, _, _ := svc.List(0, 10, "", "")
	for _, r := range list {
		if r.ID == dto.ID && len(r.MenuList) != 3 {
			t.Fatalf("expect 3 menus, got %v", r.MenuList)
		}
	}
}

func TestRoleUpdate(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	svc := newRoleSvc(db)

	dto, err := svc.Create(service.RoleUpsert{Name: "ROLE_EDIT", Desc: "旧"})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := svc.Update(service.RoleUpsert{ID: dto.ID, Name: "ROLE_EDIT2", Desc: "新"})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "ROLE_EDIT2" || updated.Desc != "新" {
		t.Fatalf("unexpected: %+v", updated)
	}

	// 重命名内置角色
	_, err = svc.Update(service.RoleUpsert{ID: "1", Name: "ROLE_RENAMED"})
	assertCode(t, err, errcode.Forbidden)
	_, err = svc.Update(service.RoleUpsert{ID: "999999", Name: "x"})
	assertCode(t, err, errcode.NotFound)
}
