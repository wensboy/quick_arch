package model

// NewPage 依据请求的 base/count、过滤后的 total 与实际数据构造分页结果.
func NewPage[T any](base, count, total int64, items []T) Page[T] {
	if items == nil {
		items = []T{}
	}
	return Page[T]{
		Total:      total,
		Base:       base,
		Count:      count,
		ExactCount: int64(len(items)),
		Items:      items,
	}
}

// Page 是通用分页结果.
type Page[T any] struct {
	Total      int64 `json:"total"`       // 过滤条件下的总条数
	Base       int64 `json:"base"`        // 起始位置
	Count      int64 `json:"count"`       // 请求的条数
	ExactCount int64 `json:"exact_count"` // 实际返回的条数
	Items      []T   `json:"items"`       // 数据
}

// Offset 返回 SQL 偏移量.
func (p Page[T]) Offset() int64 { return p.Base }

// Limit 返回 SQL 限制条数.
func (p Page[T]) Limit() int64 { return p.Count }

// Empty 表示本次未取到数据.
func (p Page[T]) Empty() bool { return len(p.Items) == 0 }

// HasMore 表示按当前位置仍有后续数据.
func (p Page[T]) HasMore() bool { return p.Base+int64(len(p.Items)) < p.Total }
