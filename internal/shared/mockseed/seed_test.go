package mockseed

import "testing"

func TestSuffix(t *testing.T) {
	if got := Suffix(WithDocumentsA); got != "2633" {
		t.Fatalf("Suffix(%s) = %q, want 2633", WithDocumentsA, got)
	}
	if got := Suffix("AB"); got != "AB" {
		t.Fatalf("Suffix(short) = %q, want AB", got)
	}
}

func TestAllHasAtLeast20(t *testing.T) {
	if len(All) < 20 {
		t.Fatalf("len(All) = %d, want >= 20", len(All))
	}
}

func TestAllVINsUniqueAndTenChars(t *testing.T) {
	seen := make(map[string]struct{}, len(All))
	for _, vin := range All {
		if len(vin.Value) != 10 {
			t.Fatalf("%q length = %d, want 10 (A1)", vin.Value, len(vin.Value))
		}
		if _, dup := seen[vin.Value]; dup {
			t.Fatalf("duplicate VIN %s", vin.Value)
		}
		seen[vin.Value] = struct{}{}
	}
}

func TestNamedKinds(t *testing.T) {
	cases := []struct {
		vin     string
		kind    Kind
		sales   bool
		service bool
	}{
		{WithDocumentsA, KindBoth, true, true},
		{WithDocumentsB, KindBoth, true, true},
		{WithNone, KindNone, false, false},
		{SalesOnly, KindSalesOnly, true, false},
		{ServiceOnly, KindServiceOnly, false, true},
	}
	for _, tc := range cases {
		found := false
		for _, vin := range All {
			if vin.Value != tc.vin {
				continue
			}
			found = true
			if vin.Kind != tc.kind {
				t.Fatalf("%s kind = %s, want %s", tc.vin, vin.Kind, tc.kind)
			}
			if HasSales(vin.Kind) != tc.sales || HasService(vin.Kind) != tc.service {
				t.Fatalf("%s HasSales=%v HasService=%v", tc.vin, HasSales(vin.Kind), HasService(vin.Kind))
			}
		}
		if !found {
			t.Fatalf("%s missing from All", tc.vin)
		}
	}
}
