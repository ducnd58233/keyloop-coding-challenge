package http

import "testing"

func TestMapTypeClosedEnum(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"invoice":    "INVOICE",
		"WORK_ORDER": "WORK_ORDER",
		"mot":        "MOT",
		"weird-type": "OTHER",
		"":           "OTHER",
		"  quote  ":  "QUOTE",
	}
	for in, want := range cases {
		if got := MapType(in); got != want {
			t.Fatalf("MapType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveURLRelativeAndAbsolute(t *testing.T) {
	t.Parallel()
	base := "http://localhost:9100"
	rel := resolveURL(base, "/sales/v1/documents/INV-1001/raw")
	if rel != "http://localhost:9100/sales/v1/documents/INV-1001/raw" {
		t.Fatalf("relative = %q", rel)
	}
	abs := "http://localhost:9101/service/v1/attachments/WO-77/raw"
	if got := resolveURL(base, abs); got != abs {
		t.Fatalf("absolute rewritten: %q", got)
	}
	if got := resolveURL(base, ""); got != "" {
		t.Fatalf("empty = %q", got)
	}
}
