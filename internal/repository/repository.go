// Package repository provides generic query helpers shared by domain repos.
package repository

import (
	"strings"
)

// ListOptions describes server-side pagination, sorting and filtering.
// All bounds are enforced so no query can materialize an unbounded result set.
type ListOptions struct {
	Limit  int
	Offset int
	Sort   string
	Order  string // "asc" or "desc"
}

// DefaultLimit is applied when a caller passes no limit.
const (
	DefaultLimit = 25
	MaxLimit     = 200
)

// ClampLimit returns a limit within [1, MaxLimit].
func ClampLimit(n int) int {
	if n <= 0 {
		return DefaultLimit
	}
	if n > MaxLimit {
		return MaxLimit
	}
	return n
}

// NormalizeOrder returns "asc" or "desc".
func NormalizeOrder(s string) string {
	if strings.EqualFold(s, "desc") {
		return "desc"
	}
	return "asc"
}

// Page is the standard list envelope.
type Page[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// NewPage wraps items in a Page, ensuring Items is never nil.
func NewPage[T any](items []T, total, limit, offset int) Page[T] {
	if items == nil {
		items = []T{}
	}
	return Page[T]{Items: items, Total: total, Limit: limit, Offset: offset}
}
