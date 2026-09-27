package domain

import "testing"

func TestNewSearchPaginationClamping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		total          int
		pageParam      string
		pageSizeParam  string
		wantPage       int
		wantPageSize   int
		wantTotalPages int
	}{
		{"defaults", 45, "", "", 1, SearchPageSizeDefault, 3},
		{"mid page", 45, "2", "20", 2, 20, 3},
		{"page zero clamps to one", 45, "0", "20", 1, 20, 3},
		{"negative page clamps to one", 45, "-5", "20", 1, 20, 3},
		{"page beyond last clamps down", 45, "50", "20", 3, 20, 3},
		{"pageSize zero falls back to default", 45, "1", "0", 1, SearchPageSizeDefault, 3},
		{"pageSize above cap clamps to cap", 45, "1", "1000", 1, SearchPageSizeMax, 1},
		{"non-numeric page falls back", 45, "abc", "20", 1, 20, 3},
		{"empty result set still has one page", 0, "3", "20", 1, 20, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewSearchPagination(tt.total, tt.pageParam, tt.pageSizeParam)

			if got.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", got.Page, tt.wantPage)
			}

			if got.PageSize != tt.wantPageSize {
				t.Errorf("PageSize = %d, want %d", got.PageSize, tt.wantPageSize)
			}

			if got.TotalPages != tt.wantTotalPages {
				t.Errorf("TotalPages = %d, want %d", got.TotalPages, tt.wantTotalPages)
			}
		})
	}
}

func TestSearchPaginationWindow(t *testing.T) {
	t.Parallel()

	p := NewSearchPagination(45, "2", "20")

	if p.Offset() != 20 {
		t.Errorf("Offset = %d, want 20", p.Offset())
	}

	if p.End() != 40 {
		t.Errorf("End = %d, want 40", p.End())
	}

	if !p.HasPrev() || !p.HasNext() {
		t.Error("middle page should have both prev and next")
	}

	last := NewSearchPagination(45, "3", "20")

	if last.HasNext() {
		t.Error("last page should not have next")
	}

	first := NewSearchPagination(45, "1", "20")

	if first.HasPrev() {
		t.Error("first page should not have prev")
	}
}

func FuzzNewSearchPagination(f *testing.F) {
	f.Add(0, "", "")
	f.Add(1, "1", "10")
	f.Add(57, "-5", "99999999999999999999")
	f.Add(1000, "999999", "0")
	f.Add(3, "abc", "-2")

	f.Fuzz(func(t *testing.T, total int, pageParam, pageSizeParam string) {
		p := NewSearchPagination(total, pageParam, pageSizeParam)

		if p.Page < 1 {
			t.Fatalf("Page = %d, want >= 1", p.Page)
		}

		if p.TotalPages < 1 {
			t.Fatalf("TotalPages = %d, want >= 1", p.TotalPages)
		}

		if p.Page > p.TotalPages {
			t.Fatalf("Page = %d exceeds TotalPages = %d", p.Page, p.TotalPages)
		}

		if p.PageSize < 1 || p.PageSize > SearchPageSizeMax {
			t.Fatalf("PageSize = %d, want in [1, %d]", p.PageSize, SearchPageSizeMax)
		}

		if p.End() < p.Offset() {
			t.Fatalf("End = %d below Offset = %d", p.End(), p.Offset())
		}

		if p.End() > p.Total {
			t.Fatalf("End = %d exceeds Total = %d", p.End(), p.Total)
		}

		if p.End() < p.Offset()+p.PageSize && p.End() != p.Total {
			t.Fatalf(
				"window smaller than page size without hitting total: offset=%d end=%d total=%d",
				p.Offset(), p.End(), p.Total,
			)
		}
	})
}
