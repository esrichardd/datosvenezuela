package store

import (
	"math"
	"slices"
	"testing"
)

func TestPaginate(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	tests := []struct {
		page Page
		want []int
	}{
		{Page{Limit: 2, Offset: 0}, []int{1, 2}},
		{Page{Limit: 2, Offset: 4}, []int{5}},
		{Page{Limit: 2, Offset: 5}, []int{}},
		{Page{Limit: 2, Offset: 99}, []int{}},
		{Page{Limit: 99, Offset: 3}, []int{4, 5}},
		{Page{Limit: 2, Offset: -1}, []int{1, 2}},
		// Without a positive limit there is nothing to return: a Page that
		// was never filled in must not select the whole list.
		{Page{}, []int{}},
		{Page{Limit: 0, Offset: 3}, []int{}},
		{Page{Limit: -1}, []int{}},
		// A huge limit must not overflow when added to the offset.
		{Page{Limit: math.MaxInt, Offset: 3}, []int{4, 5}},
	}

	for _, tt := range tests {
		got := Paginate(items, tt.page)
		if got == nil || !slices.Equal(got, tt.want) {
			t.Errorf("Paginate(%+v) = %v, want %v", tt.page, got, tt.want)
		}
	}
}

func TestPaginateReturnsCopy(t *testing.T) {
	items := []int{1, 2, 3}
	got := Paginate(items, Page{Limit: 2})
	got[0] = 99
	if items[0] != 1 {
		t.Error("modifying the page changed the original slice")
	}
}
