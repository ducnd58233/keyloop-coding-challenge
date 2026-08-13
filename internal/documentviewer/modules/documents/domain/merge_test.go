package domain

import (
	"testing"
	"time"
)

func TestMergeEmptyInput(t *testing.T) {
	t.Parallel()
	if got := Merge(nil); len(got) != 0 {
		t.Fatalf("Merge(nil) = %d, want 0", len(got))
	}
	if got := Merge([]Document{}); len(got) != 0 {
		t.Fatalf("Merge(empty) = %d, want 0", len(got))
	}
}

func TestNamespacedID(t *testing.T) {
	t.Parallel()
	got := NamespacedID(SourceSales, "INV-1001")
	if got != "sales:INV-1001" {
		t.Fatalf("NamespacedID = %q, want sales:INV-1001", got)
	}
}

func TestNamespacedIDKeepsSameLocalIDDistinct(t *testing.T) {
	t.Parallel()
	sales := NamespacedID(SourceSales, "INV-1001")
	service := NamespacedID(SourceService, "INV-1001")
	if sales == service {
		t.Fatal("DD-8: same local id from two sources must not collide")
	}
	docs := []Document{
		{ID: sales, Source: SourceSales, IssuedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)},
		{ID: service, Source: SourceService, IssuedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)},
	}
	got := Merge(docs)
	if len(got) != 2 || got[0].ID == got[1].ID {
		t.Fatalf("merge = %+v, want two distinct namespaced ids", got)
	}
}

func TestMergeSortsIssuedAtThenSourceThenID(t *testing.T) {
	t.Parallel()
	t1 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	t0 := t1.Add(-time.Hour)
	docs := []Document{
		{ID: "sales:b", Source: SourceSales, IssuedAt: t1},
		{ID: "sales:a", Source: SourceSales, IssuedAt: t1},
		{ID: "service:z", Source: SourceService, IssuedAt: t1},
		{ID: "sales:old", Source: SourceSales, IssuedAt: t0},
	}
	got := Merge(docs)
	want := []string{"sales:a", "sales:b", "service:z", "sales:old"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i] {
			t.Fatalf("got[%d].ID = %q, want %q", i, got[i].ID, want[i])
		}
	}
}
