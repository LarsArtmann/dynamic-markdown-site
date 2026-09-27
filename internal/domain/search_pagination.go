package domain

import (
	"strconv"
)

const (
	// SearchPageSizeDefault is the number of results per page when the
	// client does not ask for a specific size.
	SearchPageSizeDefault = 20

	// SearchPageSizeMax caps the client-controlled page size.
	SearchPageSizeMax = 100
)

// SearchPagination describes which slice of the full search result set a
// page covers. Total is the size of the full result set; Page is 1-based and
// clamped into [1, TotalPages].
type SearchPagination struct {
	Page       int
	PageSize   int
	Total      int
	TotalPages int
}

// NewSearchPagination parses and clamps the page/pageSize query parameters
// against the total result count. Invalid or out-of-range values fall back
// to sane bounds instead of erroring.
func NewSearchPagination(total int, pageParam, pageSizeParam string) SearchPagination {
	pageSize := SearchPageSizeDefault
	if n, err := strconv.Atoi(pageSizeParam); err == nil {
		switch {
		case n > SearchPageSizeMax:
			pageSize = SearchPageSizeMax
		case n >= 1:
			pageSize = n
		}
	}

	totalPages := max((total+pageSize-1)/pageSize, 1)

	page := 1
	if n, err := strconv.Atoi(pageParam); err == nil {
		page = n
	}

	page = min(max(page, 1), totalPages)

	return SearchPagination{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}

// Offset is the zero-based index of the first result on the page.
func (p SearchPagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// End is the exclusive index of the last result on the page.
func (p SearchPagination) End() int {
	return min(p.Offset()+p.PageSize, p.Total)
}

// FirstResult is the 1-based ordinal of the first result on the page.
func (p SearchPagination) FirstResult() int {
	return p.Offset() + 1
}

// LastResult is the 1-based ordinal of the last result on the page.
func (p SearchPagination) LastResult() int {
	return p.End()
}

// HasPrev reports whether a previous page exists.
func (p SearchPagination) HasPrev() bool {
	return p.Page > 1
}

// HasNext reports whether a next page exists.
func (p SearchPagination) HasNext() bool {
	return p.Page < p.TotalPages
}
