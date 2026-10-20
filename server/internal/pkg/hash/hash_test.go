package hash_test

import "testing"

import "github.com/Mike09811/Toze/server/internal/pkg/hash"

func TestMakeAndCheck(t *testing.T) {
	h, err := hash.Make("admin123")
	if err != nil {
		t.Fatalf("make: %v", err)
	}
	if h == "admin123" {
		t.Fatal("password should be hashed")
	}
	if !hash.Check(h, "admin123") {
		t.Fatal("check should pass")
	}
	if hash.Check(h, "wrong") {
		t.Fatal("wrong password should fail")
	}
}

func TestDifferentSalt(t *testing.T) {
	a, _ := hash.Make("same")
	b, _ := hash.Make("same")
	if a == b {
		t.Fatal("bcrypt should use random salt")
	}
}
