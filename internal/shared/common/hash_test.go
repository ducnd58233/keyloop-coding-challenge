package common

import (
	"strings"
	"testing"
)

func TestHashVINIsDeterministicAndRedacts(t *testing.T) {
	t.Parallel()
	const vin = "1HGCM82633"
	const salt = "unit-test-salt"

	h1, s1 := HashVIN(salt, vin)
	h2, s2 := HashVIN(salt, vin)
	if h1 == "" || h1 != h2 {
		t.Fatalf("hash not stable: %q vs %q", h1, h2)
	}
	if s1 != "2633" || s2 != "2633" {
		t.Fatalf("suffix = %q %q, want last 4", s1, s2)
	}
	if strings.Contains(strings.ToLower(h1), strings.ToLower(vin)) {
		t.Fatalf("hash %q contains VIN", h1)
	}
	if strings.Contains(h1, salt) {
		t.Fatalf("hash %q contains salt", h1)
	}
}

func TestHashVINSaltChangesDigest(t *testing.T) {
	t.Parallel()
	const vin = "1HGCM82633"
	a, _ := HashVIN("salt-a", vin)
	b, _ := HashVIN("salt-b", vin)
	if a == b {
		t.Fatal("different salts produced the same digest")
	}
}

func TestHashVINShortSuffix(t *testing.T) {
	t.Parallel()
	const vin = "AB"
	hash, suffix := HashVIN("unit-test-salt", vin)
	if suffix != "" {
		t.Fatalf("suffix = %q, want empty so short invalid VIN is not stored (SPEC §8)", suffix)
	}
	if hash == "" || hash == vin {
		t.Fatalf("hash = %q, want non-empty digest", hash)
	}
}
