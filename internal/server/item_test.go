package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/rabbitvictor/go-amp-test/internal/domain"
)

func TestItemHandler_CreateAndGet(t *testing.T) {
	srv, _ := newTestEcho(t)

	rec := do(t, srv, http.MethodPost, "/items", `{"name":"alpha"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var created domain.Item
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.ID == 0 || created.Name != "alpha" {
		t.Errorf("created item = %+v", created)
	}

	rec = do(t, srv, http.MethodGet, "/items/1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var got domain.Item
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode got: %v", err)
	}
	if got.ID != created.ID || got.Name != created.Name {
		t.Errorf("get item = %+v, want %+v", got, created)
	}
}

func TestItemHandler_Create_ValidationErrors(t *testing.T) {
	srv, _ := newTestEcho(t)

	cases := []struct {
		name string
		body string
		want int
	}{
		{"empty name", `{"name":""}`, http.StatusBadRequest},
		{"malformed json", `{not json}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, srv, http.MethodPost, "/items", tc.body)
			if rec.Code != tc.want {
				t.Errorf("status = %d, want %d, body=%s", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestItemHandler_Get_InvalidID(t *testing.T) {
	srv, _ := newTestEcho(t)
	rec := do(t, srv, http.MethodGet, "/items/abc", "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestItemHandler_Get_NotFound(t *testing.T) {
	srv, _ := newTestEcho(t)
	rec := do(t, srv, http.MethodGet, "/items/9999", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestItemHandler_List(t *testing.T) {
	srv, _ := newTestEcho(t)

	for _, name := range []string{"one", "two"} {
		rec := do(t, srv, http.MethodPost, "/items", `{"name":"`+name+`"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create %q status = %d, body=%s", name, rec.Code, rec.Body.String())
		}
	}

	rec := do(t, srv, http.MethodGet, "/items", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		Items []domain.Item `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(out.Items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(out.Items))
	}
}
