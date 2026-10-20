package errcode_test

import (
	"errors"
	"testing"

	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
)

func TestMessages(t *testing.T) {
	if errcode.OK.Message() != "成功" {
		t.Fatal("unexpected success message")
	}
	if int(errcode.LoginFailed) == 0 && errcode.LoginFailed.Message() == "" {
		t.Fatal("message should not be empty")
	}
	if errcode.Code(9999).Message() == "" {
		t.Fatal("unknown code should have fallback message")
	}
}

func TestAs(t *testing.T) {
	e := errcode.As(errcode.New(errcode.NotFound, "用户不存在"))
	if e.Code != errcode.NotFound || e.Msg != "用户不存在" {
		t.Fatalf("unexpected: %+v", e)
	}
	plain := errcode.As(errors.New("boom"))
	if plain.Code != errcode.Internal || plain.Msg != "boom" {
		t.Fatalf("unexpected: %+v", plain)
	}
	if errcode.As(nil) != nil {
		t.Fatal("nil should map to nil")
	}
}

func TestNewf(t *testing.T) {
	e := errcode.Newf(errcode.InvalidParams, "字段 %s 错误", "name")
	if e.Msg != "字段 name 错误" {
		t.Fatalf("got %q", e.Msg)
	}
	if e.Error() != e.Msg {
		t.Fatal("Error() should return message")
	}
}
