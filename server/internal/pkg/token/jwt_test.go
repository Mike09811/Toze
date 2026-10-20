package token_test

import (
	"testing"
	"time"

	"github.com/Mike09811/Toze/server/internal/pkg/token"
)

func TestGenerateAndParse(t *testing.T) {
	m := token.New("secret-key-123456", "issuer", time.Hour)
	raw, exp, err := m.Generate(7, "uuid-7", []string{"ROLE_ADMIN"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if time.Until(exp) <= 0 {
		t.Fatal("expire should be in the future")
	}
	claims, err := m.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.UserID != 7 || claims.UUID != "uuid-7" || len(claims.Roles) != 1 {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if claims.ID == "" {
		t.Fatal("jti should be set")
	}
}

func TestExpired(t *testing.T) {
	m := token.New("secret-key-123456", "issuer", -time.Minute)
	raw, _, err := m.Generate(1, "u", nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if _, err := m.Parse(raw); err != token.ErrExpired {
		t.Fatalf("want ErrExpired, got %v", err)
	}
}

func TestWrongSecret(t *testing.T) {
	a := token.New("secret-a", "issuer", time.Hour)
	b := token.New("secret-b", "issuer", time.Hour)
	raw, _, _ := a.Generate(1, "u", nil)
	if _, err := b.Parse(raw); err != token.ErrInvalid {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}

func TestGarbage(t *testing.T) {
	m := token.New("secret", "issuer", time.Hour)
	if _, err := m.Parse("not-a-token"); err != token.ErrInvalid {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}
