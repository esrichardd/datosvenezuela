package httpx

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/esrichardd/apivenezuela/api/internal/platform/store"
)

// Pagination limits shared by every list endpoint.
const (
	DefaultLimit = 100
	MaxLimit     = 500
)

// ParsePage reads the limit and offset query parameters. Missing values take
// defaultLimit and 0. It reports every invalid parameter, so the caller can
// answer with a single 400.
//
// An invalid parameter keeps its default, so the returned Page is always
// within the limits, even if the caller forgets to check the errors.
func ParsePage(r *http.Request, defaultLimit int) (store.Page, []InvalidParam) {
	q := r.URL.Query()
	p := store.Page{Limit: defaultLimit}
	var invalid []InvalidParam

	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > MaxLimit {
			invalid = append(invalid, InvalidParam{
				Name:   "limit",
				Reason: fmt.Sprintf("must be an integer between 1 and %d", MaxLimit),
			})
		} else {
			p.Limit = n
		}
	}

	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			invalid = append(invalid, InvalidParam{
				Name:   "offset",
				Reason: "must be an integer greater than or equal to 0",
			})
		} else {
			p.Offset = n
		}
	}

	return p, invalid
}
