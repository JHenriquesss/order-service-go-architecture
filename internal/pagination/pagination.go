// Package pagination provides the generic paginated result envelope shared by
// the list endpoints (architecture §17).
package pagination

// Page is the generic paginated result envelope.
type Page[T any] struct {
	Items      []T `json:"items"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
