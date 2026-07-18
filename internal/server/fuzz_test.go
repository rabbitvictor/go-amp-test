package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/rabbitvictor/go-amp-test/internal/domain"
)

// FuzzItemHandler_Create fuzzes the POST /items JSON body through the full
// Echo router backed by in-memory SQLite. The handler must never panic and
// must always return one of the documented status codes (201, 400, 500).
// When it accepts the input (201), the persisted name must round-trip
// byte-for-byte.
func FuzzItemHandler_Create(f *testing.F) {
	cases := []string{
		`{"name":"alpha"}`,
		`{"name":""}`,
		`{"name":null}`,
		`{}`,
		`null`,
		`42`,
		`{not json}`,
		`{"name":"\u0000"}`,
		`{"name":"` + strings.Repeat("a", 10000) + `"}`,
		`{"name":"\xff\xfe\xfd"}`,
		`{"name":"unicode \u00e9\u4e16\u754c"}`,
		`{"extra":"field","name":"ok"}`,
	}
	for _, tc := range cases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, body string) {
		srv, _ := newTestEcho(t)
		rec := do(t, srv, http.MethodPost, "/items", body)

		switch rec.Code {
		case http.StatusCreated, http.StatusBadRequest, http.StatusInternalServerError:
			// expected response codes
		default:
			t.Fatalf("unexpected status %d, body=%q", rec.Code, rec.Body.String())
		}

		if rec.Code != http.StatusCreated {
			return
		}
		var got domain.Item
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode created item: %v, body=%q", err, rec.Body.String())
		}
		var in domain.CreateItem
		_ = json.Unmarshal([]byte(body), &in)
		if in.Name != got.Name {
			t.Fatalf("name round-trip mismatch: sent %q, stored %q", in.Name, got.Name)
		}
	})
}

// FuzzItemHandler_Get fuzzes the :id path param of GET /items/:id. The id is
// url.PathEscape'd so every fuzz input produces a well-formed request path
// (httptest.NewRequest panics on malformed URLs) while still exercising
// strconv.ParseInt. The handler must return 200, 404, or 400 -- never 500
// and never panic.
func FuzzItemHandler_Get(f *testing.F) {
	cases := []string{
		"1", "9999", "0", "-1", "abc", " 1 ", "", "1.5",
		"9223372036854775807", "9223372036854775808",
		"0x10", "1e3", "+5", "\x00", "\xff\xfe",
	}
	for _, tc := range cases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, id string) {
		srv, _ := newTestEcho(t)
		// Seed an item so that some ids resolve to a real row.
		if rec := do(t, srv, http.MethodPost, "/items", `{"name":"seed"}`); rec.Code != http.StatusCreated {
			t.Fatalf("seed create status = %d, body=%s", rec.Code, rec.Body.String())
		}

		path := "/items/" + url.PathEscape(id)
		rec := do(t, srv, http.MethodGet, path, "")

		switch rec.Code {
		case http.StatusOK, http.StatusNotFound, http.StatusBadRequest:
			// expected response codes
		default:
			t.Fatalf("unexpected status %d for id %q, body=%q", rec.Code, id, rec.Body.String())
		}
	})
}
