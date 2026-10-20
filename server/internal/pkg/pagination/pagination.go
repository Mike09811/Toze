// Package pagination 处理分页与排序参数。
package pagination

const (
	DefaultLimit = 10
	MaxLimit     = 200
)

// Query 分页查询参数。
type Query struct {
	Offset   int
	Limit    int
	SortKey  string
	SortType string // normal | asc | desc
}

// Parse 归一化分页参数。
func Parse(offset, limit int, sortKey, sortType string) Query {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	switch sortType {
	case "asc", "desc":
	default:
		sortType = "normal"
	}
	return Query{Offset: offset, Limit: limit, SortKey: sortKey, SortType: sortType}
}

// Order 返回安全的排序表达式，sortKey 必须属于 allowed 白名单。
func (q Query) Order(allowed map[string]string, fallback string) string {
	if q.SortType == "normal" {
		return fallback
	}
	column, ok := allowed[q.SortKey]
	if !ok {
		return fallback
	}
	return column + " " + q.SortType
}
