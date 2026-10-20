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

func newUserSvc(db *gorm.DB) *service.UserService {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return service.NewUserService(repository.NewUserRepository(db), repository.NewRoleRepository(db), log)
}

func TestUserCreateAndList(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	svc := newUserSvc(db)

	dto, err := svc.Create(service.UserUpsert{
		Name: "张三", Account: "zhangsan", Password: "123456",
		Phone: "13800000000", Mail: "z@a.com", Gender: 1, Status: 1,
		Region: []string{"北京市"}, Role: []string{"ROLE_USER"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if dto.Account != "zhangsan" || dto.Role[0] != "ROLE_USER" {
		t.Fatalf("unexpected dto: %+v", dto)
	}

	list, total, err := svc.List(service.UserListQuery{Name: "张", Offset: 0, Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("unexpected list: total=%d len=%d", total, len(list))
	}

	_, _, err = svc.List(service.UserListQuery{Gender: 2})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUserCreateDuplicate(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	svc := newUserSvc(db)
	_, err := svc.Create(service.UserUpsert{Name: "重复", Account: "admin", Password: "123456"})
	assertCode(t, err, errcode.UserExists)
}

func TestUserCreateValidation(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	svc := newUserSvc(db)
	_, err := svc.Create(service.UserUpsert{Account: "abc", Password: "123456"})
	assertCode(t, err, errcode.InvalidParams)
	_, err = svc.Create(service.UserUpsert{Name: "abc", Account: "ab", Password: "123456"})
	assertCode(t, err, errcode.InvalidParams)
}

func TestUserUpdate(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	svc := newUserSvc(db)

	dto, err := svc.Create(service.UserUpsert{Name: "李四", Account: "lisi", Password: "123456", Role: []string{"ROLE_USER"}})
	if err != nil {
		t.Fatal(err)
	}
	dto2, err := svc.Update(service.UserUpsert{
		ID: dto.ID, Name: "李四改", Account: "lisi", Phone: "13900000000", Role: []string{"ROLE_ADMIN"},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if dto2.Name != "李四改" || dto2.Phone != "13900000000" || dto2.Role[0] != "ROLE_ADMIN" {
		t.Fatalf("unexpected: %+v", dto2)
	}

	_, err = svc.Update(service.UserUpsert{ID: "999999", Name: "x"})
	assertCode(t, err, errcode.NotFound)
}

func TestUserDeleteGuards(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	svc := newUserSvc(db)

	// 不能删除自己
	assertCode(t, svc.Delete(1, 1), errcode.Forbidden)
	// 不能删除内置 admin
	assertCode(t, svc.Delete(1, 999), errcode.Forbidden)
	// 不存在的用户
	assertCode(t, svc.Delete(12345, 999), errcode.NotFound)

	dto, err := svc.Create(service.UserUpsert{Name: "待删", Account: "todelete", Password: "123456"})
	if err != nil {
		t.Fatal(err)
	}
	id, _ := service.ParseID(dto.ID)
	if err := svc.Delete(id, 999); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, _, err := svc.List(service.UserListQuery{Account: "todelete"}); err != nil {
		t.Fatal(err)
	}
}

func TestUserDeleteMany(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	svc := newUserSvc(db)

	a, _ := svc.Create(service.UserUpsert{Name: "批量1", Account: "batch1", Password: "123456"})
	b, _ := svc.Create(service.UserUpsert{Name: "批量2", Account: "batch2", Password: "123456"})
	idA, _ := service.ParseID(a.ID)
	idB, _ := service.ParseID(b.ID)

	if err := svc.DeleteMany([]uint64{idA, idB}, 999); err != nil {
		t.Fatalf("delete many: %v", err)
	}
	list, total, _ := svc.List(service.UserListQuery{Account: "batch"})
	if total != 0 || len(list) != 0 {
		t.Fatalf("expect empty, got %d", total)
	}
	// 包含 admin 时应拒绝
	assertCode(t, svc.DeleteMany([]uint64{1}, 999), errcode.Forbidden)
}
