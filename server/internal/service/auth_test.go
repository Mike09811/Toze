package service_test

import (
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/pkg/token"
	"github.com/Mike09811/Toze/server/internal/repository"
	"github.com/Mike09811/Toze/server/internal/service"
	"github.com/Mike09811/Toze/server/internal/testutil"
)

func newAuth(t *testing.T, db *gorm.DB) *service.AuthService {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	jwt := token.New("test-secret-key-0123456789", "test", time.Hour)
	return service.NewAuthService(
		repository.NewUserRepository(db),
		repository.NewRoleRepository(db),
		repository.NewTokenRepository(db),
		jwt, log,
	)
}

func assertCode(t *testing.T, err error, want errcode.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("want error code %d, got nil", want)
	}
	var e *errcode.Error
	if !errors.As(err, &e) {
		t.Fatalf("want errcode.Error, got %T: %v", err, err)
	}
	if e.Code != want {
		t.Fatalf("want code %d, got %d (%s)", want, e.Code, e.Msg)
	}
}

func TestLoginSuccess(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	auth := newAuth(t, db)

	res, err := auth.Login("admin", "admin")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if res.Token == "" || res.UUID != "admin-uuid" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if len(res.Info.Access) == 0 || res.Info.Access[0] != "ROLE_ADMIN" {
		t.Fatalf("unexpected access: %v", res.Info.Access)
	}
	if res.ExpiresIn <= 0 {
		t.Fatalf("unexpected expires_in: %d", res.ExpiresIn)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	_, err := newAuth(t, db).Login("admin", "wrong")
	assertCode(t, err, errcode.LoginFailed)
}

func TestLoginUnknownAccount(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	_, err := newAuth(t, db).Login("nobody", "admin")
	assertCode(t, err, errcode.LoginFailed)
}

func TestLoginDisabled(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	if err := db.Model(&model.User{}).Where("account = ?", "admin").Update("status", 3).Error; err != nil {
		t.Fatal(err)
	}
	_, err := newAuth(t, db).Login("admin", "admin")
	assertCode(t, err, errcode.UserDisabled)
}

func TestRegisterAndDuplicate(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	auth := newAuth(t, db)

	res, err := auth.Register("newuser", "secret123", "新用户")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if res.Info.Name != "新用户" {
		t.Fatalf("unexpected name: %s", res.Info.Name)
	}

	_, err = auth.Register("newuser", "secret123", "重复")
	assertCode(t, err, errcode.UserExists)

	_, err = auth.Register("ab", "secret123", "短账号")
	assertCode(t, err, errcode.InvalidParams)
}

func TestChangePassword(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	auth := newAuth(t, db)

	assertCode(t, auth.ChangePassword(1, "bad", "newpass1"), errcode.OldPasswordErr)
	if err := auth.ChangePassword(1, "admin", "newpass1"); err != nil {
		t.Fatalf("change: %v", err)
	}
	if _, err := auth.Login("admin", "newpass1"); err != nil {
		t.Fatalf("login with new password: %v", err)
	}
	assertCode(t, auth.ChangePassword(1, "admin", "123"), errcode.InvalidParams)
}

func TestProfile(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	p, err := newAuth(t, db).Profile(1)
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	if p.Account != "admin" || len(p.Access) == 0 {
		t.Fatalf("unexpected profile: %+v", p)
	}
	_, err = newAuth(t, db).Profile(999)
	assertCode(t, err, errcode.NotFound)
}
