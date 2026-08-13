package domain

import "strings"

// Format is the single A1 rule: length and alphabet. ISO 3779 is a constant
// change here (SPEC §11.1), not a rewrite of callers.
type Format struct {
	Length   int
	Alphabet string
}

// DefaultFormat matches DRAFT A1 (10 uppercase alphanumeric).
var DefaultFormat = Format{
	Length:   10,
	Alphabet: "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
}

// Valid is A1: exact length and alphabet, not a prefix match.
func (f Format) Valid(vin string) bool {
	if f.Length <= 0 || f.Alphabet == "" {
		return false
	}
	n := 0
	for _, r := range vin {
		n++
		if n > f.Length {
			return false
		}
		if !strings.ContainsRune(f.Alphabet, r) {
			return false
		}
	}
	return n == f.Length
}
