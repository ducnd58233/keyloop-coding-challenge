package domain

import "testing"

func TestDefaultFormatA1(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		vin  string
		ok   bool
	}{
		{name: "ten uppercase alphanumeric", vin: "1HGCM82633", ok: true},
		{name: "nine rejected", vin: "1HGCM8263", ok: false},
		{name: "eleven rejected", vin: "1HGCM826331", ok: false},
		{name: "lowercase rejected", vin: "1hgcm82633", ok: false},
		{name: "empty rejected", vin: "", ok: false},
		{name: "whitespace padded rejected", vin: "1HGCM8263 ", ok: false},
		{name: "non-ascii rejected", vin: "1HGCM8263É", ok: false},
		{name: "punctuation rejected", vin: "1HGCM-2633", ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := DefaultFormat.Valid(tc.vin); got != tc.ok {
				t.Fatalf("Valid(%q) = %v, want %v", tc.vin, got, tc.ok)
			}
		})
	}
}

func TestFormatISO3779IsConstantChange(t *testing.T) {
	t.Parallel()
	iso := Format{Length: 17, Alphabet: "ABCDEFGHJKLMNPRSTUVWXYZ0123456789"}
	if iso.Valid("1HGCM82633") {
		t.Fatal("10-char A1 VIN must fail the 17-char ISO rule")
	}
	if !iso.Valid("1HGCM82633A000001") {
		t.Fatal("17-char VIN without I/O/Q should pass ISO rule")
	}
	if iso.Valid("1HGCM82633I000001") {
		t.Fatal("ISO alphabet excludes I")
	}
}
