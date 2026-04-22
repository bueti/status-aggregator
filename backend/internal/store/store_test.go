package store

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/bueti/status-aggregator/backend/internal/providers"
)

func open(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestCreateAutoAppendsSortOrder(t *testing.T) {
	ctx := context.Background()
	s := open(t)

	mk := func(id string) providers.Config {
		return providers.Config{
			ID:     id,
			Name:   id,
			Kind:   providers.KindStatuspageIO,
			Params: json.RawMessage(`{"base_url":"https://example.com"}`),
		}
	}

	// Three inserts with SortOrder=0 should be appended as 1, 2, 3.
	for _, id := range []string{"a", "b", "c"} {
		if err := s.Create(ctx, mk(id)); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}

	cfgs, err := s.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if got := len(cfgs); got != 3 {
		t.Fatalf("len(cfgs) = %d, want 3", got)
	}
	for i, want := range []struct {
		id    string
		order int
	}{{"a", 1}, {"b", 2}, {"c", 3}} {
		if cfgs[i].ID != want.id || cfgs[i].SortOrder != want.order {
			t.Errorf("cfgs[%d] = {%s, %d}, want {%s, %d}", i, cfgs[i].ID, cfgs[i].SortOrder, want.id, want.order)
		}
	}
}

func TestCreatePreservesExplicitSortOrder(t *testing.T) {
	ctx := context.Background()
	s := open(t)

	c := providers.Config{
		ID:        "pinned",
		Name:      "Pinned",
		Kind:      providers.KindStatuspageIO,
		SortOrder: 42,
		Params:    json.RawMessage(`{"base_url":"https://example.com"}`),
	}
	if err := s.Create(ctx, c); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := s.Get(ctx, "pinned")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.SortOrder != 42 {
		t.Errorf("SortOrder = %d, want 42", got.SortOrder)
	}

	// A subsequent auto-append should land after the pinned=42.
	next := providers.Config{
		ID:     "new",
		Name:   "New",
		Kind:   providers.KindStatuspageIO,
		Params: json.RawMessage(`{"base_url":"https://example.com"}`),
	}
	if err := s.Create(ctx, next); err != nil {
		t.Fatalf("create next: %v", err)
	}
	gotNext, err := s.Get(ctx, "new")
	if err != nil {
		t.Fatalf("get next: %v", err)
	}
	if gotNext.SortOrder != 43 {
		t.Errorf("next SortOrder = %d, want 43", gotNext.SortOrder)
	}
}
