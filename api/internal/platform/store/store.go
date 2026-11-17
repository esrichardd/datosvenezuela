// Package store holds the vocabulary shared by the handlers and the data
// sources behind them: the page of a list and the not-found error.
//
// Every data source (the CSV files loaded in memory today, a database
// tomorrow) speaks these terms, so the handlers never depend on where the
// data comes from.
package store

import "errors"

// ErrNotFound is returned by every data source when a code does not exist.
// A database source translates its own error (such as sql.ErrNoRows) into
// this one.
var ErrNotFound = errors.New("not found")

// Page selects a window of a list: at most Limit items, skipping the first
// Offset. A Limit of zero or less selects no items.
type Page struct {
	Limit  int
	Offset int
}

// Paginate returns a copy of the window of items selected by p, so callers
// cannot modify the source. It never returns nil. It is meant for data held
// in memory; a database applies the page in its query instead.
//
// A Limit of zero or less selects no items: a Page that was never filled in
// yields an empty list, not the whole dataset.
func Paginate[T any](items []T, p Page) []T {
	start := min(max(p.Offset, 0), len(items))
	// Clamp the size to what is left, so a huge Limit cannot overflow.
	size := min(max(p.Limit, 0), len(items)-start)
	return append(make([]T, 0, size), items[start:start+size]...)
}
