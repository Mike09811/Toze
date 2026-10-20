package pagination_test

import (
	"testing"

	"github.com/Mike09811/Toze/server/internal/pkg/pagination"
)

func TestParseDefaults(t *testing.T) {
	p := pagination.Parse(-1, 0, "name", "weird")
	if p.Offset != 0 || p.Limit != pagination.DefaultLimit || p.SortType != "normal" {
		t.Fatalf("unexpected: %+v", p)
	}
}

func TestParseMaxLimit(t *testing.T) {
	p := pagination.Parse(10, 9999, "", "")
	if p.Limit != pagination.MaxLimit {
		t.Fatalf("limit should be capped, got %d", p.Limit)
	}
}

func TestOrder(t *testing.T) {
	allowed := map[string]string{"name": "name", "date": "created_at"}
	p := pagination.Parse(0, 10, "date", "desc")
	if got := p.Order(allowed, "id desc"); got != "created_at desc" {
		t.Fatalf("got %q", got)
	}
	p = pagination.Parse(0, 10, "password", "asc")
	if got := p.Order(allowed, "id desc"); got != "id desc" {
		t.Fatalf("unknown sort key should fallback, got %q", got)
	}
	p = pagination.Parse(0, 10, "name", "normal")
	if got := p.Order(allowed, "id desc"); got != "id desc" {
		t.Fatalf("normal sort should fallback, got %q", got)
	}
}
